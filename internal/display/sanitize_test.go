package display

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func hostileDisplayPayload() string {
	return "ok" +
		string([]byte{0x1B, 0x5B, 0x31, 0x6D}) +
		string([]byte{0x1B, 0x5D, 0x30, 0x3B}) +
		"\r\n\t\a" +
		string(rune(0x7F)) +
		string(rune(0x9B)) +
		"\u2028\u2029" +
		"\u202A\u202B\u202C\u202D\u202E" +
		"\u2066\u2067\u2068\u2069" +
		"end"
}

func TestSanitize_ReplacesControlsWithFFFD(t *testing.T) {
	in := hostileDisplayPayload()
	out := Sanitize(in)

	assert.NotContains(t, out, "\x1b")
	assert.NotContains(t, out, "\r")
	assert.NotContains(t, out, "\n")
	assert.NotContains(t, out, "\t")
	assert.NotContains(t, out, "\a")
	assert.NotContains(t, out, string(rune(0x7F)))
	assert.NotContains(t, out, string(rune(0x9B)))
	assert.NotContains(t, out, "\u2028")
	assert.NotContains(t, out, "\u2029")
	assert.NotContains(t, out, "\u202A")
	assert.NotContains(t, out, "\u2066")
	assert.True(t, strings.HasPrefix(out, "ok"))
	assert.True(t, strings.HasSuffix(out, "end"))
	assert.Contains(t, out, "\uFFFD")
	assert.Equal(t, 0, strings.Count(out, "\n"))
	assert.Equal(t, 0, strings.Count(out, "\r"))
	assert.True(t, utf8.ValidString(out))
}

func TestSanitize_EmptyAndSafe(t *testing.T) {
	assert.Equal(t, "", Sanitize(""))
	assert.Equal(t, "hello.zip", Sanitize("hello.zip"))
	assert.Equal(t, "正常名", Sanitize("正常名"))
}
