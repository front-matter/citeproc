package csl

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

var (
	bundledStyles  = []string{"apa", "modern-language-association", "vancouver"}
	bundledLocales = []string{"en-US", "de-DE", "es-ES"}
)

func readStyle(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "styles", name+".csl"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readLocale(t testing.TB, tag string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "locales", "locales-"+tag+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustParse(t *testing.T, data []byte) *Style {
	t.Helper()
	s, err := ParseStyle(data)
	if err != nil {
		t.Fatalf("ParseStyle: %v", err)
	}
	return s
}

func TestParseBundledStyles(t *testing.T) {
	for _, name := range bundledStyles {
		t.Run(name, func(t *testing.T) {
			s := mustParse(t, readStyle(t, name))
			for _, w := range s.Warnings {
				t.Errorf("warning: %v", w)
			}
			if s.Citation == nil || s.Bibliography == nil {
				t.Errorf("missing citation or bibliography")
			}
		})
	}
}

func TestParseBundledLocales(t *testing.T) {
	for _, tag := range bundledLocales {
		t.Run(tag, func(t *testing.T) {
			l, err := ParseLocale(readLocale(t, tag))
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range l.Warnings {
				t.Errorf("warning: %v", w)
			}
			if l.Lang != tag || len(l.Dates) != 2 || len(l.Terms) < 300 {
				t.Errorf("Lang %q, %d dates, %d terms", l.Lang, len(l.Dates), len(l.Terms))
			}
		})
	}
}

func TestGolden(t *testing.T) {
	tests := []struct {
		golden string
		parse  func(t *testing.T) any
	}{
		{"apa.golden", func(t *testing.T) any { return mustParse(t, readStyle(t, "apa")) }},
		{"locales-en-US.golden", func(t *testing.T) any {
			l, err := ParseLocale(readLocale(t, "en-US"))
			if err != nil {
				t.Fatal(err)
			}
			return l
		}},
	}
	for _, tt := range tests {
		t.Run(tt.golden, func(t *testing.T) {
			var got bytes.Buffer
			if err := Dump(&got, tt.parse(t)); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", tt.golden)
			if *update {
				if err := os.WriteFile(path, got.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run with -update to create)", err)
			}
			if !bytes.Equal(got.Bytes(), want) {
				t.Errorf("AST dump differs from %s; run go test ./internal/csl -run TestGolden -update and review the diff", path)
			}
		})
	}
}

// minimal wraps body in a style whose first body line is line 4.
func minimal(body string) []byte {
	return []byte(`<?xml version="1.0" encoding="utf-8"?>
<style xmlns="http://purl.org/net/xbiblio/csl" class="in-text" version="1.0">
  <info><title>T</title><id>t</id><updated>2020-01-01T00:00:00+00:00</updated></info>
` + body + `
</style>`)
}

const layout = `<citation><layout><text variable="title"/></layout></citation>`

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want []string // "line:col: <elem>: message" substrings, one per error
	}{
		{"undefined macro", minimal(`<citation><layout>
  <text macro="nope"/></layout></citation>`), []string{`5:3: <text>: call to undefined macro "nope"`}},
		{"undefined sort macro", minimal(`<citation><sort><key macro="nope"/></sort><layout><text value="x"/></layout></citation>`),
			[]string{`4:17: <key>: call to undefined macro "nope"`}},
		{"macro cycle", minimal(`<macro name="a"><text macro="b"/></macro>
<macro name="b"><group><text macro="a"/></group></macro>
` + layout), []string{`5:24: <text>: macro cycle: a → b → a`}},
		{"self-recursive macro", minimal(`<macro name="a"><text macro="a"/></macro>` + layout),
			[]string{`4:17: <text>: macro cycle: a → a`}},
		{"duplicate macro", minimal(`<macro name="a"><text value="1"/></macro>
<macro name="a"><text value="2"/></macro>` + layout), []string{`5:1: <macro>: duplicate macro "a" (first defined at line 4)`}},
		{"missing layout", minimal(`<citation/>`), []string{`4:1: <citation>: missing required <layout>`}},
		{"invalid enum", minimal(`<citation><layout><text variable="title" font-style="slanted"/></layout></citation>`),
			[]string{`4:19: <text>: attribute "font-style": invalid value "slanted"`}},
		{"invalid integer", minimal(`<citation et-al-min="many">` + `<layout><text value="x"/></layout></citation>`),
			[]string{`4:1: <citation>: attribute "et-al-min": "many" is not a non-negative integer`}},
		{"text with two sources", minimal(`<citation><layout><text variable="title" value="x"/></layout></citation>`),
			[]string{`4:19: <text>: exactly one of "variable", "macro", "term" and "value" is required`}},
		{"choose without if", minimal(`<citation><layout><choose><else><text value="x"/></else></choose></layout></citation>`),
			[]string{`<else> must be the last child of <choose>, after <if>`, `4:19: <choose>: <choose> needs an <if>`}},
		{"if without condition", minimal(`<citation><layout><choose><if><text value="x"/></if></choose></layout></citation>`),
			[]string{`4:27: <if>: <if> needs at least one condition`}},
		{"names without variable", minimal(`<citation><layout><names/></layout></citation>`),
			[]string{`4:19: <names>: missing required attribute "variable"`}},
		{"missing class", []byte(`<style xmlns="http://purl.org/net/xbiblio/csl" version="1.0">` + layout + `</style>`),
			[]string{`1:1: <style>: missing required attribute "class"`}},
		{"no citation", minimal(``), []string{`style has neither <citation> nor an independent-parent link`}},
		{"wrong root", []byte(`<locale xmlns="http://purl.org/net/xbiblio/csl"/>`), []string{`root element must be <style>`}},
		{"syntax error", []byte("<style>\n<citation>\n</style>"), []string{`3: XML syntax error`}},
		{"empty", []byte(""), []string{`document has no root element`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseStyle(tt.data)
			if err == nil {
				t.Fatalf("ParseStyle succeeded: %+v", s)
			}
			var list ErrorList
			if !errors.As(err, &list) {
				t.Fatalf("error %T is not an ErrorList", err)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("errors.As(*ParseError) failed")
			}
			if len(list) != len(tt.want) {
				t.Fatalf("got %d errors, want %d:\n%v", len(list), len(tt.want), err)
			}
			for i, w := range tt.want {
				if !strings.Contains(list[i].Error(), w) {
					t.Errorf("error %d = %q, want it to contain %q", i, list[i].Error(), w)
				}
			}
		})
	}
}

