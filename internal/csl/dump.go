package csl

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Dump writes a readable, deterministic tree of an AST value (typically a
// *Style or *Locale) to w. Zero-valued fields are omitted, embedded structs
// are flattened, and positions are shown as @line:col. A non-nil pointer to
// an empty value is printed, so "unset" and "set to empty" stay distinct.
// Golden tests compare this output.
func Dump(w io.Writer, v any) error {
	var d dumper
	d.value("", reflect.ValueOf(v), 0)
	_, err := io.WriteString(w, d.b.String())
	return err
}

var posType = reflect.TypeOf(Pos{})

type dumper struct{ b strings.Builder }

func (d *dumper) line(indent int, s string) {
	d.b.WriteString(strings.Repeat("  ", indent))
	d.b.WriteString(s)
	d.b.WriteByte('\n')
}

func (d *dumper) value(label string, v reflect.Value, indent int) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		header := label + v.Type().Name()
		if f, ok := v.Type().FieldByName("Pos"); ok && f.Type == posType && f.Anonymous {
			p := v.FieldByIndex(f.Index).Interface().(Pos)
			header += fmt.Sprintf(" @%d:%d", p.Line, p.Col)
		}
		d.line(indent, header)
		d.fields(v, indent+1)
	case reflect.Slice:
		if v.Len() == 0 {
			d.line(indent, label+"[]")
			return
		}
		d.line(indent, strings.TrimSuffix(label, " "))
		for i := 0; i < v.Len(); i++ {
			d.value("- ", v.Index(i), indent+1)
		}
	case reflect.Map:
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
		d.line(indent, strings.TrimSuffix(label, " "))
		for _, k := range keys {
			d.value(strconv.Quote(fmt.Sprint(k))+": ", v.MapIndex(k), indent+1)
		}
	case reflect.String:
		d.line(indent, label+strconv.Quote(v.String()))
	default:
		d.line(indent, label+fmt.Sprint(v.Interface()))
	}
}

func (d *dumper) fields(v reflect.Value, indent int) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f, fv := t.Field(i), v.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			if f.Type != posType {
				d.fields(fv, indent)
			}
			continue
		}
		if fv.IsZero() {
			continue
		}
		d.value(f.Name+": ", fv, indent)
	}
}
