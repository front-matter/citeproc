package citeproc

import (
	"io/fs"
	"testing"
)

func TestEmbeddedFiles(t *testing.T) {
	tests := []struct {
		name  string
		fsys  fs.FS
		files []string
	}{
		{"Styles", Styles, []string{"apa.csl", "modern-language-association.csl", "vancouver.csl"}},
		{"Locales", Locales, []string{"locales-en-US.xml", "locales-de-DE.xml", "locales-es-ES.xml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, name := range tt.files {
				data, err := fs.ReadFile(tt.fsys, name)
				if err != nil {
					t.Errorf("ReadFile(%q): %v", name, err)
					continue
				}
				if len(data) == 0 {
					t.Errorf("ReadFile(%q): empty file", name)
				}
			}
		})
	}
}
