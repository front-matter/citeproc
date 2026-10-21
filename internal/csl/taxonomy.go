package csl

import (
	"regexp"
	"strings"
)

// The tables in this file are transcribed from the CSL 1.0.2 schema
// (github.com/citation-style-language/schema, tag v1.0.2, commit 5060400):
// csl-variables.rnc, csl-terms.rnc and csl-types.rnc. The parser uses them to
// warn about unknown names. VariableKindOf tells the renderer how a variable
// is resolved.

// VariableKind classifies a CSL variable.
type VariableKind int

const (
	// UnknownVariable is not a CSL 1.0.2 variable.
	UnknownVariable VariableKind = iota
	// StandardVariable is a string or title variable.
	StandardVariable
	// NumberVariable is a number variable (rendered with cs:number or cs:text).
	NumberVariable
	// DateVariable is a date variable (rendered with cs:date).
	DateVariable
	// NameVariable is a name variable (rendered with cs:names).
	NameVariable
)

func (k VariableKind) String() string {
	switch k {
	case StandardVariable:
		return "standard"
	case NumberVariable:
		return "number"
	case DateVariable:
		return "date"
	case NameVariable:
		return "name"
	}
	return "unknown"
}

var variableKinds = map[string]VariableKind{}

func init() {
	add := func(kind VariableKind, names string) {
		for _, n := range strings.Fields(names) {
			variableKinds[n] = kind
		}
	}
	add(DateVariable, `accessed available-date event-date issued original-date submitted`)
	add(NameVariable, `author chair collection-editor compiler composer container-author
		contributor curator director editor editor-translator editorial-director
		executive-producer guest host illustrator interviewer narrator organizer
		original-author performer producer recipient reviewed-author script-writer
		series-creator translator`)
	add(NumberVariable, `chapter-number citation-number collection-number edition
		first-reference-note-number issue locator number number-of-pages
		number-of-volumes page page-first part-number printing-number section
		supplement-number version volume`)
	// Titles and strings.
	add(StandardVariable, `collection-title container-title original-title part-title
		reviewed-title title volume-title title-short container-title-short
		abstract annote archive archive_collection archive_location archive-place
		authority call-number citation-key citation-label dimensions division DOI
		event event-title event-place genre ISBN ISSN jurisdiction keyword language
		license medium note original-publisher original-publisher-place PMCID PMID
		publisher publisher-place references reviewed-genre scale source status URL
		year-suffix`)
}

// VariableKindOf reports the kind of a CSL variable name.
func VariableKindOf(name string) VariableKind { return variableKinds[name] }

var itemTypes = set(`article article-journal article-magazine article-newspaper bill
	book broadcast chapter classic collection dataset document entry
	entry-dictionary entry-encyclopedia event figure graphic hearing interview
	legal_case legislation manuscript map motion_picture musical_score pamphlet
	paper-conference patent performance periodical personal_communication post
	post-weblog regulation report review review-book software song speech
	standard thesis treaty webpage`)

// IsItemType reports whether name is a CSL 1.0.2 item type.
func IsItemType(name string) bool { return itemTypes[name] }

var locatorTerms = set(`act appendix article-locator book canon chapter column
	elocation equation figure folio line note opus paragraph rule scene sub-verbo
	table timestamp title-locator verse issue page part section supplement
	version volume`)

// IsLocator reports whether name is a CSL 1.0.2 locator type.
func IsLocator(name string) bool { return locatorTerms[name] }

var miscTerms = set(`editortranslator accessed ad advance-online-publication album
	and anonymous at audio-recording bc bce by ce circa cited et-al film
	forthcoming from henceforth ibid in internet interview letter loc-cit
	no-place no-publisher on online op-cit original-work-published
	personal-communication podcast podcast-episode preprint radio-broadcast
	radio-series radio-series-episode reference retrieved review-of scale
	special-issue special-section television-broadcast television-series
	television-series-episode video working-paper open-quote close-quote
	open-inner-quote close-inner-quote page-range-delimiter colon comma semicolon
	season-01 season-02 season-03 season-04 month-01 month-02 month-03 month-04
	month-05 month-06 month-07 month-08 month-09 month-10 month-11 month-12
	chapter-number citation-number collection-number edition
	first-reference-note-number number number-of-pages number-of-volumes
	page-first printing long-ordinal-01 long-ordinal-02 long-ordinal-03
	long-ordinal-04 long-ordinal-05 long-ordinal-06 long-ordinal-07
	long-ordinal-08 long-ordinal-09 long-ordinal-10`)

// Terms whose names contain spaces cannot go through set().
var spacedTerms = map[string]bool{
	"and others": true, "available at": true, "in press": true,
	"no date": true, "presented at": true,
}

var ordinalTerm = regexp.MustCompile(`^ordinal(-\d{2})?$`)

// IsTerm reports whether name is a CSL 1.0.2 term name.
func IsTerm(name string) bool {
	return miscTerms[name] || spacedTerms[name] || locatorTerms[name] ||
		itemTypes[name] || variableKinds[name] == NameVariable ||
		ordinalTerm.MatchString(name)
}

// IsOrdinalTerm reports whether name is "ordinal" or "ordinal-00".."ordinal-99".
// Long ordinals are not included.
func IsOrdinalTerm(name string) bool { return ordinalTerm.MatchString(name) }

func set(names string) map[string]bool {
	m := map[string]bool{}
	for _, n := range strings.Fields(names) {
		m[n] = true
	}
	return m
}
