package csl

import (
	"strings"
)

// ParseStyle parses and validates a CSL style.
//
// Fatal problems (malformed XML, missing required elements or attributes,
// invalid attribute values, undefined or duplicate macros, macro cycles) are
// returned together as an ErrorList and the style is nil. Non-fatal problems
// (unknown elements, attributes, terms or variables) are recorded in
// Style.Warnings.
func ParseStyle(data []byte) (*Style, error) {
	root, err := readTree(data)
	if err != nil {
		return nil, err
	}
	p := &parser{}
	if root.name != "style" {
		p.errorf(root, "root element must be <style>")
		return nil, p.errs
	}
	s := p.style(root)
	if len(p.errs) > 0 {
		return nil, p.errs
	}
	s.Warnings = p.warns
	return s, nil
}

// ParseLocale parses a CSL locale file. Errors and warnings are reported as
// for ParseStyle; warnings go to Locale.Warnings.
func ParseLocale(data []byte) (*Locale, error) {
	root, err := readTree(data)
	if err != nil {
		return nil, err
	}
	p := &parser{}
	if root.name != "locale" {
		p.errorf(root, "root element must be <locale>")
		return nil, p.errs
	}
	if root.space != Namespace {
		p.warnf(root, "root element is not in the CSL namespace %q", Namespace)
	}
	l := p.locale(root, true)
	if len(p.errs) > 0 {
		return nil, p.errs
	}
	l.Warnings = p.warns
	return l, nil
}

func (p *parser) style(n *node) *Style {
	if n.space != Namespace {
		p.warnf(n, "root element is not in the CSL namespace %q", Namespace)
	}
	a := p.attrs(n)
	s := &Style{
		Pos:                       n.pos,
		Class:                     StyleClass(a.enum("class", "", "in-text", "note")),
		Version:                   a.str("version"),
		DefaultLocale:             a.str("default-locale"),
		DemoteNonDroppingParticle: DemoteParticle(a.enum("demote-non-dropping-particle", "display-and-sort", "never", "sort-only", "display-and-sort")),
		InitializeWithHyphen:      a.boolean("initialize-with-hyphen", true),
		PageRangeFormat: PageRangeFormat(a.enum("page-range-format", "",
			"expanded", "minimal", "minimal-two", "chicago", "chicago-15", "chicago-16")),
		NameOptions: a.inheritableNameOptions(),
		Macros:      map[string]*Macro{},
	}
	a.finish()

	seenInfo := false
	for _, k := range n.cslKids() {
		switch k.name {
		case "info":
			if seenInfo {
				p.errorf(k, "duplicate <info>")
			}
			seenInfo = true
			s.Info = p.info(k)
		case "locale":
			s.Locales = append(s.Locales, p.locale(k, false))
		case "macro":
			m := p.macro(k)
			if prev, dup := s.Macros[m.Name]; dup {
				p.errorf(k, "duplicate macro %q (first defined at line %d)", m.Name, prev.Line)
				continue
			}
			s.Macros[m.Name] = m
			s.MacroOrder = append(s.MacroOrder, m.Name)
		case "citation":
			if s.Citation != nil {
				p.errorf(k, "duplicate <citation>")
				continue
			}
			s.Citation = p.citation(k)
		case "bibliography":
			if s.Bibliography != nil {
				p.errorf(k, "duplicate <bibliography>")
				continue
			}
			s.Bibliography = p.bibliography(k)
		default:
			p.warnf(k, "unknown element ignored")
		}
	}

	switch {
	case s.Citation != nil:
		if s.Class == "" {
			p.errorf(n, "missing required attribute %q", "class")
		}
	case s.IndependentParent() == "":
		p.errorf(n, "style has neither <citation> nor an independent-parent link")
	}
	p.checkMacros(s)
	return s
}

