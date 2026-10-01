package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func TestValidateFormat_AcceptsValidFormats(t *testing.T) {
	for _, format := range ValidFormats() {
		require.NoError(t, ValidateFormat(format))
	}
}

func TestValidateFormat_RejectsInvalidFormat(t *testing.T) {
	err := ValidateFormat("bogus")
	require.Error(t, err)
	assert.Equal(t, `invalid output format "bogus" (valid options: table, json, ndjson, csv, list)`, err.Error())
}

type overviewOnly struct {
	Shown   string `table:"Shown" view:"overview"`
	Hidden  string `table:"Hidden"`
	AlsoAll string `table:"Also All"`
}

type noViewTags struct {
	Alpha string `table:"Alpha"`
	Beta  string `table:"Beta"`
}

func sampleTask() task.Task {
	return task.Task{
		ID:            "task-1",
		Name:          "file.zip",
		Status:        task.StatusRunning,
		Progress:      50.5,
		ReceivedBytes: 512,
		FileSize:      1024,
		Speed:         100,
		CreatedAt:     "2025-01-15T10:00:00",
		CanPause:      true,
		CanOpenFile:   false,
		CanOpenFolder: true,
		FileExt:       ".zip",
		PackName:      "pack-a",
		URL:           "https://example.com/file.zip",
	}
}

func countTableColumns(out string) int {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		return 0
	}
	if strings.Contains(lines[0], "\t") {
		return len(strings.Split(lines[0], "\t"))
	}
	return len(strings.Fields(lines[0]))
}

func TestPrintTable_StructOverview(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "table", false).Print(sampleTask()))
	assert.Equal(t, 5, countTableColumns(buf.String()))
	assert.Contains(t, buf.String(), "ID")
	assert.NotContains(t, buf.String(), "Speed")
}

func TestPrintTable_StructDetail(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "table", true).Print(sampleTask()))
	out := buf.String()
	assert.Contains(t, out, "Speed")
	assert.Contains(t, out, "URL")
	assert.Contains(t, out, "Received")
	assert.Contains(t, out, "Created At")
}

func TestPrintTable_SliceOverview(t *testing.T) {
	tasks := []task.Task{sampleTask(), sampleTask()}
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "table", false).Print(tasks))
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, 5, countTableColumns(buf.String()))
}

func TestPrintTable_SliceDetail(t *testing.T) {
	tasks := []task.Task{sampleTask()}
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "table", true).Print(tasks))
	out := buf.String()
	assert.Contains(t, out, "Speed")
	assert.Contains(t, out, "URL")
}

func TestPrintTable_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "table", false).Print([]task.Task{}))
	assert.Equal(t, "(empty)\n", buf.String())
}

func TestPrintTable_NoViewTagsRendersAllFields(t *testing.T) {
	data := noViewTags{Alpha: "a", Beta: "b"}

	var overview bytes.Buffer
	require.NoError(t, NewFormatterWithView(&overview, "table", false).Print(data))
	assert.Equal(t, 2, countTableColumns(overview.String()))

	var detail bytes.Buffer
	require.NoError(t, NewFormatterWithView(&detail, "table", true).Print(data))
	assert.Equal(t, 2, countTableColumns(detail.String()))
}

func TestPrintTable_ViewTagFiltering(t *testing.T) {
	data := overviewOnly{Shown: "yes", Hidden: "no", AlsoAll: "all"}

	var overview bytes.Buffer
	require.NoError(t, NewFormatterWithView(&overview, "table", false).Print(data))
	assert.Contains(t, overview.String(), "Shown")
	assert.NotContains(t, overview.String(), "Hidden")
	assert.NotContains(t, overview.String(), "Also All")

	var detail bytes.Buffer
	require.NoError(t, NewFormatterWithView(&detail, "table", true).Print(data))
	assert.Contains(t, detail.String(), "Shown")
	assert.Contains(t, detail.String(), "Hidden")
	assert.Contains(t, detail.String(), "Also All")
}

func TestPrintNDJSON_MarshalError(t *testing.T) {
	var buf bytes.Buffer
	err := NewFormatter(&buf, "ndjson").Print(map[string]any{"bad": make(chan int)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json: unsupported type")
}

func TestNewFormatterDefaultsToOverview(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatter(&buf, "table").Print(sampleTask()))
	assert.Equal(t, 5, countTableColumns(buf.String()))
}
