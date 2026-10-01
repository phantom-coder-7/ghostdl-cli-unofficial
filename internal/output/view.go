package output

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
)

// renderField holds metadata for a struct field selected for output.
type renderField struct {
	index int
	label string
}

// selectFields returns ordered exported struct field indices to render.
// When detail is false and the type has view tags, only view:"overview" fields
// are included. When detail is true or no field carries a view tag, all exported
// fields are included.
func selectFields(typ reflect.Type, detail bool) []renderField {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	hasViewTags := false
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.IsExported() && field.Tag.Get("view") != "" {
			hasViewTags = true
			break
		}
	}

	var fields []renderField
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		if hasViewTags && !detail && field.Tag.Get("view") != "overview" {
			continue
		}
		label := field.Tag.Get("table")
		if label == "" {
			label = field.Name
		}
		fields = append(fields, renderField{index: i, label: label})
	}
	return fields
}

func formatFieldValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	switch v.Kind() {
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Float32:
		return strconv.FormatFloat(float64(v.Float()), 'f', -1, 32)
	default:
		return display.Sanitize(fmt.Sprintf("%v", v.Interface()))
	}
}

func formatCSVFieldValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	switch v.Kind() {
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', 1, 64)
	case reflect.Float32:
		return strconv.FormatFloat(float64(v.Float()), 'f', 1, 32)
	default:
		return display.Sanitize(fmt.Sprintf("%v", v.Interface()))
	}
}