func (p *parser) info(n *node) Info {
	var info Info
	for _, k := range n.cslKids() {
		text := strings.TrimSpace(k.text.String())
		switch k.name {
		case "title":
			info.Title = text
		case "title-short":
			info.TitleShort = text
		case "id":
			info.ID = text
		case "author":
			info.Authors = append(info.Authors, person(k))
		case "contributor":
			info.Contributors = append(info.Contributors, person(k))
		case "category":
			a := p.attrs(k)
			if v, ok := a.get("citation-format"); ok {
				info.CitationFormat = v
			}
			if v, ok := a.get("field"); ok {
				info.Fields = append(info.Fields, v)
			}
		case "issn":
			info.ISSN = append(info.ISSN, text)
		case "eissn":
			info.EISSN = text
		case "issnl":
			info.ISSNL = text
		case "link":
			a := p.attrs(k)
			info.Links = append(info.Links, Link{Href: a.str("href"), Rel: a.str("rel"), Lang: a.str("xml:lang")})
		case "published":
			info.Published = text
		case "rights":
			info.Rights = text
			info.RightsLicense = p.attrs(k).str("license")
		case "summary":
			info.Summary = text
		case "updated":
			info.Updated = text
		}
		// Unknown metadata is ignored silently: it cannot affect rendering.
	}
	return info
}

func person(n *node) Person {
	var pr Person
	for _, k := range n.cslKids() {
		text := strings.TrimSpace(k.text.String())
		switch k.name {
		case "name":
			pr.Name = text
		case "email":
			pr.Email = text
		case "uri":
			pr.URI = text
		}
	}
	return pr
}

func (p *parser) locale(n *node, file bool) *Locale {
	a := p.attrs(n)
	l := &Locale{Pos: n.pos}
	if file {
		l.Lang = a.required("xml:lang")
		l.Version = a.str("version")
	} else {
		l.Lang = a.str("xml:lang")
	}
	a.finish()

	for _, k := range n.cslKids() {
		switch k.name {
		case "info":
			for _, ik := range k.cslKids() {
				switch ik.name {
				case "translator":
					l.Translators = append(l.Translators, person(ik))
				case "rights":
					l.Rights = strings.TrimSpace(ik.text.String())
				case "updated":
					l.Updated = strings.TrimSpace(ik.text.String())
				}
			}
		case "style-options":
			oa := p.attrs(k)
			l.StyleOptions = LocaleOptions{
				LimitDayOrdinalsToDay1: oa.boolPtr("limit-day-ordinals-to-day-1"),
				PunctuationInQuote:     oa.boolPtr("punctuation-in-quote"),
			}
			oa.finish()
		case "date":
			l.Dates = append(l.Dates, p.date(k, true))
		case "terms":
			for _, tk := range k.cslKids() {
				if tk.name != "term" {
					p.warnf(tk, "unknown element ignored")
					continue
				}
				l.Terms = append(l.Terms, p.term(tk))
			}
		default:
			p.warnf(k, "unknown element ignored")
		}
	}
	return l
}

func (p *parser) term(n *node) *Term {
	a := p.attrs(n)
	t := &Term{
		Pos:        n.pos,
		Name:       a.required("name"),
		Form:       TermForm(a.enum("form", "long", termForms...)),
		Gender:     Gender(a.enum("gender", "", "masculine", "feminine")),
		GenderForm: Gender(a.enum("gender-form", "", "masculine", "feminine")),
		Match:      OrdinalMatch(a.enum("match", "", "last-digit", "last-two-digits", "whole-number")),
	}
	a.finish()
	if t.Name != "" && !IsTerm(t.Name) {
		p.warnf(n, "unknown term %q", t.Name)
	}

	var single, multiple *string
	for _, k := range n.cslKids() {
		v := k.text.String()
		switch k.name {
		case "single":
			single = &v
		case "multiple":
			multiple = &v
		default:
			p.warnf(k, "unknown element ignored")
		}
	}
	switch {
	case single == nil && multiple == nil:
		// Plain-text term. Leading and trailing spaces are significant
		// (e.g. <term name="bc"> B.C.E.</term>), so the text is not trimmed.
		t.Single = n.text.String()
		t.Multiple = t.Single
	case single == nil:
		t.Single, t.Multiple = *multiple, *multiple
	case multiple == nil:
		t.Single, t.Multiple = *single, *single
	default:
		t.Single, t.Multiple = *single, *multiple
	}
	return t
}

func (p *parser) macro(n *node) *Macro {
	a := p.attrs(n)
	m := &Macro{Pos: n.pos, Name: a.required("name")}
	a.finish()
	m.Children = p.elements(n)
	if len(m.Children) == 0 {
		p.warnf(n, "macro %q is empty", m.Name)
	}
	return m
}

