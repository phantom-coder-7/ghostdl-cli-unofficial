package output

import (
	"fmt"
	"reflect"
	"sort"
	"text/tabwriter"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
)

func (f *Formatter) printList(data any) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Struct:
		return f.writeListStruct(rv)
	case reflect.Slice, reflect.Array:
		return f.writeListSlice(rv)
	case reflect.Map:
		return f.writeListMap(rv)
	default:
		fmt.Fprintln(f.w, display.Sanitize(fmt.Sprint(data)))
		return nil
	}
}

func (f *Formatter) writeListStruct(rv reflect.Value) error {
	fields := selectFields(rv.Type(), f.detail)
	w := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
	defer w.Flush()

	for _, field := range fields {
		fmt.Fprintf(w, "%s:\t%s\n", field.label, formatFieldValue(rv.Field(field.index)))
	}
	return nil
}

func (f *Formatter) writeListSlice(rv reflect.Value) error {
	if rv.Len() == 0 {
		fmt.Fprintln(f.w, "(empty)")
		return nil
	}

	for i := 0; i < rv.Len(); i++ {
		if i > 0 {
			fmt.Fprintln(f.w)
		}
		item := rv.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		if err := f.writeListStruct(item); err != nil {
			return err
		}
	}
	return nil
}

func (f *Formatter) writeListMap(rv reflect.Value) error {
	w := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
	defer w.Flush()

	keys := rv.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})

	for _, key := range keys {
		fmt.Fprintf(w, "%s:\t%s\n",
			display.Sanitize(key.String()),
			display.Sanitize(fmt.Sprint(rv.MapIndex(key).Interface())))
	}
	return nil
}
