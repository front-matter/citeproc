// Package csl parses CSL 1.0.2 styles and locale files into a typed AST.
//
// It refactors the Rust crate citationberg, which hayagriva uses for the same
// purpose (see docs/port-map.md §3). The AST follows the CSL 1.0.2 schema
// (github.com/citation-style-language/schema, tag v1.0.2):
//
//   - Attributes that the schema gives a default are stored with the default
//     applied, except where "unset" and "default" mean different things (for
//     example inheritable name options and delimiters with a fallback chain).
//     Those are pointers or empty strings, and their doc comments say which.
//   - Enumerated attributes are string types whose zero value means "unset".
//   - Every node records the line and column of its start tag.
//
// The package does no rendering and resolves no locales; see internal/locale
// and internal/render.
package csl

// Pos is the 1-based line and column of an element's start tag.
type Pos struct {
	Line, Col int
}

// Position returns p. It lets every AST node that embeds Pos report its
// position through the Element interface.
func (p Pos) Position() Pos { return p }

// Element is a CSL rendering element: *Text, *Number, *Label, *Date, *Names,
// *Group or *Choose.
type Element interface {
	Position() Pos
	isElement()
}

func (*Text) isElement()   {}
func (*Number) isElement() {}
func (*Label) isElement()  {}
func (*Date) isElement()   {}
func (*Names) isElement()  {}
func (*Group) isElement()  {}
func (*Choose) isElement() {}

// ---------------------------------------------------------------------------
// Style root

// Style is a parsed independent or dependent CSL style.
type Style struct {
	Pos
	Class         StyleClass // "in-text" or "note"; empty for dependent styles
	Version       string
	DefaultLocale string // "" if unset

	// Global options (style element only).
	DemoteNonDroppingParticle DemoteParticle  // default "display-and-sort"
	InitializeWithHyphen      bool            // default true
	PageRangeFormat           PageRangeFormat // "" if unset (no reformatting)

	// NameOptions are the inheritable name options set on cs:style.
	NameOptions InheritableNameOptions

	Info    Info
	Locales []*Locale
	Macros  map[string]*Macro
	// MacroOrder lists macro names in document order.
	MacroOrder   []string
	Citation     *Citation     // nil for dependent styles
	Bibliography *Bibliography // nil if absent

	// Warnings are non-fatal problems: unknown elements, attributes, terms or
	// variables. The parser ignores the offending construct.
	Warnings []*ParseError
}

// IndependentParent returns the URI of the parent style of a dependent style,
// or "" for an independent style.
func (s *Style) IndependentParent() string {
	for _, l := range s.Info.Links {
		if l.Rel == "independent-parent" {
			return l.Href
		}
	}
	return ""
}

// IsDependent reports whether s is a dependent style (it has no citation
// element and links to an independent parent).
func (s *Style) IsDependent() bool { return s.Citation == nil && s.IndependentParent() != "" }

// CitationNameOptions returns the name options in scope for cs:citation:
// those set on cs:citation, falling back to cs:style.
func (s *Style) CitationNameOptions() InheritableNameOptions {
	if s.Citation == nil {
		return s.NameOptions
	}
	return s.Citation.NameOptions.Inherit(s.NameOptions)
}

// BibliographyNameOptions returns the name options in scope for
// cs:bibliography: those set on cs:bibliography, falling back to cs:style.
func (s *Style) BibliographyNameOptions() InheritableNameOptions {
	if s.Bibliography == nil {
		return s.NameOptions
	}
	return s.Bibliography.NameOptions.Inherit(s.NameOptions)
}

// Info holds style metadata (cs:info).
type Info struct {
	Title          string
	TitleShort     string
	ID             string
	Authors        []Person
	Contributors   []Person
	CitationFormat string   // category citation-format, e.g. "author-date"
	Fields         []string // category field values
	ISSN           []string
	EISSN          string
	ISSNL          string
	Links          []Link
	Published      string
	Rights         string
	RightsLicense  string
	Summary        string
	Updated        string
}

