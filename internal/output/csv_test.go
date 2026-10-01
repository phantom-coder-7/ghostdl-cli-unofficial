package output

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func parseCSVOutput(t *testing.T, out string) [][]string {
	t.Helper()
	r := csv.NewReader(strings.NewReader(out))
	records, err := r.ReadAll()
	require.NoError(t, err)
	return records
}

func TestCSV_StructOverview(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "csv", false).Print(sampleTask()))
	records := parseCSVOutput(t, buf.String())
	require.Len(t, records, 2)
	assert.Equal(t, []string{"ID", "Name", "Status", "Progress%", "Size"}, records[0])
	assert.Equal(t, "task-1", records[1][0])
	assert.Equal(t, "50.5", records[1][3])
}

func TestCSV_StructDetail(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "csv", true).Print(sampleTask()))
	records := parseCSVOutput(t, buf.String())
	require.Len(t, records, 2)
	assert.Len(t, records[0], 14)
	assert.Contains(t, records[0], "Speed")
	assert.Contains(t, records[0], "URL")
}

func TestCSV_SliceMultipleRows(t *testing.T) {
	tasks := []task.Task{sampleTask(), {ID: "task-2", Name: "other", Status: task.StatusPaused, FileSize: 2048}}
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "csv", false).Print(tasks))
	records := parseCSVOutput(t, buf.String())
	require.Len(t, records, 3)
}

func TestCSV_EmptySliceHeaderOnly(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "csv", false).Print([]task.Task{}))
	records := parseCSVOutput(t, buf.String())
	require.Len(t, records, 1)
	assert.Equal(t, []string{"ID", "Name", "Status", "Progress%", "Size"}, records[0])
}

func TestCSV_NameWithCommaAndQuotes(t *testing.T) {
	task := sampleTask()
	task.Name = `file, "special".zip`
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "csv", false).Print(task))
	records := parseCSVOutput(t, buf.String())
	require.Len(t, records, 2)
	assert.Equal(t, `file, "special".zip`, records[1][1])

	assert.Contains(t, buf.String(), `"`)
}

func TestCSV_ScalarFallback(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "csv", false).Print("hello"))
	records := parseCSVOutput(t, buf.String())
	require.Len(t, records, 1)
	assert.Equal(t, []string{"hello"}, records[0])
}

func TestCSV_Map(t *testing.T) {
	data := map[string]string{"name": "gd", "commit": "abc1234"}
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "csv", false).Print(data))
	records := parseCSVOutput(t, buf.String())
	require.Len(t, records, 3)
	assert.Equal(t, []string{"key", "value"}, records[0])
}

func TestCSV_ProgressDeterministic(t *testing.T) {
	task := sampleTask()
	task.Progress = 33.333333
	var buf bytes.Buffer
	require.NoError(t, NewFormatterWithView(&buf, "csv", false).Print(task))
	records := parseCSVOutput(t, buf.String())
	require.Len(t, records, 2)
	assert.Equal(t, "33.3", records[1][3])
}
