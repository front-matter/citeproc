package csl

import (
	"fmt"
	"strings"
)

// Walk calls fn for every element in els and, recursively, for the elements
// nested in groups, choose branches and substitutes. It stops descending into
// an element when fn returns false.
func Walk(els []Element, fn func(Element) bool) {
	for _, el := range els {
		if !fn(el) {
			continue
		}
		switch el := el.(type) {
		case *Group:
			Walk(el.Children, fn)
		case *Choose:
			for _, b := range el.Branches {
				Walk(b.Children, fn)
			}
			Walk(el.Else, fn)
		case *Names:
			if el.Substitute != nil {
				Walk(el.Substitute.Children, fn)
			}
		}
	}
}

// checkMacros reports calls to undefined macros and macro cycles. Both are
// fatal: an undefined macro is a schema (Schematron) violation, and a cycle
// would make rendering recurse forever.
func (p *parser) checkMacros(s *Style) {
	callsIn := func(els []Element) []*Text {
		var calls []*Text
		Walk(els, func(el Element) bool {
			if t, ok := el.(*Text); ok && t.Source == TextMacro {
				calls = append(calls, t)
			}
			return true
		})
		return calls
	}
	undefined := func(pos Pos, elem, name string) {
		p.errs = append(p.errs, &ParseError{Line: pos.Line, Col: pos.Col, Element: elem,
			Msg: fmt.Sprintf("call to undefined macro %q", name)})
	}
	checkCalls := func(els []Element) {
		for _, t := range callsIn(els) {
			if _, ok := s.Macros[t.Name]; !ok {
				undefined(t.Pos, "text", t.Name)
			}
		}
	}
	checkSort := func(sort *Sort) {
		if sort == nil {
			return
		}
		for _, k := range sort.Keys {
			if k.Macro != "" {
				if _, ok := s.Macros[k.Macro]; !ok {
					undefined(k.Pos, "key", k.Macro)
				}
			}
		}
	}

	for _, name := range s.MacroOrder {
		checkCalls(s.Macros[name].Children)
	}
	if c := s.Citation; c != nil {
		checkCalls(c.Layout.Children)
		checkSort(c.Sort)
	}
	if b := s.Bibliography; b != nil {
		checkCalls(b.Layout.Children)
		checkSort(b.Sort)
	}

	// Cycle detection: depth-first search over the macro call graph. Each
	// cycle is reported once, at the call that closes it.
	const (
		unvisited = iota
		active
		done
	)
	state := map[string]int{}
	var path []string
	var visit func(name string)
	visit = func(name string) {
		state[name] = active
		path = append(path, name)
		for _, call := range callsIn(s.Macros[name].Children) {
			m, ok := s.Macros[call.Name]
			if !ok {
				continue // already reported as undefined
			}
			switch state[m.Name] {
			case unvisited:
				visit(m.Name)
			case active:
				start := 0
				for i, n := range path {
					if n == m.Name {
						start = i
					}
				}
				cycle := append(append([]string{}, path[start:]...), m.Name)
				p.errs = append(p.errs, &ParseError{Line: call.Line, Col: call.Col, Element: "text",
					Msg: "macro cycle: " + strings.Join(cycle, " → ")})
			}
		}
		path = path[:len(path)-1]
		state[name] = done
	}
	for _, name := range s.MacroOrder {
		if state[name] == unvisited {
			visit(name)
		}
	}
}