// Person is a style author, contributor or locale translator.
type Person struct {
	Name, Email, URI string
}

// Link is a cs:link element in cs:info.
type Link struct {
	Href, Rel, Lang string
}

// Macro is a cs:macro element.
type Macro struct {
	Pos
	Name     string
	Children []Element
}

// Citation is the cs:citation element.
type Citation struct {
	Pos
	DisambiguateAddNames        bool
	DisambiguateAddGivenname    bool
	DisambiguateAddYearSuffix   bool
	GivennameDisambiguationRule GivennameRule // default "by-cite"
	Collapse                    Collapse      // "" if unset
	// CiteGroupDelimiter is nil if unset. Cite grouping is active when it is
	// set or when Collapse is a year mode; the default is ", ".
	CiteGroupDelimiter *string
	// YearSuffixDelimiter and AfterCollapseDelimiter are nil if unset; both
	// default to the layout delimiter.
	YearSuffixDelimiter    *string
	AfterCollapseDelimiter *string
	NearNoteDistance       int // default 5

	NameOptions InheritableNameOptions
	Sort        *Sort
	Layout      *Layout
}

// DefaultCiteGroupDelimiter is the schema default of cite-group-delimiter.
const DefaultCiteGroupDelimiter = ", "

// CiteGroupDelimiterOrDefault returns cite-group-delimiter, or ", " if unset.
func (c *Citation) CiteGroupDelimiterOrDefault() string {
	if c.CiteGroupDelimiter != nil {
		return *c.CiteGroupDelimiter
	}
	return DefaultCiteGroupDelimiter
}

// YearSuffixDelimiterOrDefault returns year-suffix-delimiter, falling back to
// the layout delimiter as the spec requires.
func (c *Citation) YearSuffixDelimiterOrDefault() string {
	if c.YearSuffixDelimiter != nil {
		return *c.YearSuffixDelimiter
	}
	return c.Layout.Delimiter
}

// AfterCollapseDelimiterOrDefault returns after-collapse-delimiter, falling
// back to the layout delimiter as the spec requires.
func (c *Citation) AfterCollapseDelimiterOrDefault() string {
	if c.AfterCollapseDelimiter != nil {
		return *c.AfterCollapseDelimiter
	}
	return c.Layout.Delimiter
}

// Bibliography is the cs:bibliography element.
type Bibliography struct {
	Pos
	HangingIndent    bool
	SecondFieldAlign SecondFieldAlign // "" if unset
	LineSpacing      int              // default 1
	EntrySpacing     int              // default 1
	// SubsequentAuthorSubstitute is nil if unset (no substitution).
	SubsequentAuthorSubstitute     *string
	SubsequentAuthorSubstituteRule SubstituteRule // default "complete-all"

	NameOptions InheritableNameOptions
	Sort        *Sort
	Layout      *Layout
}

// Layout is a cs:layout element.
type Layout struct {
	Pos
	Affixes
	Formatting
	Delimiter string // citation layouts only
	Children  []Element
}

// Sort is a cs:sort element.
type Sort struct {
	Pos
	Keys []*SortKey
}

// SortKey is a cs:key element. Exactly one of Variable and Macro is set.
type SortKey struct {
	Pos
	Variable   string
	Macro      string
	Descending bool
	// Overrides of et-al-min, et-al-use-first and et-al-use-last for names
	// rendered through Macro; nil if unset.
	NamesMin      *int
	NamesUseFirst *int
	NamesUseLast  *bool
}

// ---------------------------------------------------------------------------
// Rendering elements

// TextSource says what a cs:text element renders.
type TextSource string

const (
	TextVariable TextSource = "variable"
	TextMacro    TextSource = "macro"
	TextTerm     TextSource = "term"
	TextValue    TextSource = "value"
)