func TestParseWarnings(t *testing.T) {
	s := mustParse(t, minimal(`<citation><layout>
  <text variable="title" bogus="1"/>
  <frob/>
  <text term="nonsense"/>
  <text variable="tittle"/>
  <text variable="issued"/>
  <choose><if type="boook" locator="pgae" position="last"><text value="x"/></if></choose>
  <date variable="issued" form="text"><date-part name="year" suffix="."/></date>
</layout></citation>`))
	want := []string{
		`5:3: <text>: unknown attribute "bogus" ignored`,
		`6:3: <frob>: unknown element ignored`,
		`7:3: <text>: unknown term "nonsense"`,
		`8:3: <text>: unknown variable "tittle"`,
		`9:3: <text>: date variable "issued" cannot be rendered with <text>`,
		`10:11: <if>: unknown locator type "pgae"`,
		`10:11: <if>: unknown position "last"`,
		`10:11: <if>: unknown item type "boook"`,
		`11:39: <date-part>: affixes on a <date-part> of a localized date are ignored`,
	}
	if len(s.Warnings) != len(want) {
		t.Fatalf("got %d warnings, want %d: %v", len(s.Warnings), len(want), s.Warnings)
	}
	for i, w := range want {
		if got := s.Warnings[i].Error(); got != w {
			t.Errorf("warning %d = %q, want %q", i, got, w)
		}
	}
}

func TestTermContent(t *testing.T) {
	s := mustParse(t, minimal(`<locale xml:lang="en"><terms>
  <term name="bc"> B.C.E.</term>
  <term name="page"><single>page</single><multiple>pages</multiple></term>
  <term name="editor" form="short"><single>ed.</single></term>
  <term name="and"/>
  <term name="ordinal-01" gender-form="feminine" match="whole-number">re</term>
  <term name="edition" gender="feminine">édition</term>
</terms></locale>`+layout))
	if len(s.Locales) != 1 || s.Locales[0].Lang != "en" {
		t.Fatalf("locales: %+v", s.Locales)
	}
	terms := s.Locales[0].Terms
	want := []Term{
		{Name: "bc", Form: FormLong, Single: " B.C.E.", Multiple: " B.C.E."},
		{Name: "page", Form: FormLong, Single: "page", Multiple: "pages"},
		{Name: "editor", Form: FormShort, Single: "ed.", Multiple: "ed."},
		{Name: "and", Form: FormLong},
		{Name: "ordinal-01", Form: FormLong, Single: "re", Multiple: "re", GenderForm: Feminine, Match: MatchWholeNumber},
		{Name: "edition", Form: FormLong, Single: "édition", Multiple: "édition", Gender: Feminine},
	}
	if len(terms) != len(want) {
		t.Fatalf("got %d terms", len(terms))
	}
	for i, w := range want {
		got := *terms[i]
		got.Pos = Pos{}
		if got != w {
			t.Errorf("term %d = %+v, want %+v", i, got, w)
		}
	}
}

