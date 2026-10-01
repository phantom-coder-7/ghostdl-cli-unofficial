package task_test

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
)

func TestCreateCommand_CSVFormat(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"task", "create", "--url", "https://example.com/a.zip", "--format", "csv",
	})

	require.NoError(t, root.Execute())

	records, err := csv.NewReader(strings.NewReader(out.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Contains(t, records[0], "Status")
	assert.Equal(t, "created", records[1][0])
}

func TestCreateCommand_TableFormatHumanText(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"task", "create", "--url", "https://example.com/a.zip",
	})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "Task created:")
}