// Text is a cs:text element.
type Text struct {
	Pos
	Source TextSource
	// Name is the variable, macro or term name; Value is the literal for
	// Source == TextValue.
	Name  string
	Value string
	// Form is "long" or "short" for variables, any TermForm for terms.
	Form   TermForm
	Plural bool // terms only

	Affixes
	Formatting
	Display      Display
	Quotes       bool
	StripPeriods bool
	TextCase     TextCase
}

// Number is a cs:number element.
type Number struct {
	Pos
	Variable string
	Form     NumberForm // default "numeric"
	Affixes
	Formatting
	Display  Display
	TextCase TextCase
}

// Label is a cs:label element, either standalone or inside cs:names.
type Label struct {
	Pos
	Variable string // "" inside cs:names
	Form     TermForm
	Plural   LabelPlural // default "contextual"
	Affixes
	Formatting
	StripPeriods bool
	TextCase     TextCase
}

// Date is a cs:date element. In a style it renders a date variable; in a
// locale it defines a localized date format (Variable is then "").
type Date struct {
	Pos
	Variable string
	// Form is "text" or "numeric" for localized dates and locale formats,
	// "" for non-localized dates.
	Form DateForm
	// DateParts selects the parts of a localized date; default
	// "year-month-day". Empty for non-localized dates and locale formats.
	DateParts DatePartsSel
	Parts     []*DatePart
	Delimiter string // non-localized dates and locale formats only
	Affixes
	Formatting
	Display  Display
	TextCase TextCase
}

// IsLocalized reports whether d uses a localized date format.
func (d *Date) IsLocalized() bool { return d.Form != "" }

// DatePart is a cs:date-part element.
type DatePart struct {
	Pos
	Name DatePartName
	// Form is "" if unset. Defaults depend on Name and on whether the part
	// overrides a localized format, so they are applied at render time.
	Form string
	// RangeDelimiter is nil if unset (the schema default is an en dash).
	RangeDelimiter *string
	Affixes
	Formatting
	StripPeriods bool
	TextCase     TextCase
}

// Names is a cs:names element.
type Names struct {
	Pos
	Variables []string
	Name      *Name // nil if absent (default cs:name)
	EtAl      *EtAl
	Label     *Label
	// LabelBeforeName reports whether cs:label precedes cs:name.
	LabelBeforeName bool
	Substitute      *Substitute
	// Delimiter separates the names of different variables; nil if unset
	// (inherits names-delimiter).
	Delimiter *string
	Affixes
	Formatting
	Display Display
}

// Name is a cs:name element.
type Name struct {
	Pos
	// Options holds the name attributes set on this element, including form
	// and delimiter. Unset options inherit from cs:citation/cs:bibliography
	// and cs:style.
	Options NameOptions
	Affixes
	Formatting
	Given  *NamePart
	Family *NamePart
}

// NamePart is a cs:name-part element.
type NamePart struct {
	Pos
	Name string // "given" or "family"
	Affixes
	Formatting
	TextCase TextCase
}

// EtAl is a cs:et-al element.
type EtAl struct {
	Pos
	Term string // "et-al" (default) or "and others"
	Formatting
}

// Substitute is a cs:substitute element.
type Substitute struct {
	Pos
	Children []Element
}

// Group is a cs:group element.
type Group struct {
	Pos
	Delimiter string
	Affixes
	Formatting
	Display  Display
	Children []Element
}

// Choose is a cs:choose element.
type Choose struct {
	Pos
	// Branches holds cs:if followed by any cs:else-if elements.
	Branches []*Branch
	// Else holds the children of cs:else; nil if there is no cs:else.
	Else []Element
}

// Branch is a cs:if or cs:else-if element.
type Branch struct {
	Pos
	Match Match // default "all"
	Conditions
	Children []Element
}

// Conditions are the tests of a cs:if or cs:else-if element.
type Conditions struct {
	// Disambiguate is nil if unset. The schema only allows "true"; "false" is
	// accepted as hayagriva does.
	Disambiguate    *bool
	IsNumeric       []string
	IsUncertainDate []string
	Locator         []string
	Positions       []string // the position attribute
	Type            []string
	Variable        []string
}

