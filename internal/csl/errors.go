package csl

import (
	"fmt"
	"strings"
)

// ParseError is a problem found while parsing a style or locale file. It is
// used both for fatal errors (returned as part of an ErrorList) and for
// non-fatal warnings (Style.Warnings, Locale.Warnings).
type ParseError struct {
	Line, Col int    // 1-based position of the element's start tag; 0 if unknown
	Element   string // local element name, "" for document-level problems
	Msg       string
}

func (e *ParseError) Error() string {
	var b strings.Builder
	switch {
	case e.Line > 0 && e.Col > 0:
		fmt.Fprintf(&b, "%d:%d: ", e.Line, e.Col)
	case e.Line > 0:
		fmt.Fprintf(&b, "%d: ", e.Line)
	}
	if e.Element != "" {
		fmt.Fprintf(&b, "<%s>: ", e.Element)
	}
	b.WriteString(e.Msg)
	return b.String()
}

// ErrorList is the error returned by ParseStyle and ParseLocale. It holds
// every fatal problem found, in document order. errors.As can extract the
// first *ParseError from it.
type ErrorList []*ParseError

func (l ErrorList) Error() string {
	switch len(l) {
	case 0:
		return "no errors"
	case 1:
		return l[0].Error()
	}
	msgs := make([]string, len(l))
	for i, e := range l {
		msgs[i] = e.Error()
	}
	return fmt.Sprintf("%d errors:\n%s", len(l), strings.Join(msgs, "\n"))
}

// Unwrap returns the individual errors.
func (l ErrorList) Unwrap() []error {
	errs := make([]error, len(l))
	for i, e := range l {
		errs[i] = e
	}
	return errs
}
