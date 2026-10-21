package csl

import (
	"bytes"
	"testing"
)

// The fuzz targets check that arbitrary input never panics, and that a
// successfully parsed value can be dumped.
//
//	go test ./internal/csl -run '^$' -fuzz FuzzParseStyle -fuzztime 10m

func FuzzParseStyle(f *testing.F) {
	for _, name := range bundledStyles {
		f.Add(readStyle(f, name))
	}
	f.Add(minimal(`<macro name="a"><text macro="a"/></macro>` + layout))
	f.Add(minimal(`<citation><layout><choose><if variable="title"><names variable="author"><name/><substitute><text macro="x"/></substitute></names></if><else/></choose></layout></citation>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := ParseStyle(data)
		if err != nil {
			return
		}
		if err := Dump(&bytes.Buffer{}, s); err != nil {
			t.Fatal(err)
		}
	})
}

func FuzzParseLocale(f *testing.F) {
	for _, tag := range bundledLocales {
		f.Add(readLocale(f, tag))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		l, err := ParseLocale(data)
		if err != nil {
			return
		}
		if err := Dump(&bytes.Buffer{}, l); err != nil {
			t.Fatal(err)
		}
	})
}
