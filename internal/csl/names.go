package csl

// NameOptions are the attributes of cs:name. On cs:style, cs:citation and
// cs:bibliography the same options are set through the inheritable name
// attributes, where Form and Delimiter are spelled name-form and
// name-delimiter.
//
// Nil pointers and empty enums mean "unset". Inherit fills unset fields from
// the enclosing scope; the renderer applies the schema defaults (documented
// on each field) to whatever is still unset.
type NameOptions struct {
	And                    NameAnd       // default: no "and"
	DelimiterPrecedesEtAl  DelimiterRule // default "contextual"
	DelimiterPrecedesLast  DelimiterRule // default "contextual"
	EtAlMin                *int
	EtAlUseFirst           *int
	EtAlSubsequentMin      *int
	EtAlSubsequentUseFirst *int
	EtAlUseLast            *bool // default false
	Initialize             *bool // default true
	InitializeWith         *string
	NameAsSortOrder        NameAsSortOrder // default: no inversion
	SortSeparator          *string         // default ", "
	Form                   NameForm        // default "long"
	Delimiter              *string         // default ", "
}

// Inherit returns o with every unset field taken from parent.
func (o NameOptions) Inherit(parent NameOptions) NameOptions {
	r := o
	if r.And == "" {
		r.And = parent.And
	}
	if r.DelimiterPrecedesEtAl == "" {
		r.DelimiterPrecedesEtAl = parent.DelimiterPrecedesEtAl
	}
	if r.DelimiterPrecedesLast == "" {
		r.DelimiterPrecedesLast = parent.DelimiterPrecedesLast
	}
	inheritPtr(&r.EtAlMin, parent.EtAlMin)
	inheritPtr(&r.EtAlUseFirst, parent.EtAlUseFirst)
	inheritPtr(&r.EtAlSubsequentMin, parent.EtAlSubsequentMin)
	inheritPtr(&r.EtAlSubsequentUseFirst, parent.EtAlSubsequentUseFirst)
	inheritPtr(&r.EtAlUseLast, parent.EtAlUseLast)
	inheritPtr(&r.Initialize, parent.Initialize)
	inheritPtr(&r.InitializeWith, parent.InitializeWith)
	if r.NameAsSortOrder == "" {
		r.NameAsSortOrder = parent.NameAsSortOrder
	}
	inheritPtr(&r.SortSeparator, parent.SortSeparator)
	if r.Form == "" {
		r.Form = parent.Form
	}
	inheritPtr(&r.Delimiter, parent.Delimiter)
	return r
}

// InheritableNameOptions are the inheritable name attributes of cs:style,
// cs:citation and cs:bibliography.
type InheritableNameOptions struct {
	NameOptions
	// NamesDelimiter (names-delimiter) is the default delimiter of cs:names;
	// nil if unset.
	NamesDelimiter *string
}

// Inherit returns o with every unset field taken from parent.
func (o InheritableNameOptions) Inherit(parent InheritableNameOptions) InheritableNameOptions {
	r := InheritableNameOptions{NameOptions: o.NameOptions.Inherit(parent.NameOptions), NamesDelimiter: o.NamesDelimiter}
	inheritPtr(&r.NamesDelimiter, parent.NamesDelimiter)
	return r
}

// NameOptionsIn returns the options of a cs:name inside n, given the options
// in scope (from Style.CitationNameOptions or Style.BibliographyNameOptions).
// A cs:names without cs:name uses the scope's options unchanged.
func (n *Names) NameOptionsIn(scope InheritableNameOptions) NameOptions {
	if n.Name == nil {
		return scope.NameOptions
	}
	return n.Name.Options.Inherit(scope.NameOptions)
}

// DelimiterIn returns the delimiter between the variables of n, given the
// options in scope: the cs:names delimiter, then names-delimiter, then "".
func (n *Names) DelimiterIn(scope InheritableNameOptions) string {
	switch {
	case n.Delimiter != nil:
		return *n.Delimiter
	case scope.NamesDelimiter != nil:
		return *scope.NamesDelimiter
	}
	return ""
}

func inheritPtr[T any](dst **T, parent *T) {
	if *dst == nil {
		*dst = parent
	}
}