func TestNameOptionsCascade(t *testing.T) {
	s := mustParse(t, readStyle(t, "apa"))

	cite := s.CitationNameOptions()
	if *cite.EtAlMin != 3 || *cite.EtAlUseFirst != 1 || cite.EtAlUseLast != nil {
		t.Errorf("citation scope: et-al-min %v, use-first %v, use-last %v", *cite.EtAlMin, *cite.EtAlUseFirst, cite.EtAlUseLast)
	}
	bib := s.BibliographyNameOptions()
	if *bib.EtAlMin != 21 || *bib.EtAlUseFirst != 19 || !*bib.EtAlUseLast {
		t.Errorf("bibliography scope: et-al-min %v, use-first %v, use-last %v", *bib.EtAlMin, *bib.EtAlUseFirst, *bib.EtAlUseLast)
	}

	// The first cs:names in APA's "author-bib" macro sets its own name
	// attributes and inherits et-al from the bibliography.
	var names *Names
	Walk(s.Macros["author-bib"].Children, func(el Element) bool {
		if n, ok := el.(*Names); ok && names == nil && n.Name != nil {
			names = n
		}
		return true
	})
	if names == nil {
		t.Fatal("no cs:names with cs:name in macro author-bib")
	}
	got := names.NameOptionsIn(bib)
	if got.And != AndSymbol || *got.InitializeWith != ". " || got.NameAsSortOrder != SortOrderAll ||
		got.DelimiterPrecedesLast != DelimiterAlways || *got.EtAlMin != 21 || !*got.EtAlUseLast {
		t.Errorf("resolved cs:name options: %+v", got)
	}
}

func TestInherit(t *testing.T) {
	one, two := 1, 2
	comma, semi := ", ", "; "
	parent := InheritableNameOptions{
		NameOptions:    NameOptions{And: AndText, EtAlMin: &one, Delimiter: &comma, Form: NameShort},
		NamesDelimiter: &semi,
	}
	child := InheritableNameOptions{NameOptions: NameOptions{EtAlMin: &two, Form: NameLong}}
	got := child.Inherit(parent)
	if got.And != AndText || *got.EtAlMin != 2 || *got.Delimiter != ", " || got.Form != NameLong || *got.NamesDelimiter != "; " {
		t.Errorf("Inherit = %+v", got)
	}

	n := &Names{}
	if d := n.DelimiterIn(got); d != "; " {
		t.Errorf("DelimiterIn without cs:names delimiter = %q, want names-delimiter", d)
	}
	n.Delimiter = &comma
	if d := n.DelimiterIn(got); d != ", " {
		t.Errorf("DelimiterIn = %q, want the cs:names delimiter", d)
	}
}

func TestDependentStyle(t *testing.T) {
	s := mustParse(t, []byte(`<style xmlns="http://purl.org/net/xbiblio/csl" version="1.0" default-locale="en-GB">
  <info><title>D</title><id>d</id>
    <link href="http://www.zotero.org/styles/apa" rel="independent-parent"/>
    <updated>2020-01-01T00:00:00+00:00</updated></info>
</style>`))
	if !s.IsDependent() || s.IndependentParent() != "http://www.zotero.org/styles/apa" || s.DefaultLocale != "en-GB" {
		t.Errorf("dependent style: %+v", s)
	}
}

// TestCorpus parses external corpora when their directories are given in
// environment variables. It is skipped otherwise. See testdata/README.md
// for the pinned commits.
//
//	CITEPROC_TEST_SUITE   test-suite processor-tests/humans directory
//	CITEPROC_CSL_STYLES   styles repository (top level and dependent/)
//	CITEPROC_CSL_LOCALES  locales repository
func TestCorpus(t *testing.T) {
	cslSection := regexp.MustCompile(`(?s)>>=+ CSL =+>>\r?\n(.*?)\r?\n<<=+ CSL =+<<`)
	corpora := []struct {
		env, glob string
		extract   func([]byte) []byte
		parse     func([]byte) ([]*ParseError, error)
	}{
		{"CITEPROC_TEST_SUITE", "*.txt", func(b []byte) []byte {
			if m := cslSection.FindSubmatch(b); m != nil {
				return m[1]
			}
			return nil
		}, parseStyleWarnings},
		{"CITEPROC_CSL_STYLES", "*.csl", nil, parseStyleWarnings},
		{"CITEPROC_CSL_STYLES", "dependent/*.csl", nil, parseStyleWarnings},
		{"CITEPROC_CSL_LOCALES", "locales-*.xml", nil, func(b []byte) ([]*ParseError, error) {
			l, err := ParseLocale(b)
			if err != nil {
				return nil, err
			}
			return l.Warnings, nil
		}},
	}
	for _, c := range corpora {
		t.Run(c.env+"/"+c.glob, func(t *testing.T) {
			dir := os.Getenv(c.env)
			if dir == "" {
				t.Skipf("%s not set", c.env)
			}
			files, err := filepath.Glob(filepath.Join(dir, c.glob))
			if err != nil || len(files) == 0 {
				t.Fatalf("no files match %s in %s", c.glob, dir)
			}
			for _, f := range files {
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				if c.extract != nil {
					if data = c.extract(data); data == nil {
						t.Errorf("%s: no CSL section", filepath.Base(f))
						continue
					}
				}
				warns, err := c.parse(data)
				if err != nil {
					t.Errorf("%s: %v", filepath.Base(f), err)
				}
				for _, w := range warns {
					t.Errorf("%s: warning: %v", filepath.Base(f), w)
				}
			}
			t.Logf("%d files", len(files))
		})
	}
}

func parseStyleWarnings(b []byte) ([]*ParseError, error) {
	s, err := ParseStyle(b)
	if err != nil {
		return nil, err
	}
	return s.Warnings, nil
}
