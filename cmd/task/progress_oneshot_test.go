package task_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func TestProgressCommand_OneShotJSON(t *testing.T) {
	tasks := []ws.GDTask{{
		TaskID: "task-prog", Status: "running", Progress: 50,
		ReceivedBytes: 500, FileSize: 1000, Speed: 100,
	}}
	f, out := setupMockTaskCmd(t, tasks, nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "progress", "task-prog", "--format", "json"})
	require.NoError(t, root.Execute())

	var envelope struct {
		Data task.ProgressResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.Equal(t, "task-prog", envelope.Data.TaskID)
	assert.Equal(t, task.StatusRunning, envelope.Data.Status)
	assert.InDelta(t, 0.5, envelope.Data.Progress, 0.01)
	assert.Equal(t, int64(500), envelope.Data.Consumed)
	assert.Equal(t, int64(1000), envelope.Data.Total)
}
