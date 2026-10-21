package csl

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Namespace is the CSL XML namespace.
const Namespace = "http://purl.org/net/xbiblio/csl"

const xmlNamespace = "http://www.w3.org/XML/1998/namespace"

// maxDepth bounds element nesting so that malicious input cannot exhaust the
// stack in the recursive conversion. Real styles nest fewer than 20 levels.
const maxDepth = 256

// node is an XML element with its position. Parsing first builds a tree of
// nodes and then converts it to the typed AST, which keeps the conversion
// independent of encoding/xml's streaming API.
type node struct {
	name  string // local name
	space string // namespace URI
	attrs []xml.Attr
	kids  []*node
	text  strings.Builder
	pos   Pos
}

func readTree(data []byte) (*node, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	var root *node
	var stack []*node
	for {
		// Before Token is called, InputPos is the position of the next
		// token's first character, i.e. the "<" of a start tag.
		line, col := d.InputPos()
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			var se *xml.SyntaxError
			if errors.As(err, &se) {
				return nil, ErrorList{{Line: se.Line, Msg: "XML syntax error: " + se.Msg}}
			}
			return nil, ErrorList{{Line: line, Col: col, Msg: err.Error()}}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if len(stack) >= maxDepth {
				return nil, ErrorList{{Line: line, Col: col, Element: t.Name.Local,
					Msg: fmt.Sprintf("elements nested deeper than %d levels", maxDepth)}}
			}
			n := &node{name: t.Name.Local, space: t.Name.Space, attrs: t.Attr, pos: Pos{line, col}}
			if len(stack) == 0 {
				root = n
			} else {
				p := stack[len(stack)-1]
				p.kids = append(p.kids, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write(t)
			}
		}
	}
	if root == nil {
		return nil, ErrorList{{Msg: "document has no root element"}}
	}
	return root, nil
}

// parser converts nodes to AST values and collects diagnostics.
type parser struct {
	errs  ErrorList
	warns []*ParseError
}

func (p *parser) errorf(n *node, format string, args ...any) {
	p.errs = append(p.errs, &ParseError{Line: n.pos.Line, Col: n.pos.Col, Element: n.name, Msg: fmt.Sprintf(format, args...)})
}

func (p *parser) warnf(n *node, format string, args ...any) {
	p.warns = append(p.warns, &ParseError{Line: n.pos.Line, Col: n.pos.Col, Element: n.name, Msg: fmt.Sprintf(format, args...)})
}

// isCSL reports whether n is in the CSL namespace. Elements without a
// namespace are accepted too, since some hand-written styles omit it.
func (n *node) isCSL() bool { return n.space == Namespace || n.space == "" }

// cslKids returns the CSL-namespace children of n.
func (n *node) cslKids() []*node {
	kids := n.kids[:0:0]
	for _, k := range n.kids {
		if k.isCSL() {
			kids = append(kids, k)
		}
	}
	return kids
}

// attrs gives typed access to an element's attributes and remembers which
// ones were read, so that finish can warn about the rest.
type attrs struct {
	p    *parser
	n    *node
	used []bool
}

func (p *parser) attrs(n *node) *attrs {
	return &attrs{p: p, n: n, used: make([]bool, len(n.attrs))}
}

func (a *attrs) get(name string) (string, bool) {
	space := ""
	if local, ok := strings.CutPrefix(name, "xml:"); ok {
		space, name = xmlNamespace, local
	}
	for i, at := range a.n.attrs {
		if at.Name.Local == name && (at.Name.Space == space || space == xmlNamespace && at.Name.Space == "xml") {
			a.used[i] = true
			return at.Value, true
		}
	}
	return "", false
}

func (a *attrs) has(name string) bool {
	_, ok := a.get(name)
	return ok
}

func (a *attrs) str(name string) string {
	v, _ := a.get(name)
	return v
}

func (a *attrs) strPtr(name string) *string {
	if v, ok := a.get(name); ok {
		return &v
	}
	return nil
}

func (a *attrs) required(name string) string {
	v, ok := a.get(name)
	if !ok {
		a.p.errorf(a.n, "missing required attribute %q", name)
	}
	return v
}

func (a *attrs) boolPtr(name string) *bool {
	v, ok := a.get(name)
	if !ok {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1":
		b := true
		return &b
	case "false", "0":
		b := false
		return &b
	}
	a.p.errorf(a.n, "attribute %q: %q is not a boolean", name, v)
	return nil
}

func (a *attrs) boolean(name string, def bool) bool {
	if b := a.boolPtr(name); b != nil {
		return *b
	}
	return def
}

