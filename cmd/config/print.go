package config

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// printSettings writes cfg.List()-shaped maps with top-level and nested keys
// in sorted (alphabetical) order, then the config file path.
func printSettings(w io.Writer, settings map[string]any, path string) {
	keys := sortedMapKeys(settings)
	for _, k := range keys {
		fmt.Fprintf(w, "  %-30s = %s\n", k, formatConfigValue(settings[k]))
	}
	fmt.Fprintf(w, "\nConfig file: %s\n", path)
}

func formatConfigValue(v any) string {
	if m, ok := v.(map[string]any); ok {
		return formatSortedMap(m)
	}
	return fmt.Sprint(v)
}

// formatSortedMap renders a map like Go's %v for map[string]any, but with
// keys in sorted order (including nested maps).
func formatSortedMap(m map[string]any) string {
	keys := sortedMapKeys(m)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+":"+formatConfigValue(m[k]))
	}
	return "map[" + strings.Join(parts, " ") + "]"
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
