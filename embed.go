package citeproc

import (
	"embed"
	"io/fs"
)

//go:embed styles/*.csl
var embeddedStyles embed.FS

//go:embed locales/*.xml
var embeddedLocales embed.FS

// Styles holds the bundled CSL styles, one file per style named
// "<name>.csl" (for example "apa.csl") at the root of the file system.
var Styles fs.FS = mustSub(embeddedStyles, "styles")

// Locales holds the bundled CSL locale files, named "locales-<tag>.xml"
// (for example "locales-en-US.xml") at the root of the file system.
var Locales fs.FS = mustSub(embeddedLocales, "locales")

// mustSub panics if dir is missing, which can only happen if the go:embed
// patterns above and the directory names disagree.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic("citeproc: embedded " + dir + ": " + err.Error())
	}
	return sub
}
