package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func TestList_StructOverview(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "list", false).Print(sampleTask()))
	out := buf.String()
	assert.Contains(t, out, "ID:")
	assert.Contains(t, out, "task-1")
	assert.Contains(t, out, "Name:")
	assert.Contains(t, out, "file.zip")
	assert.NotContains(t, out, "Speed:")
}

func TestList_StructDetail(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "list", true).Print(sampleTask()))
	out := buf.String()
	assert.Contains(t, out, "Speed:")
	assert.Contains(t, out, "100")
	assert.Contains(t, out, "URL:")
	assert.Contains(t, out, "https://example.com/file.zip")
}

func TestList_SliceBlankLineBetweenRecords(t *testing.T) {
	tasks := []task.Task{sampleTask(), {ID: "task-2", Name: "other", Status: task.StatusPaused}}
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "list", false).Print(tasks))
	parts := strings.Split(strings.TrimSpace(buf.String()), "\n\n")
	assert.Len(t, parts, 2)
}

func TestList_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "list", false).Print([]task.Task{}))
	assert.Equal(t, "(empty)\n", buf.String())
}

func TestList_NoViewTagsRendersAllFields(t *testing.T) {
	data := noViewTags{Alpha: "a", Beta: "b"}
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "list", false).Print(data))
	assert.Contains(t, buf.String(), "Alpha:")
	assert.Contains(t, buf.String(), "a")
	assert.Contains(t, buf.String(), "Beta:")
	assert.Contains(t, buf.String(), "b")
}

func TestList_Map(t *testing.T) {
	data := map[string]string{"name": "gd", "commit": "abc1234"}
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "list", false).Print(data))
	assert.Contains(t, buf.String(), "commit:")
	assert.Contains(t, buf.String(), "abc1234")
	assert.Contains(t, buf.String(), "name:")
	assert.Contains(t, buf.String(), "gd")
}