func (p *parser) citation(n *node) *Citation {
	a := p.attrs(n)
	c := &Citation{
		Pos:                       n.pos,
		DisambiguateAddNames:      a.boolean("disambiguate-add-names", false),
		DisambiguateAddGivenname:  a.boolean("disambiguate-add-givenname", false),
		DisambiguateAddYearSuffix: a.boolean("disambiguate-add-year-suffix", false),
		GivennameDisambiguationRule: GivennameRule(a.enum("givenname-disambiguation-rule", "by-cite",
			"all-names", "all-names-with-initials", "primary-name", "primary-name-with-initials", "by-cite")),
		Collapse:               Collapse(a.enum("collapse", "", "citation-number", "year", "year-suffix", "year-suffix-ranged")),
		CiteGroupDelimiter:     a.strPtr("cite-group-delimiter"),
		YearSuffixDelimiter:    a.strPtr("year-suffix-delimiter"),
		AfterCollapseDelimiter: a.strPtr("after-collapse-delimiter"),
		NearNoteDistance:       a.integer("near-note-distance", 5),
		NameOptions:            a.inheritableNameOptions(),
	}
	a.finish()
	c.Sort, c.Layout = p.sortAndLayout(n, true)
	return c
}

func (p *parser) bibliography(n *node) *Bibliography {
	a := p.attrs(n)
	b := &Bibliography{
		Pos:                        n.pos,
		HangingIndent:              a.boolean("hanging-indent", false),
		SecondFieldAlign:           SecondFieldAlign(a.enum("second-field-align", "", "flush", "margin")),
		LineSpacing:                a.integer("line-spacing", 1),
		EntrySpacing:               a.integer("entry-spacing", 1),
		SubsequentAuthorSubstitute: a.strPtr("subsequent-author-substitute"),
		SubsequentAuthorSubstituteRule: SubstituteRule(a.enum("subsequent-author-substitute-rule", "complete-all",
			"complete-all", "complete-each", "partial-each", "partial-first")),
		NameOptions: a.inheritableNameOptions(),
	}
	a.finish()
	if b.LineSpacing < 1 {
		p.errorf(n, "attribute %q must be at least 1", "line-spacing")
	}
	b.Sort, b.Layout = p.sortAndLayout(n, false)
	return b
}

func (p *parser) sortAndLayout(n *node, citation bool) (*Sort, *Layout) {
	var sort *Sort
	var layout *Layout
	for _, k := range n.cslKids() {
		switch k.name {
		case "sort":
			if sort != nil {
				p.errorf(k, "duplicate <sort>")
				continue
			}
			sort = p.sort(k)
		case "layout":
			if layout != nil {
				p.errorf(k, "duplicate <layout>")
				continue
			}
			layout = p.layout(k, citation)
		default:
			p.warnf(k, "unknown element ignored")
		}
	}
	if layout == nil {
		p.errorf(n, "missing required <layout>")
		layout = &Layout{}
	}
	return sort, layout
}

func (p *parser) layout(n *node, citation bool) *Layout {
	a := p.attrs(n)
	l := &Layout{Pos: n.pos, Affixes: a.affixes(), Formatting: a.formatting()}
	if citation {
		l.Delimiter = a.str("delimiter")
	}
	a.finish()
	l.Children = p.elements(n)
	return l
}

func (p *parser) sort(n *node) *Sort {
	s := &Sort{Pos: n.pos}
	for _, k := range n.cslKids() {
		if k.name != "key" {
			p.warnf(k, "unknown element ignored")
			continue
		}
		a := p.attrs(k)
		key := &SortKey{
			Pos:           k.pos,
			Variable:      a.str("variable"),
			Macro:         a.str("macro"),
			Descending:    a.enum("sort", "ascending", "ascending", "descending") == "descending",
			NamesMin:      a.intPtr("names-min"),
			NamesUseFirst: a.intPtr("names-use-first"),
			NamesUseLast:  a.boolPtr("names-use-last"),
		}
		a.finish()
		switch {
		case (key.Variable == "") == (key.Macro == ""):
			p.errorf(k, "exactly one of %q and %q is required", "variable", "macro")
		case key.Variable != "":
			p.checkVariables(k, "variable", key.Variable)
		}
		s.Keys = append(s.Keys, key)
	}
	if len(s.Keys) == 0 {
		p.errorf(n, "<sort> needs at least one <key>")
	}
	return s
}

