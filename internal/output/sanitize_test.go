package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

// hostileDisplayPayload includes ESC, OSC-like bytes, CR, LF, TAB, BEL, DEL,
// a C1 control, line/paragraph separators, and bidi spoofing runes.
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

func TestHumanFormats_SanitizeServerStrings(t *testing.T) {
	hostile := hostileDisplayPayload()
	tk := sampleTask()
	tk.Name = hostile
	tk.URL = hostile
	tk.Status = task.Status(hostile)
	tk.ID = hostile
	tk.PackName = hostile
	tk.FileExt = hostile

	for _, format := range []string{"table", "list", "csv"} {
		t.Run(format, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, NewFormatterWithView(&buf, format, true).Print(tk))
			out := buf.String()
			assert.NotContains(t, out, "\x1b")
			for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
				assert.NotContains(t, line, "\r")
				assert.NotContains(t, line, "\x1b")
			}
			assert.Contains(t, out, "\uFFFD")
		})
	}
}

func TestJSONAndNDJSON_PreserveOriginalControls(t *testing.T) {
	hostile := hostileDisplayPayload()
	tk := sampleTask()
	tk.Name = hostile

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, NewFormatterWithView(&buf, "json", false).Print(tk))
		var envelope struct {
			OK   bool      `json:"ok"`
			Data task.Task `json:"data"`
		}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
		assert.Equal(t, hostile, envelope.Data.Name)
		assert.Contains(t, buf.String(), `\u001b`)
	})

	t.Run("ndjson", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, NewFormatterWithView(&buf, "ndjson", false).Print([]task.Task{tk}))
		var decoded task.Task
		require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
		assert.Equal(t, hostile, decoded.Name)
		assert.Contains(t, buf.String(), `\u001b`)
	})
}

func TestActionableError_SanitizesServerMessageOnDisplay(t *testing.T) {
	hostile := hostileDisplayPayload()
	err := MapGDServerMessage(hostile)
	assert.Equal(t, hostile, err.Message)
	assert.Equal(t, hostile, err.ServerMessage)

	rendered := err.Error()
	assert.NotContains(t, rendered, "\x1b")
	assert.NotContains(t, rendered, "\r")
	assert.Contains(t, rendered, "\uFFFD")
	assert.True(t, utf8.ValidString(rendered))
}