// Count returns the number of individual tests (one per listed value).
func (c Conditions) Count() int {
	n := len(c.IsNumeric) + len(c.IsUncertainDate) + len(c.Locator) +
		len(c.Positions) + len(c.Type) + len(c.Variable)
	if c.Disambiguate != nil {
		n++
	}
	return n
}

// ---------------------------------------------------------------------------
// Locales

// Locale is a cs:locale element, either inline in a style or the root of a
// locale file.
type Locale struct {
	Pos
	Lang    string // "" for an untagged inline locale
	Version string // locale files only
	// Translators, Rights and Updated come from cs:info in locale files.
	Translators []Person
	Rights      string
	Updated     string

	StyleOptions LocaleOptions
	Dates        []*Date
	Terms        []*Term

	// Warnings are non-fatal problems in a locale file. Problems in inline
	// locales are reported in Style.Warnings instead.
	Warnings []*ParseError
}

// LocaleOptions is cs:style-options. Nil fields are unset, so that each
// option falls back through the locale chain on its own.
type LocaleOptions struct {
	LimitDayOrdinalsToDay1 *bool
	PunctuationInQuote     *bool
}

// Term is a cs:term element.
type Term struct {
	Pos
	Name string
	Form TermForm // default "long"
	// Single and Multiple are the singular and plural forms. A term given as
	// plain text sets both; a term with only one of cs:single and
	// cs:multiple uses it for both.
	Single, Multiple string
	Gender           Gender       // nouns: gender of the long form
	GenderForm       Gender       // ordinals: gender variant ("" = neuter)
	Match            OrdinalMatch // ordinals: "" if unset
}

// Value returns the plural or singular form.
func (t *Term) Value(plural bool) string {
	if plural {
		return t.Multiple
	}
	return t.Single
}

// ---------------------------------------------------------------------------
// Shared attribute groups

// Affixes holds the prefix and suffix attributes.
type Affixes struct {
	Prefix, Suffix string
}

// Formatting holds the font-formatting attributes. Empty fields are unset
// and inherit the surrounding formatting.
type Formatting struct {
	FontStyle      FontStyle
	FontVariant    FontVariant
	FontWeight     FontWeight
	TextDecoration TextDecoration
	VerticalAlign  VerticalAlign
}

// IsZero reports whether no formatting attribute is set.
func (f Formatting) IsZero() bool { return f == Formatting{} }

// ---------------------------------------------------------------------------
// Enumerations. The empty string means "unset" for all of them.

type (
	StyleClass       string
	DemoteParticle   string
	PageRangeFormat  string
	GivennameRule    string
	Collapse         string
	SecondFieldAlign string
	SubstituteRule   string
	TermForm         string
	NumberForm       string
	LabelPlural      string
	DateForm         string
	DatePartsSel     string
	DatePartName     string
	Match            string
	Display          string
	TextCase         string
	FontStyle        string
	FontVariant      string
	FontWeight       string
	TextDecoration   string
	VerticalAlign    string
	Gender           string
	OrdinalMatch     string
	NameAnd          string
	DelimiterRule    string
	NameAsSortOrder  string
	NameForm         string
)