// elements converts the rendering-element children of n.
func (p *parser) elements(n *node) []Element {
	var els []Element
	for _, k := range n.cslKids() {
		if el := p.element(k); el != nil {
			els = append(els, el)
		}
	}
	return els
}

func (p *parser) element(n *node) Element {
	switch n.name {
	case "text":
		return p.text(n)
	case "number":
		return p.number(n)
	case "label":
		return p.label(n, false)
	case "date":
		return p.date(n, false)
	case "names":
		return p.names(n)
	case "group":
		return p.group(n)
	case "choose":
		return p.choose(n)
	}
	p.warnf(n, "unknown element ignored")
	return nil
}

func (p *parser) text(n *node) *Text {
	a := p.attrs(n)
	t := &Text{Pos: n.pos}
	sources := 0
	if v, ok := a.get("variable"); ok {
		t.Source, t.Name = TextVariable, v
		t.Form = TermForm(a.enum("form", "long", "long", "short"))
		sources++
		switch VariableKindOf(v) {
		case DateVariable, NameVariable:
			p.warnf(n, "%s variable %q cannot be rendered with <text>", VariableKindOf(v), v)
		case UnknownVariable:
			p.warnf(n, "unknown variable %q", v)
		}
	}
	if v, ok := a.get("macro"); ok {
		t.Source, t.Name = TextMacro, v
		sources++
	}
	if v, ok := a.get("term"); ok {
		t.Source, t.Name = TextTerm, v
		t.Form = TermForm(a.enum("form", "long", termForms...))
		t.Plural = a.boolean("plural", false)
		sources++
		if !IsTerm(v) {
			p.warnf(n, "unknown term %q", v)
		}
	}
	if v, ok := a.get("value"); ok {
		t.Source, t.Value = TextValue, v
		sources++
	}
	if sources != 1 {
		p.errorf(n, "exactly one of %q, %q, %q and %q is required", "variable", "macro", "term", "value")
	}
	t.Affixes = a.affixes()
	t.Formatting = a.formatting()
	t.Display = a.display()
	t.Quotes = a.boolean("quotes", false)
	t.StripPeriods = a.boolean("strip-periods", false)
	t.TextCase = a.textCase()
	a.finish()
	p.noKids(n)
	return t
}

func (p *parser) number(n *node) *Number {
	a := p.attrs(n)
	num := &Number{
		Pos:        n.pos,
		Variable:   a.required("variable"),
		Form:       NumberForm(a.enum("form", "numeric", "numeric", "ordinal", "long-ordinal", "roman")),
		Affixes:    a.affixes(),
		Formatting: a.formatting(),
		Display:    a.display(),
		TextCase:   a.textCase(),
	}
	a.finish()
	if k := VariableKindOf(num.Variable); num.Variable != "" && k != NumberVariable {
		if k == UnknownVariable {
			p.warnf(n, "unknown variable %q", num.Variable)
		} else {
			p.warnf(n, "%s variable %q cannot be rendered with <number>", k, num.Variable)
		}
	}
	p.noKids(n)
	return num
}

func (p *parser) label(n *node, inNames bool) *Label {
	a := p.attrs(n)
	l := &Label{Pos: n.pos}
	if inNames {
		l.Form = TermForm(a.enum("form", "long", termForms...))
	} else {
		l.Variable = a.required("variable")
		l.Form = TermForm(a.enum("form", "long", "long", "short", "symbol"))
		if k := VariableKindOf(l.Variable); l.Variable != "" && k != NumberVariable {
			p.warnf(n, "label variable %q is not a number variable", l.Variable)
		}
	}
	l.Plural = LabelPlural(a.enum("plural", "contextual", "contextual", "always", "never"))
	l.Affixes = a.affixes()
	l.Formatting = a.formatting()
	l.StripPeriods = a.boolean("strip-periods", false)
	l.TextCase = a.textCase()
	a.finish()
	p.noKids(n)
	return l
}

var datePartForms = map[DatePartName][]string{
	PartDay:   {"numeric", "numeric-leading-zeros", "ordinal"},
	PartMonth: {"long", "short", "numeric", "numeric-leading-zeros"},
	PartYear:  {"long", "short"},
}

