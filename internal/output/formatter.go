package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
)

var validFormats = []string{"table", "json", "ndjson", "csv", "list"}

func ValidFormats() []string {
	out := make([]string, len(validFormats))
	copy(out, validFormats)
	return out
}

func ValidateFormat(format string) error {
	for _, v := range validFormats {
		if format == v {
			return nil
		}
	}
	return fmt.Errorf("invalid output format %q (valid options: %s)", format, strings.Join(validFormats, ", "))
}

type Formatter struct {
	w      io.Writer
	format string
	detail bool
}

func NewFormatter(w io.Writer, format string) *Formatter {
	return NewFormatterWithView(w, format, false)
}

func NewFormatterWithView(w io.Writer, format string, detail bool) *Formatter {
	return &Formatter{w: w, format: format, detail: detail}
}

func (f *Formatter) Print(data any) error {
	switch f.format {
	case "json":
		return f.printJSON(data)
	case "ndjson":
		return f.printNDJSON(data)
	case "csv":
		return f.printCSV(data)
	case "list":
		return f.printList(data)
	default:
		return f.printTable(data)
	}
}

type jsonEnvelope struct {
	OK   bool `json:"ok"`
	Data any  `json:"data"`
}

func (f *Formatter) printJSON(data any) error {
	enc := json.NewEncoder(f.w)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonEnvelope{OK: true, Data: data})
}

func (f *Formatter) printNDJSON(data any) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			b, err := json.Marshal(rv.Index(i).Interface())
			if err != nil {
				return err
			}
			fmt.Fprintln(f.w, string(b))
		}
		return nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Fprintln(f.w, string(b))
	return nil
}

func (f *Formatter) printTable(data any) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Struct {
		return f.printStructTable(rv)
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return f.printSliceTable(rv)
	}
	if rv.Kind() == reflect.Map {
		return f.printMapTable(rv)
	}

	fmt.Fprintln(f.w, display.Sanitize(fmt.Sprint(data)))
	return nil
}

func (f *Formatter) printStructTable(rv reflect.Value) error {
	fields := selectFields(rv.Type(), f.detail)
	w := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
	defer w.Flush()

	for i, field := range fields {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, field.label)
	}
	fmt.Fprintln(w)

	for i, field := range fields {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, formatFieldValue(rv.Field(field.index)))
	}
	fmt.Fprintln(w)

	return nil
}

func (f *Formatter) printSliceTable(rv reflect.Value) error {
	if rv.Len() == 0 {
		fmt.Fprintln(f.w, "(empty)")
		return nil
	}

	first := rv.Index(0)
	if first.Kind() == reflect.Ptr {
		first = first.Elem()
	}
	fields := selectFields(first.Type(), f.detail)

	w := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
	defer w.Flush()

	for i, field := range fields {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, field.label)
	}
	fmt.Fprintln(w)

	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		for j, field := range fields {
			if j > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, formatFieldValue(item.Field(field.index)))
		}
		fmt.Fprintln(w)
	}

	return nil
}

func (f *Formatter) printMapTable(rv reflect.Value) error {
	w := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
	defer w.Flush()

	for _, key := range rv.MapKeys() {
		fmt.Fprintf(w, "  %-30s\t= %s\n",
			display.Sanitize(key.String()),
			display.Sanitize(fmt.Sprint(rv.MapIndex(key).Interface())))
	}
	return nil
}
