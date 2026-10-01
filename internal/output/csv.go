package output

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"sort"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
)

func (f *Formatter) printCSV(data any) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Struct:
		return f.writeCSVStruct(rv)
	case reflect.Slice, reflect.Array:
		return f.writeCSVSlice(rv)
	case reflect.Map:
		return f.writeCSVMap(rv)
	default:
		w := csv.NewWriter(f.w)
		if err := w.Write([]string{display.Sanitize(fmt.Sprintf("%v", data))}); err != nil {
			return err
		}
		w.Flush()
		return w.Error()
	}
}

func (f *Formatter) writeCSVStruct(rv reflect.Value) error {
	fields := selectFields(rv.Type(), f.detail)
	w := csv.NewWriter(f.w)

	header := make([]string, len(fields))
	row := make([]string, len(fields))
	for i, field := range fields {
		header[i] = field.label
		row[i] = formatCSVFieldValue(rv.Field(field.index))
	}

	if err := w.Write(header); err != nil {
		return err
	}
	if err := w.Write(row); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func (f *Formatter) writeCSVSlice(rv reflect.Value) error {
	w := csv.NewWriter(f.w)

	if rv.Len() == 0 {

		elemType := rv.Type().Elem()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		if elemType.Kind() == reflect.Struct {
			fields := selectFields(elemType, f.detail)
			header := make([]string, len(fields))
			for i, field := range fields {
				header[i] = field.label
			}
			if err := w.Write(header); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	}

	first := rv.Index(0)
	if first.Kind() == reflect.Ptr {
		first = first.Elem()
	}
	fields := selectFields(first.Type(), f.detail)

	header := make([]string, len(fields))
	for i, field := range fields {
		header[i] = field.label
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		row := make([]string, len(fields))
		for j, field := range fields {
			row[j] = formatCSVFieldValue(item.Field(field.index))
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func (f *Formatter) writeCSVMap(rv reflect.Value) error {
	w := csv.NewWriter(f.w)

	if err := w.Write([]string{"key", "value"}); err != nil {
		return err
	}

	keys := rv.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})

	for _, key := range keys {
		if err := w.Write([]string{
			display.Sanitize(key.String()),
			display.Sanitize(fmt.Sprintf("%v", rv.MapIndex(key).Interface())),
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
