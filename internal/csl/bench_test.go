package csl

import "testing"

// BenchmarkParseStyle measures a cold parse (XML decoding, conversion and
// validation) of each bundled style. The "xml" sub-benchmark isolates the
// encoding/xml tokenizer cost for APA.
func BenchmarkParseStyle(b *testing.B) {
	for _, name := range bundledStyles {
		data := readStyle(b, name)
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := ParseStyle(data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
	data := readStyle(b, "apa")
	b.Run("apa/xml", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := readTree(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkParseLocale(b *testing.B) {
	for _, tag := range bundledLocales {
		data := readLocale(b, tag)
		b.Run(tag, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := ParseLocale(data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
