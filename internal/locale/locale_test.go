package locale

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/front-matter/citeproc/internal/csl"
)

func parseLocale(t *testing.T, lang, body string) *csl.Locale {
	t.Helper()
	l, err := csl.ParseLocale([]byte(fmt.Sprintf(
		`<locale xmlns="http://purl.org/net/xbiblio/csl" version="1.0" xml:lang="%s">%s</locale>`, lang, body)))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func parseStyle(t *testing.T, locales string) *csl.Style {
	t.Helper()
	s, err := csl.ParseStyle([]byte(`<style xmlns="http://purl.org/net/xbiblio/csl" class="in-text" version="1.0">` +
		locales + `<citation><layout><text value="x"/></layout></citation></style>`))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// mapLoader serves parsed locale files from a map.
func mapLoader(files map[string]*csl.Locale) Loader {
	return func(tag string) (*csl.Locale, error) {
		if l, ok := files[tag]; ok {
			return l, nil
		}
		return nil, fmt.Errorf("locale %s: %w", tag, fs.ErrNotExist)
	}
}

func terms(defs ...string) string {
	return "<terms>" + strings.Join(defs, "") + "</terms>"
}

func TestFallbackOrder(t *testing.T) {
	style := parseStyle(t, `
<locale xml:lang="de-AT">`+terms(`<term name="and">style de-AT</term>`)+`</locale>
<locale xml:lang="de">`+terms(`<term name="and">style de</term><term name="et-al">style de</term>`)+`</locale>
<locale>`+terms(`<term name="et-al">style untagged</term><term name="ibid">style untagged</term><term name="no date"/>`)+`</locale>
<locale xml:lang="fr">`+terms(`<term name="at">style fr</term>`)+`</locale>`)
	files := map[string]*csl.Locale{
		"de-DE": parseLocale(t, "de-DE", terms(`<term name="ibid">file de-DE</term><term name="in">file de-DE</term><term name="no date">o. J.</term>`)),
		"en-US": parseLocale(t, "en-US", terms(`<term name="in">file en-US</term><term name="at">file en-US</term>`)),
	}
	c, err := New(style, "de-AT", mapLoader(files))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		term, want string
		found      bool
	}{
		{"and", "style de-AT", true},     // 1. exact dialect
		{"et-al", "style de", true},      // 2. language
		{"ibid", "style untagged", true}, // 3. untagged, before any file
		{"in", "file de-DE", true},       // 5. primary dialect (no de-AT file)
		{"at", "file en-US", true},       // 6. en-US; the fr style locale is not in the chain
		{"no date", "", true},            // defined empty: found, stops fallback
		{"henceforth", "", false},        // nowhere
	}
	for _, tt := range tests {
		got, found := c.Term(tt.term, csl.FormLong, false)
		if got != tt.want || found != tt.found {
			t.Errorf("Term(%q) = %q, %v; want %q, %v", tt.term, got, found, tt.want, tt.found)
		}
	}
	if c.Lang() != "de-AT" {
		t.Errorf("Lang() = %q", c.Lang())
	}
}

func TestFormFallback(t *testing.T) {
	style := parseStyle(t, `<locale>`+terms(`<term name="editor" form="short">style short</term>`)+`</locale>`)
	files := map[string]*csl.Locale{"en-US": parseLocale(t, "en-US", terms(
		`<term name="editor">editor</term>`,
		`<term name="editor" form="verb">edited by</term>`,
		`<term name="editor" form="verb-short">file verb-short</term>`,
		`<term name="page"><single>page</single><multiple>pages</multiple></term>`,
		`<term name="page" form="short"><single>p.</single><multiple>pp.</multiple></term>`,
		`<term name="section" form="symbol"><single>§</single><multiple>§§</multiple></term>`,
	))}
	c, err := New(style, "en-US", mapLoader(files))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		term   string
		form   csl.TermForm
		plural bool
		want   string
	}{
		// Locale fallback first: verb-short exists in the file, so the
		// style's short form is not used.
		{"editor", csl.FormVerbShort, false, "file verb-short"},
		{"editor", csl.FormShort, false, "style short"},
		{"editor", csl.FormSymbol, false, "style short"}, // symbol → short
		{"page", csl.FormSymbol, true, "pp."},            // symbol → short
		{"page", csl.FormVerb, true, "pages"},            // verb → long
		{"section", csl.FormSymbol, true, "§§"},
		{"section", csl.FormShort, false, ""}, // short → long: undefined
	}
	for _, tt := range tests {
		if got, _ := c.Term(tt.term, tt.form, tt.plural); got != tt.want {
			t.Errorf("Term(%q, %s, %v) = %q, want %q", tt.term, tt.form, tt.plural, got, tt.want)
		}
	}
}

