package api

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func taskWithNumericCreatedAt(tt *testing.T, loc *time.Location) *task.Task {
	const raw = `{"taskId":"parity","name":"file.zip","status":"running","progress":1,` +
		`"receivedBytes":0,"fileSize":100,"speed":0,"createdAt":1736928000}`
	var wireTask ws.GDTask
	require.NoError(tt, json.Unmarshal([]byte(raw), &wireTask))
	return convertGDTask(&wireTask, loc)
}

func renderTaskAllFormats(tt *testing.T, tsk *task.Task, detail bool) map[string]string {
	formats := []string{"json", "ndjson", "table", "csv", "list"}
	out := make(map[string]string, len(formats))
	for _, format := range formats {
		var buf bytes.Buffer
		require.NoError(tt, output.NewFormatterWithView(&buf, format, detail).Print(tsk))
		out[format] = buf.String()
	}
	return out
}

func TestCreatedAtParity_AllFormats_FixedZone(t *testing.T) {
	loc := time.FixedZone("+08:00", 8*60*60)
	tsk := taskWithNumericCreatedAt(t, loc)
	want := tsk.CreatedAt
	assert.Equal(t, "2025-01-15T16:00:00+08:00", want)

	outs := renderTaskAllFormats(t, tsk, true)
	for format, s := range outs {
		assert.Contains(t, s, want, "format %q should contain createdAt %q", format, want)
	}
}

func TestCreatedAtParity_AllFormats_UTC(t *testing.T) {
	tsk := taskWithNumericCreatedAt(t, time.UTC)
	want := "2025-01-15T08:00:00Z"
	assert.Equal(t, want, tsk.CreatedAt)

	outs := renderTaskAllFormats(t, tsk, true)
	for format, s := range outs {
		assert.Contains(t, s, want, "format %q should contain createdAt %q", format, want)
		assert.False(t, strings.Contains(s, "+08:00"), "format %q should not contain local offset", format)
	}
}
