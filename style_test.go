package citeproc

import (
	"errors"
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/front-matter/citeproc/internal/csl"
	"github.com/front-matter/citeproc/internal/locale"
)

func TestLoadStyle(t *testing.T) {
	for _, name := range []string{"apa", "modern-language-association", "vancouver"} {
		s, err := LoadStyle(Styles, name)
		if err != nil {
			t.Fatalf("LoadStyle(%q): %v", name, err)
		}
		if s.Title() == "" || s.ID() == "" || len(s.Warnings()) != 0 {
			t.Errorf("%s: title %q, id %q, %d warnings", name, s.Title(), s.ID(), len(s.Warnings()))
		}
	}
	s, err := LoadStyle(Styles, "apa.csl")
	if err != nil || s.Title() != "American Psychological Association 7th edition" {
		t.Errorf("LoadStyle(apa.csl) = %v, %v", s, err)
	}
}

func TestLoadStyleUnknown(t *testing.T) {
	_, err := LoadStyle(Styles, "nonexistent")
	if !errors.Is(err, ErrUnknownStyle) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want ErrUnknownStyle wrapping fs.ErrNotExist", err)
	}
}

func TestLoadStyleParseError(t *testing.T) {
	fsys := fstest.MapFS{"bad.csl": {Data: []byte(`<style xmlns="http://purl.org/net/xbiblio/csl" class="in-text" version="1.0">
<citation><layout><text macro="missing"/></layout></citation></style>`)}}
	_, err := LoadStyle(fsys, "bad")
	var pe *ParseError
	if !errors.As(err, &pe) || pe.Line != 2 || pe.Col != 19 {
		t.Fatalf("err = %v, want a *ParseError at 2:19", err)
	}
}

// The cache is keyed by content: the same bytes in a different (and
// non-comparable) file system give the same *Style, concurrently.
func TestLoadStyleCache(t *testing.T) {
	data, err := fs.ReadFile(Styles, "apa.csl")
	if err != nil {
		t.Fatal(err)
	}
	want, err := LoadStyle(Styles, "apa")
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"copy.csl": {Data: data}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, err := LoadStyle(fsys, "copy"); err != nil || got != want {
				t.Errorf("LoadStyle(copy) = %p, %v; want cached %p", got, err, want)
			}
		}()
	}
	wg.Wait()
	if fresh, _ := ParseStyle(data); fresh == want {
		t.Error("ParseStyle returned the cached value")
	}
}

func TestLocaleLoader(t *testing.T) {
	style, err := LoadStyle(Styles, "apa")
	if err != nil {
		t.Fatal(err)
	}
	c, err := locale.New(style.ast, "en-US", localeLoader(Locales))
	if err != nil {
		t.Fatal(err)
	}
	// APA's inline locale overrides the en-US file.
	if got, _ := c.Term("bc", csl.FormLong, false); got != " B.C.E." {
		t.Errorf("Term(bc) = %q", got)
	}
	if got, _ := c.Term("et-al", csl.FormLong, false); got != "et al." {
		t.Errorf("Term(et-al) = %q", got)
	}
}

func BenchmarkLoadStyle(b *testing.B) {
	data, err := fs.ReadFile(Styles, "apa.csl")
	if err != nil {
		b.Fatal(err)
	}
	b.Run("apa/cold", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := ParseStyle(data); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("apa/cached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := LoadStyle(Styles, "apa"); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkLoadLocale(b *testing.B) {
	data, err := fs.ReadFile(Locales, "locales-en-US.xml")
	if err != nil {
		b.Fatal(err)
	}
	b.Run("en-US/cold", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := csl.ParseLocale(data); err != nil {
				b.Fatal(err)
			}
		}
	})
	load := localeLoader(Locales)
	b.Run("en-US/cached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := load("en-US"); err != nil {
				b.Fatal(err)
			}
		}
	})
}