// date converts a cs:date element of a style (inLocale false) or a locale
// date format (inLocale true).
func (p *parser) date(n *node, inLocale bool) *Date {
	a := p.attrs(n)
	d := &Date{Pos: n.pos}
	if inLocale {
		d.Form = DateForm(a.enum("form", "", "text", "numeric"))
		if d.Form == "" {
			p.errorf(n, "missing required attribute %q", "form")
		}
		d.Delimiter = a.str("delimiter")
	} else {
		d.Variable = a.required("variable")
		if k := VariableKindOf(d.Variable); d.Variable != "" && k != DateVariable {
			p.warnf(n, "<date> variable %q is not a date variable", d.Variable)
		}
		d.Form = DateForm(a.enum("form", "", "text", "numeric"))
		if d.IsLocalized() {
			d.DateParts = DatePartsSel(a.enum("date-parts", "year-month-day", "year-month-day", "year-month", "year"))
		} else {
			d.Delimiter = a.str("delimiter")
		}
		d.Affixes = a.affixes()
		d.Display = a.display()
	}
	d.Formatting = a.formatting()
	d.TextCase = a.textCase()
	a.finish()

	localizedOverride := !inLocale && d.IsLocalized()
	for _, k := range n.cslKids() {
		if k.name != "date-part" {
			p.warnf(k, "unknown element ignored")
			continue
		}
		d.Parts = append(d.Parts, p.datePart(k, localizedOverride))
	}
	if !d.IsLocalized() && len(d.Parts) == 0 {
		p.errorf(n, "a non-localized <date> needs at least one <date-part>")
	}
	if inLocale && len(d.Parts) == 0 {
		p.errorf(n, "a localized date format needs at least one <date-part>")
	}
	return d
}

func (p *parser) datePart(n *node, localizedOverride bool) *DatePart {
	a := p.attrs(n)
	dp := &DatePart{Pos: n.pos, Name: DatePartName(a.enum("name", "", "day", "month", "year"))}
	if dp.Name == "" && !a.has("name") {
		p.errorf(n, "missing required attribute %q", "name")
	}
	if forms, ok := datePartForms[dp.Name]; ok {
		dp.Form = a.enum("form", "", forms...)
	}
	dp.RangeDelimiter = a.strPtr("range-delimiter")
	dp.Affixes = a.affixes()
	if localizedOverride && dp.Affixes != (Affixes{}) {
		p.warnf(n, "affixes on a <date-part> of a localized date are ignored")
	}
	dp.Formatting = a.formatting()
	dp.TextCase = a.textCase()
	if dp.Name == PartMonth {
		dp.StripPeriods = a.boolean("strip-periods", false)
	}
	a.finish()
	p.noKids(n)
	return dp
}

func (p *parser) names(n *node) *Names {
	a := p.attrs(n)
	ns := &Names{
		Pos:        n.pos,
		Variables:  a.list("variable"),
		Delimiter:  a.strPtr("delimiter"),
		Affixes:    a.affixes(),
		Formatting: a.formatting(),
		Display:    a.display(),
	}
	a.finish()
	if len(ns.Variables) == 0 {
		p.errorf(n, "missing required attribute %q", "variable")
	}
	for _, v := range ns.Variables {
		if k := VariableKindOf(v); k != NameVariable {
			p.warnf(n, "<names> variable %q is not a name variable", v)
		}
	}

	nameIdx, labelIdx := -1, -1
	for i, k := range n.cslKids() {
		switch k.name {
		case "name":
			if ns.Name != nil {
				p.errorf(k, "duplicate <name>")
				continue
			}
			ns.Name, nameIdx = p.name(k), i
		case "et-al":
			if ns.EtAl != nil {
				p.errorf(k, "duplicate <et-al>")
				continue
			}
			ea := p.attrs(k)
			ns.EtAl = &EtAl{Pos: k.pos, Term: ea.enum("term", "et-al", "et-al", "and others"), Formatting: ea.formatting()}
			ea.finish()
		case "label":
			if ns.Label != nil {
				p.errorf(k, "duplicate <label>")
				continue
			}
			ns.Label, labelIdx = p.label(k, true), i
		case "substitute":
			if ns.Substitute != nil {
				p.errorf(k, "duplicate <substitute>")
				continue
			}
			ns.Substitute = &Substitute{Pos: k.pos, Children: p.elements(k)}
			if len(ns.Substitute.Children) == 0 {
				p.warnf(k, "<substitute> is empty")
			}
		default:
			p.warnf(k, "unknown element ignored")
		}
	}
	ns.LabelBeforeName = labelIdx >= 0 && nameIdx >= 0 && labelIdx < nameIdx
	return ns
}

