package display

import "strings"

const replacement = '\uFFFD'

// Sanitize replaces control and bidi-spoofing runes in text destined for a
// human-facing stdout/stderr stream. JSON/ndjson encoders must marshal original
// strings and must not call this.
func Sanitize(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isUnsafeDisplayRune(r) {
			b.WriteRune(replacement)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isUnsafeDisplayRune(r rune) bool {
	switch {
	case r <= 0x1F:
		return true
	case r == 0x7F:
		return true
	case r >= 0x80 && r <= 0x9F:
		return true
	case r == 0x2028 || r == 0x2029:
		return true
	case r >= 0x202A && r <= 0x202E:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	default:
		return false
	}
}