func TestNewLangResolution(t *testing.T) {
	en := parseLocale(t, "en-US", "")
	de := parseLocale(t, "de-DE", terms(`<term name="and">und</term>`))
	load := mapLoader(map[string]*csl.Locale{"en-US": en, "de-DE": de})

	style := parseStyle(t, `<locale xml:lang="de-DE">`+terms(`<term name="et-al">u. a.</term>`)+`</locale>`)
	style.DefaultLocale = "de"
	c, err := New(style, "", load)
	if err != nil {
		t.Fatal(err)
	}
	if c.Lang() != "de-DE" {
		t.Errorf("Lang() = %q, want default-locale \"de\" resolved to de-DE", c.Lang())
	}
	if got, _ := c.Term("et-al", csl.FormLong, false); got != "u. a." {
		t.Errorf("style locale de-DE not used for output locale \"de\": %q", got)
	}
	if got, _ := c.Term("and", csl.FormLong, false); got != "und" {
		t.Errorf("Term(and) = %q", got)
	}

	if c, _ := New(nil, "", load); c.Lang() != "en-US" {
		t.Errorf("default Lang() = %q", c.Lang())
	}
	if _, err := New(nil, "de-DE", mapLoader(map[string]*csl.Locale{"de-DE": de})); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing en-US: err = %v, want fs.ErrNotExist", err)
	}
	broken := func(tag string) (*csl.Locale, error) { return nil, errors.New("boom") }
	if _, err := New(nil, "en-US", broken); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("loader error not propagated: %v", err)
	}
}

func TestOrdinalsEnglish(t *testing.T) {
	c := realChain(t, nil, "en-US")
	for n, want := range map[int]string{
		0: "th", 1: "st", 2: "nd", 3: "rd", 4: "th", 11: "th", 12: "th", 13: "th",
		21: "st", 22: "nd", 23: "rd", 101: "st", 111: "th", 112: "th", -2: "nd",
	} {
		if got := c.Ordinal(n, ""); got != want {
			t.Errorf("Ordinal(%d) = %q, want %q", n, got, want)
		}
	}
	for n, want := range map[int]string{1: "first", 2: "second", 10: "tenth", 11: "11th", 22: "22nd"} {
		if got := c.LongOrdinal(n, ""); got != want {
			t.Errorf("LongOrdinal(%d) = %q, want %q", n, got, want)
		}
	}
}

// TestOrdinalsGender uses the French example from the specification
// ("Gender-specific Ordinals").
func TestOrdinalsGender(t *testing.T) {
	fr := parseLocale(t, "fr-FR", terms(
		`<term name="edition" gender="feminine"><single>édition</single><multiple>éditions</multiple></term>`,
		`<term name="month-01" gender="masculine">janvier</term>`,
		`<term name="ordinal">e</term>`,
		`<term name="ordinal-01" gender-form="feminine" match="whole-number">re</term>`,
		`<term name="ordinal-01" gender-form="masculine" match="whole-number">er</term>`,
		`<term name="long-ordinal-01">premier</term>`,
		`<term name="long-ordinal-01" gender-form="feminine">première</term>`,
	))
	c, err := New(nil, "fr-FR", mapLoader(map[string]*csl.Locale{"fr-FR": fr, "en-US": parseLocale(t, "en-US", "")}))
	if err != nil {
		t.Fatal(err)
	}
	if g := c.Gender("edition"); g != csl.Feminine {
		t.Errorf("Gender(edition) = %q", g)
	}
	if g := c.Gender("month-01"); g != csl.Masculine {
		t.Errorf("Gender(month-01) = %q", g)
	}
	tests := []struct {
		n      int
		gender csl.Gender
		want   string
	}{
		{1, csl.Feminine, "re"},
		{1, csl.Masculine, "er"},
		{1, "", "e"}, // no neuter ordinal-01: falls back to "ordinal"
		{21, csl.Feminine, "e"},
		{3, csl.Masculine, "e"},
	}
	for _, tt := range tests {
		if got := c.Ordinal(tt.n, tt.gender); got != tt.want {
			t.Errorf("Ordinal(%d, %q) = %q, want %q", tt.n, tt.gender, got, tt.want)
		}
	}
	if got := c.LongOrdinal(1, csl.Feminine); got != "première" {
		t.Errorf("LongOrdinal(1, feminine) = %q", got)
	}
	if got := c.LongOrdinal(1, csl.Masculine); got != "premier" {
		t.Errorf("LongOrdinal(1, masculine) = %q, want neuter fallback", got)
	}
	if got := c.LongOrdinal(2, csl.Feminine); got != "2e" {
		t.Errorf("LongOrdinal(2, feminine) = %q, want fallback to ordinal", got)
	}
}

// Redefining any ordinal term in a style replaces the whole ordinal set of
// the locale files (specification, "Ordinal Suffixes").
func TestOrdinalsReplaceSet(t *testing.T) {
	style := parseStyle(t, `<locale>`+terms(`<term name="ordinal">x</term>`)+`</locale>`)
	c := realChain(t, style, "en-US")
	for _, n := range []int{1, 2, 11, 23} {
		if got := c.Ordinal(n, ""); got != "x" {
			t.Errorf("Ordinal(%d) = %q, want only the style's ordinal set", n, got)
		}
	}
}