func (p *parser) name(n *node) *Name {
	a := p.attrs(n)
	nm := &Name{Pos: n.pos, Options: a.nameOptions(false), Affixes: a.affixes(), Formatting: a.formatting()}
	a.finish()
	for _, k := range n.cslKids() {
		if k.name != "name-part" {
			p.warnf(k, "unknown element ignored")
			continue
		}
		pa := p.attrs(k)
		part := &NamePart{
			Pos:        k.pos,
			Name:       pa.enum("name", "", "given", "family"),
			Affixes:    pa.affixes(),
			Formatting: pa.formatting(),
			TextCase:   pa.textCase(),
		}
		if !pa.has("name") {
			p.errorf(k, "missing required attribute %q", "name")
		}
		pa.finish()
		switch part.Name {
		case "given":
			nm.Given = part
		case "family":
			nm.Family = part
		}
	}
	return nm
}

func (p *parser) group(n *node) *Group {
	a := p.attrs(n)
	g := &Group{
		Pos:        n.pos,
		Delimiter:  a.str("delimiter"),
		Affixes:    a.affixes(),
		Formatting: a.formatting(),
		Display:    a.display(),
	}
	a.finish()
	g.Children = p.elements(n)
	return g
}

func (p *parser) choose(n *node) *Choose {
	c := &Choose{Pos: n.pos}
	p.attrs(n).finish()
	kids := n.cslKids()
	for i, k := range kids {
		switch k.name {
		case "if", "else-if":
			if (k.name == "if") != (i == 0) {
				p.errorf(k, "<if> must be the first child of <choose>, followed by any <else-if>")
			}
			c.Branches = append(c.Branches, p.branch(k))
		case "else":
			if i != len(kids)-1 || i == 0 {
				p.errorf(k, "<else> must be the last child of <choose>, after <if>")
			}
			p.attrs(k).finish()
			c.Else = p.elements(k)
			if c.Else == nil {
				c.Else = []Element{}
			}
		default:
			p.warnf(k, "unknown element ignored")
		}
	}
	if len(c.Branches) == 0 {
		p.errorf(n, "<choose> needs an <if>")
	}
	return c
}

var positions = map[string]bool{"first": true, "subsequent": true, "ibid": true, "ibid-with-locator": true, "near-note": true}

func (p *parser) branch(n *node) *Branch {
	a := p.attrs(n)
	b := &Branch{Pos: n.pos, Match: Match(a.enum("match", "all", "all", "any", "none"))}
	b.Disambiguate = a.boolPtr("disambiguate")
	b.IsNumeric = a.list("is-numeric")
	b.IsUncertainDate = a.list("is-uncertain-date")
	b.Locator = a.list("locator")
	b.Positions = a.list("position")
	b.Type = a.list("type")
	b.Variable = a.list("variable")
	a.finish()

	if b.Count() == 0 {
		p.errorf(n, "<%s> needs at least one condition", n.name)
	}
	p.checkVariables(n, "is-numeric", b.IsNumeric...)
	p.checkVariables(n, "variable", b.Variable...)
	for _, v := range b.IsUncertainDate {
		if VariableKindOf(v) != DateVariable {
			p.warnf(n, "is-uncertain-date: %q is not a date variable", v)
		}
	}
	for _, v := range b.Locator {
		if !IsLocator(v) {
			p.warnf(n, "unknown locator type %q", v)
		}
	}
	for _, v := range b.Positions {
		if !positions[v] {
			p.warnf(n, "unknown position %q", v)
		}
	}
	for _, v := range b.Type {
		if !IsItemType(v) {
			p.warnf(n, "unknown item type %q", v)
		}
	}
	b.Children = p.elements(n)
	return b
}

func (p *parser) checkVariables(n *node, attr string, vars ...string) {
	for _, v := range vars {
		if VariableKindOf(v) == UnknownVariable {
			p.warnf(n, "%s: unknown variable %q", attr, v)
		}
	}
}

// noKids warns about child elements of an element that has none.
func (p *parser) noKids(n *node) {
	for _, k := range n.cslKids() {
		p.warnf(k, "unknown element ignored")
	}
}
