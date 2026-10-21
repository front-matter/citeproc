package citeproc

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/front-matter/citeproc/internal/csl"
	"github.com/front-matter/citeproc/internal/locale"
)

// ParseError describes a problem at a line and column of a style or locale
// file. Fatal problems are returned in an ErrorList; non-fatal ones are
// reported by Style.Warnings. Use errors.As to extract the first ParseError
// from an error.
type ParseError = csl.ParseError

// ErrorList holds all fatal problems found while parsing a style or locale.
type ErrorList = csl.ErrorList

// ErrUnknownStyle is returned (wrapped) by LoadStyle when the style file does
// not exist.
var ErrUnknownStyle = errors.New("citeproc: unknown style")

// Style is a parsed and validated CSL style. It is immutable and safe for
// concurrent use; one Style can serve any number of processors.
type Style struct {
	ast *csl.Style
}

// LoadStyle reads "<name>.csl" from fsys (for example LoadStyle(Styles,
// "apa")) and parses it. Parsed styles are cached by file content, so loading
// the same style again is cheap and returns the same *Style.
func LoadStyle(fsys fs.FS, name string) (*Style, error) {
	file := strings.TrimSuffix(name, ".csl") + ".csl"
	data, err := fs.ReadFile(fsys, file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w %q: %w", ErrUnknownStyle, name, err)
	}
	if err != nil {
		return nil, fmt.Errorf("citeproc: reading style %q: %w", name, err)
	}
	s, err := styleCache.get(data, parseStyle)
	if err != nil {
		return nil, fmt.Errorf("citeproc: style %q: %w", name, err)
	}
	return s, nil
}

// ParseStyle parses a CSL style from XML without caching.
func ParseStyle(data []byte) (*Style, error) {
	s, err := parseStyle(data)
	if err != nil {
		return nil, fmt.Errorf("citeproc: %w", err)
	}
	return s, nil
}

func parseStyle(data []byte) (*Style, error) {
	ast, err := csl.ParseStyle(data)
	if err != nil {
		return nil, err
	}
	return &Style{ast: ast}, nil
}

// Title returns the style's title from its metadata.
func (s *Style) Title() string { return s.ast.Info.Title }

// ID returns the style's identifier URI from its metadata.
func (s *Style) ID() string { return s.ast.Info.ID }

// Warnings returns non-fatal problems found while parsing: unknown
// elements, attributes, terms or variables, which were ignored.
func (s *Style) Warnings() []*ParseError { return s.ast.Warnings }

// localeLoader returns a locale.Loader that reads "locales-<tag>.xml" from
// fsys and caches parsed locales by file content.
func localeLoader(fsys fs.FS) locale.Loader {
	return func(tag string) (*csl.Locale, error) {
		data, err := fs.ReadFile(fsys, "locales-"+tag+".xml")
		if err != nil {
			return nil, err
		}
		l, err := localeCache.get(data, csl.ParseLocale)
		if err != nil {
			return nil, fmt.Errorf("citeproc: locale %q: %w", tag, err)
		}
		return l, nil
	}
}

// Caches of parsed styles and locales, keyed by the SHA-256 of the file.
// Keying by content rather than by (fs.FS, name) works for every fs.FS
// implementation (some are not comparable) and picks up edited files. The
// cached values are immutable, so sharing them does not affect output.
var (
	styleCache  memo[*Style]
	localeCache memo[*csl.Locale]
)

type memo[T any] struct{ m sync.Map }

type memoEntry[T any] struct {
	once sync.Once
	val  T
	err  error
}

func (c *memo[T]) get(data []byte, parse func([]byte) (T, error)) (T, error) {
	key := sha256.Sum256(data)
	v, _ := c.m.LoadOrStore(key, &memoEntry[T]{})
	e := v.(*memoEntry[T])
	e.once.Do(func() { e.val, e.err = parse(data) })
	return e.val, e.err
}