const (
	ClassInText StyleClass = "in-text"
	ClassNote   StyleClass = "note"

	DemoteNever          DemoteParticle = "never"
	DemoteSortOnly       DemoteParticle = "sort-only"
	DemoteDisplayAndSort DemoteParticle = "display-and-sort"

	PageRangeExpanded   PageRangeFormat = "expanded"
	PageRangeMinimal    PageRangeFormat = "minimal"
	PageRangeMinimalTwo PageRangeFormat = "minimal-two"
	PageRangeChicago    PageRangeFormat = "chicago"
	PageRangeChicago15  PageRangeFormat = "chicago-15"
	PageRangeChicago16  PageRangeFormat = "chicago-16"

	RuleAllNames                GivennameRule = "all-names"
	RuleAllNamesWithInitials    GivennameRule = "all-names-with-initials"
	RulePrimaryName             GivennameRule = "primary-name"
	RulePrimaryNameWithInitials GivennameRule = "primary-name-with-initials"
	RuleByCite                  GivennameRule = "by-cite"

	CollapseCitationNumber   Collapse = "citation-number"
	CollapseYear             Collapse = "year"
	CollapseYearSuffix       Collapse = "year-suffix"
	CollapseYearSuffixRanged Collapse = "year-suffix-ranged"

	AlignFlush  SecondFieldAlign = "flush"
	AlignMargin SecondFieldAlign = "margin"

	SubstituteCompleteAll  SubstituteRule = "complete-all"
	SubstituteCompleteEach SubstituteRule = "complete-each"
	SubstitutePartialEach  SubstituteRule = "partial-each"
	SubstitutePartialFirst SubstituteRule = "partial-first"

	FormLong      TermForm = "long"
	FormShort     TermForm = "short"
	FormVerb      TermForm = "verb"
	FormVerbShort TermForm = "verb-short"
	FormSymbol    TermForm = "symbol"

	NumberNumeric     NumberForm = "numeric"
	NumberOrdinal     NumberForm = "ordinal"
	NumberLongOrdinal NumberForm = "long-ordinal"
	NumberRoman       NumberForm = "roman"

	PluralContextual LabelPlural = "contextual"
	PluralAlways     LabelPlural = "always"
	PluralNever      LabelPlural = "never"

	DateText    DateForm = "text"
	DateNumeric DateForm = "numeric"

	PartsYearMonthDay DatePartsSel = "year-month-day"
	PartsYearMonth    DatePartsSel = "year-month"
	PartsYear         DatePartsSel = "year"

	PartDay   DatePartName = "day"
	PartMonth DatePartName = "month"
	PartYear  DatePartName = "year"

	MatchAll  Match = "all"
	MatchAny  Match = "any"
	MatchNone Match = "none"

	DisplayBlock       Display = "block"
	DisplayLeftMargin  Display = "left-margin"
	DisplayRightInline Display = "right-inline"
	DisplayIndent      Display = "indent"

	CaseLowercase       TextCase = "lowercase"
	CaseUppercase       TextCase = "uppercase"
	CaseCapitalizeFirst TextCase = "capitalize-first"
	CaseCapitalizeAll   TextCase = "capitalize-all"
	CaseTitle           TextCase = "title"
	CaseSentence        TextCase = "sentence"

	StyleNormal  FontStyle = "normal"
	StyleItalic  FontStyle = "italic"
	StyleOblique FontStyle = "oblique"

	VariantNormal    FontVariant = "normal"
	VariantSmallCaps FontVariant = "small-caps"

	WeightNormal FontWeight = "normal"
	WeightBold   FontWeight = "bold"
	WeightLight  FontWeight = "light"

	DecorationNone      TextDecoration = "none"
	DecorationUnderline TextDecoration = "underline"

	AlignBaseline VerticalAlign = "baseline"
	AlignSup      VerticalAlign = "sup"
	AlignSub      VerticalAlign = "sub"

	Masculine Gender = "masculine"
	Feminine  Gender = "feminine"

	MatchLastDigit     OrdinalMatch = "last-digit"
	MatchLastTwoDigits OrdinalMatch = "last-two-digits"
	MatchWholeNumber   OrdinalMatch = "whole-number"

	AndText   NameAnd = "text"
	AndSymbol NameAnd = "symbol"

	DelimiterContextual        DelimiterRule = "contextual"
	DelimiterAlways            DelimiterRule = "always"
	DelimiterNever             DelimiterRule = "never"
	DelimiterAfterInvertedName DelimiterRule = "after-inverted-name"

	SortOrderFirst NameAsSortOrder = "first"
	SortOrderAll   NameAsSortOrder = "all"

	NameLong  NameForm = "long"
	NameShort NameForm = "short"
	NameCount NameForm = "count"
)