func TestOrdinalsLegacy(t *testing.T) {
	style := parseStyle(t, `<locale>`+terms(
		`<term name="ordinal-01">st</term>`, `<term name="ordinal-02">nd</term>`,
		`<term name="ordinal-03">rd</term>`, `<term name="ordinal-04">th</term>`)+`</locale>`)
	c := realChain(t, style, "en-US")
	for n, want := range map[int]string{1: "st", 2: "nd", 3: "rd", 4: "th", 10: "th", 11: "th", 12: "th", 13: "th", 21: "st", 112: "th", 123: "rd"} {
		if got := c.Ordinal(n, ""); got != want {
			t.Errorf("Ordinal(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestDateFormatsAndOptions(t *testing.T) {
	c := realChain(t, nil, "en-US")
	d := c.DateFormat(csl.DateText)
	if d == nil || len(d.Parts) != 3 || d.Parts[0].Name != csl.PartMonth || d.Parts[0].Suffix != " " {
		t.Fatalf("en-US text date format: %+v", d)
	}
	if !c.PunctuationInQuote() || c.LimitDayOrdinalsToDay1() {
		t.Errorf("en-US options: punctuation-in-quote %v, limit-day-ordinals %v", c.PunctuationInQuote(), c.LimitDayOrdinalsToDay1())
	}

	style := parseStyle(t, `<locale><style-options limit-day-ordinals-to-day-1="true"/>
  <date form="text"><date-part name="year"/></date></locale>`)
	c = realChain(t, style, "en-US")
	if d := c.DateFormat(csl.DateText); len(d.Parts) != 1 {
		t.Errorf("style date format did not replace the file format as a unit: %+v", d.Parts)
	}
	if d := c.DateFormat(csl.DateNumeric); d == nil || len(d.Parts) != 3 {
		t.Errorf("numeric format should still come from the file")
	}
	if !c.PunctuationInQuote() || !c.LimitDayOrdinalsToDay1() {
		t.Errorf("options do not fall back individually")
	}
}

func TestBundledLocales(t *testing.T) {
	c := realChain(t, nil, "en-US")
	for _, tt := range []struct {
		term   string
		form   csl.TermForm
		plural bool
		want   string
	}{
		{"page", csl.FormShort, true, "pp."},
		{"section", csl.FormSymbol, false, "§"},
		{"and", csl.FormLong, false, "and"},
		{"editor", csl.FormVerbShort, false, "ed. by"},
		{"month-03", csl.FormShort, false, "Mar."},
	} {
		if got, _ := c.Term(tt.term, tt.form, tt.plural); got != tt.want {
			t.Errorf("en-US Term(%q, %s) = %q, want %q", tt.term, tt.form, got, tt.want)
		}
	}

	// No de-AT file: falls back to de-DE, then en-US.
	c = realChain(t, nil, "de-AT")
	if got, _ := c.Term("and", csl.FormLong, false); got != "und" {
		t.Errorf("de-AT Term(and) = %q", got)
	}
	if got := c.Ordinal(5, ""); got != "." {
		t.Errorf("de-AT Ordinal(5) = %q", got)
	}
	if got := c.LongOrdinal(2, ""); got != "zweiter" {
		t.Errorf("de-AT LongOrdinal(2) = %q", got)
	}
	if c.PunctuationInQuote() {
		t.Errorf("de-DE punctuation-in-quote should be false")
	}
}

func TestBaseLangAndPrimaryDialect(t *testing.T) {
	for tag, want := range map[string][2]string{
		"de-AT": {"de", "de-DE"}, "de": {"de", "de-DE"}, "EN-gb": {"en", "en-US"},
		"sr-Latn-RS": {"sr", "sr-Latn-RS"}, "xx-YY": {"xx", ""},
	} {
		if b, p := BaseLang(tag), PrimaryDialect(tag); b != want[0] || p != want[1] {
			t.Errorf("%s: BaseLang %q PrimaryDialect %q, want %v", tag, b, p, want)
		}
	}
	if !IsEnglish("en-GB") || IsEnglish("de-DE") {
		t.Error("IsEnglish")
	}
}

func realChain(t testing.TB, style *csl.Style, lang string) *Chain {
	t.Helper()
	c, err := New(style, lang, FSLoader(os.DirFS("../../locales")))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func BenchmarkChain(b *testing.B) {
	data, err := os.ReadFile("../../styles/apa.csl")
	if err != nil {
		b.Fatal(err)
	}
	style, err := csl.ParseStyle(data)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("New/apa-en-US", func(b *testing.B) {
		en := realChain(b, nil, "en-US").levels[0].loc
		load := mapLoader(map[string]*csl.Locale{"en-US": en})
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := New(style, "en-US", load); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Term", func(b *testing.B) {
		c := realChain(b, style, "en-US")
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.Term("editor", csl.FormVerbShort, false)
		}
	})
}