func (a *attrs) intPtr(name string) *int {
	v, ok := a.get(name)
	if !ok {
		return nil
	}
	i, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || i < 0 {
		a.p.errorf(a.n, "attribute %q: %q is not a non-negative integer", name, v)
		return nil
	}
	return &i
}

func (a *attrs) integer(name string, def int) int {
	if i := a.intPtr(name); i != nil {
		return *i
	}
	return def
}

// enum returns the attribute value if it is one of allowed, def if the
// attribute is absent, and records an error otherwise.
func (a *attrs) enum(name, def string, allowed ...string) string {
	v, ok := a.get(name)
	if !ok {
		return def
	}
	for _, x := range allowed {
		if v == x {
			return v
		}
	}
	a.p.errorf(a.n, "attribute %q: invalid value %q (want one of %s)", name, v, strings.Join(allowed, ", "))
	return def
}

func (a *attrs) list(name string) []string {
	v, ok := a.get(name)
	if !ok {
		return nil
	}
	return strings.Fields(v)
}

// finish warns about attributes that were never read. Namespace declarations
// and attributes in foreign namespaces are allowed.
func (a *attrs) finish() {
	for i, at := range a.n.attrs {
		if a.used[i] || at.Name.Space == "xmlns" || at.Name.Space == "" && at.Name.Local == "xmlns" {
			continue
		}
		if at.Name.Space != "" && at.Name.Space != Namespace {
			continue
		}
		a.p.warnf(a.n, "unknown attribute %q ignored", at.Name.Local)
	}
}

// Shared attribute groups.

func (a *attrs) affixes() Affixes {
	return Affixes{Prefix: a.str("prefix"), Suffix: a.str("suffix")}
}

func (a *attrs) formatting() Formatting {
	return Formatting{
		FontStyle:      FontStyle(a.enum("font-style", "", "normal", "italic", "oblique")),
		FontVariant:    FontVariant(a.enum("font-variant", "", "normal", "small-caps")),
		FontWeight:     FontWeight(a.enum("font-weight", "", "normal", "bold", "light")),
		TextDecoration: TextDecoration(a.enum("text-decoration", "", "none", "underline")),
		VerticalAlign:  VerticalAlign(a.enum("vertical-align", "", "baseline", "sup", "sub")),
	}
}

func (a *attrs) display() Display {
	return Display(a.enum("display", "", "block", "left-margin", "right-inline", "indent"))
}

func (a *attrs) textCase() TextCase {
	return TextCase(a.enum("text-case", "", "lowercase", "uppercase", "capitalize-first",
		"capitalize-all", "title", "sentence"))
}

var termForms = []string{"long", "short", "verb", "verb-short", "symbol"}

// nameOptions reads the cs:name attributes. With inheritable set it reads the
// cs:style/cs:citation/cs:bibliography spelling, where form and delimiter are
// called name-form and name-delimiter.
func (a *attrs) nameOptions(inheritable bool) NameOptions {
	formAttr, delimAttr := "form", "delimiter"
	if inheritable {
		formAttr, delimAttr = "name-form", "name-delimiter"
	}
	delimiterRules := []string{"contextual", "always", "never", "after-inverted-name"}
	return NameOptions{
		And:                    NameAnd(a.enum("and", "", "text", "symbol")),
		DelimiterPrecedesEtAl:  DelimiterRule(a.enum("delimiter-precedes-et-al", "", delimiterRules...)),
		DelimiterPrecedesLast:  DelimiterRule(a.enum("delimiter-precedes-last", "", delimiterRules...)),
		EtAlMin:                a.intPtr("et-al-min"),
		EtAlUseFirst:           a.intPtr("et-al-use-first"),
		EtAlSubsequentMin:      a.intPtr("et-al-subsequent-min"),
		EtAlSubsequentUseFirst: a.intPtr("et-al-subsequent-use-first"),
		EtAlUseLast:            a.boolPtr("et-al-use-last"),
		Initialize:             a.boolPtr("initialize"),
		InitializeWith:         a.strPtr("initialize-with"),
		NameAsSortOrder:        NameAsSortOrder(a.enum("name-as-sort-order", "", "first", "all")),
		SortSeparator:          a.strPtr("sort-separator"),
		Form:                   NameForm(a.enum(formAttr, "", "long", "short", "count")),
		Delimiter:              a.strPtr(delimAttr),
	}
}

func (a *attrs) inheritableNameOptions() InheritableNameOptions {
	return InheritableNameOptions{NameOptions: a.nameOptions(true), NamesDelimiter: a.strPtr("names-delimiter")}
}
