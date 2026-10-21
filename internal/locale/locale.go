// Package locale resolves CSL localization data: terms, localized date
// formats, ordinals and locale options.
//
// It implements the CSL 1.0.2 locale fallback (specification, "Locale
// Fallback"). For an output locale such as "de-AT", each localizable unit is
// taken from the first of these sources that defines it:
//
//  1. in-style cs:locale with xml:lang "de-AT"
//  2. in-style cs:locale with xml:lang "de"
//  3. in-style cs:locale without xml:lang
//  4. the locale file for "de-AT"
//  5. the locale file for the primary dialect, "de-DE"
//  6. the locale file for "en-US"
//
// A bare language ("de") is treated as its primary dialect ("de-DE"). Locale
// fallback takes precedence over term-form fallback, as the spec requires.
//
// Ported from hayagriva 0.10.1 src/csl/mod.rs (StyleContext::lookup_locale,
// Context::term, Context::gender) and citationberg 0.7.0 src/lib.rs
// (TermForm::fallback, OrdinalLookup).
package locale

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/front-matter/citeproc/internal/csl"
)

// Loader returns the parsed locale file for a tag such as "de-DE". It must
// return an error wrapping fs.ErrNotExist if there is no file for the tag.
type Loader func(tag string) (*csl.Locale, error)

// FSLoader returns a Loader that reads "locales-<tag>.xml" from fsys and
// parses it on every call. Callers that load repeatedly should cache.
func FSLoader(fsys fs.FS) Loader {
	return func(tag string) (*csl.Locale, error) {
		data, err := fs.ReadFile(fsys, "locales-"+tag+".xml")
		if err != nil {
			return nil, err
		}
		l, err := csl.ParseLocale(data)
		if err != nil {
			return nil, fmt.Errorf("locales-%s.xml: %w", tag, err)
		}
		return l, nil
	}
}

// Chain is the resolved locale fallback chain for one style and output
// locale. It is immutable and safe for concurrent use.
type Chain struct {
	lang     string
	levels   []level
	ordinals []*csl.Term
}

type termKey struct {
	name string
	form csl.TermForm
}

// level is one source in the chain, with its terms indexed.
type level struct {
	loc   *csl.Locale
	terms map[termKey]*csl.Term // neuter (non-gender-form) terms only
}

func newLevel(l *csl.Locale) level {
	lv := level{loc: l, terms: make(map[termKey]*csl.Term, len(l.Terms))}
	for _, t := range l.Terms {
		if t.GenderForm != "" {
			continue
		}
		k := termKey{t.Name, t.Form}
		if _, dup := lv.terms[k]; !dup {
			lv.terms[k] = t
		}
	}
	return lv
}

// ResolveLang returns the output locale for a requested tag: the request,
// else the style's default-locale, else "en-US". A bare language becomes its
// primary dialect.
func ResolveLang(requested string, style *csl.Style) string {
	lang := requested
	if lang == "" && style != nil {
		lang = style.DefaultLocale
	}
	if lang == "" {
		lang = "en-US"
	}
	if !strings.Contains(lang, "-") {
		if p := PrimaryDialect(lang); p != "" {
			lang = p
		}
	}
	return lang
}

// New builds the chain for style (may be nil) and the requested output
// locale (see ResolveLang). Missing locale files are skipped, except that
// en-US, the final fallback, must exist.
func New(style *csl.Style, requested string, load Loader) (*Chain, error) {
	lang := ResolveLang(requested, style)
	c := &Chain{lang: lang}

	if style != nil {
		base := BaseLang(lang)
		for _, match := range []func(string) bool{
			func(t string) bool { return sameTag(t, lang) },
			func(t string) bool { return sameTag(t, base) && !sameTag(base, lang) },
			func(t string) bool { return t == "" },
		} {
			for _, l := range style.Locales {
				if match(l.Lang) {
					c.levels = append(c.levels, newLevel(l))
				}
			}
		}
	}

	files := []string{lang}
	if p := PrimaryDialect(lang); p != "" && !sameTag(p, lang) {
		files = append(files, p)
	}
	if !sameTag(lang, "en-US") && !sameTag(files[len(files)-1], "en-US") {
		files = append(files, "en-US")
	}
	for _, tag := range files {
		l, err := load(tag)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if sameTag(tag, "en-US") {
				return nil, fmt.Errorf("locale: fallback locale en-US: %w", err)
			}
			continue
		case err != nil:
			return nil, fmt.Errorf("locale %s: %w", tag, err)
		}
		c.levels = append(c.levels, newLevel(l))
	}

	// Ordinal terms do not fall back individually: the first source that
	// defines any of "ordinal" and "ordinal-00".."ordinal-99" supplies all
	// of them (specification, "Ordinal Suffixes").
	for _, lv := range c.levels {
		for _, t := range lv.loc.Terms {
			if csl.IsOrdinalTerm(t.Name) {
				c.ordinals = append(c.ordinals, t)
			}
		}
		if c.ordinals != nil {
			break
		}
	}
	return c, nil
}

// Lang returns the resolved output locale, e.g. "en-US".
func (c *Chain) Lang() string { return c.lang }

// formFallback is the next form to try when a term form is undefined.
var formFallback = map[csl.TermForm]csl.TermForm{
	csl.FormVerbShort: csl.FormVerb,
	csl.FormSymbol:    csl.FormShort,
	csl.FormVerb:      csl.FormLong,
	csl.FormShort:     csl.FormLong,
}

// Term returns the singular or plural text of a term. Locale fallback is
// tried for each form before falling back to the next form. A term defined
// as empty is found (and returns ""), which lets styles suppress terms.
// Found is false if no source defines the term in any fallback form.
func (c *Chain) Term(name string, form csl.TermForm, plural bool) (text string, found bool) {
	if t := c.lookup(name, form); t != nil {
		return t.Value(plural), true
	}
	return "", false
}

func (c *Chain) lookup(name string, form csl.TermForm) *csl.Term {
	if form == "" {
		form = csl.FormLong
	}
	for {
		for _, lv := range c.levels {
			if t, ok := lv.terms[termKey{name, form}]; ok {
				return t
			}
		}
		next, ok := formFallback[form]
		if !ok {
			return nil
		}
		form = next
	}
}

// Gender returns the grammatical gender of a noun term (the gender attribute
// of its long form), or "" if it has none.
func (c *Chain) Gender(name string) csl.Gender {
	for _, lv := range c.levels {
		if t, ok := lv.terms[termKey{name, csl.FormLong}]; ok {
			return t.Gender
		}
	}
	return ""
}

// DateFormat returns the localized date format of the given form, or nil.
// A format is a single unit: its parts never mix sources.
func (c *Chain) DateFormat(form csl.DateForm) *csl.Date {
	for _, lv := range c.levels {
		for _, d := range lv.loc.Dates {
			if d.Form == form {
				return d
			}
		}
	}
	return nil
}

// PunctuationInQuote reports the punctuation-in-quote locale option.
func (c *Chain) PunctuationInQuote() bool {
	return c.option(func(o csl.LocaleOptions) *bool { return o.PunctuationInQuote })
}

// LimitDayOrdinalsToDay1 reports the limit-day-ordinals-to-day-1 option.
func (c *Chain) LimitDayOrdinalsToDay1() bool {
	return c.option(func(o csl.LocaleOptions) *bool { return o.LimitDayOrdinalsToDay1 })
}

func (c *Chain) option(get func(csl.LocaleOptions) *bool) bool {
	for _, lv := range c.levels {
		if v := get(lv.loc.StyleOptions); v != nil {
			return *v
		}
	}
	return false
}
