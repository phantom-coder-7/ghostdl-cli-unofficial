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

func sampleListTasks() []ws.GDTask {
	return []ws.GDTask{
		{TaskID: "task-run", Name: "a.zip", Status: "running", Progress: 10, CanPause: true},
		{TaskID: "task-done", Name: "b.zip", Status: "completed", Progress: 100},
		{TaskID: "task-wait", Name: "c.zip", Status: "waiting", Progress: 0},
	}
}

func TestListCommand_TableOutput(t *testing.T) {
	f, out := setupMockTaskCmd(t, sampleListTasks(), nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list"})
	require.NoError(t, root.Execute())

	outStr := out.String()
	assert.Contains(t, outStr, "task-run")
	assert.Contains(t, outStr, "task-done")
	assert.Contains(t, outStr, "task-wait")
}

func TestListCommand_StatusFilterJSON(t *testing.T) {
	f, out := setupMockTaskCmd(t, sampleListTasks(), nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list", "--status", "running", "--format", "json"})
	require.NoError(t, root.Execute())

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Tasks []task.Task `json:"tasks"`
			Total int         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.True(t, envelope.OK)
	assert.Equal(t, 1, envelope.Data.Total)
	require.Len(t, envelope.Data.Tasks, 1)
	assert.Equal(t, "task-run", envelope.Data.Tasks[0].ID)
	assert.Equal(t, task.StatusRunning, envelope.Data.Tasks[0].Status)
}

func TestListCommand_QuerySortJSON(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "task-z", Name: "report-final.zip", Status: "completed", FileSize: 100, Progress: 100},
		{TaskID: "task-a", Name: "alpha-report.zip", Status: "completed", FileSize: 500, Progress: 100},
		{TaskID: "task-m", Name: "misc.bin", Status: "running", FileSize: 200, Progress: 10},
	}
	f, out := setupMockTaskCmd(t, tasks, nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list", "--query", "report", "--status", "completed", "--sort", "name", "--format", "json"})
	require.NoError(t, root.Execute())

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Tasks []task.Task `json:"tasks"`
			Total int         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.True(t, envelope.OK)
	assert.Equal(t, 2, envelope.Data.Total)
	require.Len(t, envelope.Data.Tasks, 2)
	assert.Equal(t, "task-a", envelope.Data.Tasks[0].ID)
	assert.Equal(t, "task-z", envelope.Data.Tasks[1].ID)
}

func TestListCommand_InvalidSort(t *testing.T) {
	f, out := setupMockTaskCmd(t, sampleListTasks(), nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list", "--sort", "bogus"})
	err := root.Execute()
	require.Error(t, err)
	assert.Equal(t, `invalid sort field "bogus" (valid options: created, id, name, pack, progress, received, size, speed, status)`, err.Error())
}

func TestListCommand_ExtFilterJSON(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "task-zip", Name: "a.zip", Status: "running", FileExt: ".zip"},
		{TaskID: "task-bin", Name: "b.bin", Status: "running", FileExt: ".bin"},
	}
	f, out := setupMockTaskCmd(t, tasks, nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list", "--ext", "zip", "--format", "json"})
	require.NoError(t, root.Execute())

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Tasks []task.Task `json:"tasks"`
			Total int         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.True(t, envelope.OK)
	assert.Equal(t, 1, envelope.Data.Total)
	require.Len(t, envelope.Data.Tasks, 1)
	assert.Equal(t, "task-zip", envelope.Data.Tasks[0].ID)
	assert.Equal(t, ".zip", envelope.Data.Tasks[0].FileExt)
}

func TestListCommand_PackFilterJSON(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "task-bt", Name: "torrent.bin", Status: "running", PackName: "bt"},
		{TaskID: "task-http", Name: "file.zip", Status: "running", PackName: "http"},
	}
	f, out := setupMockTaskCmd(t, tasks, nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list", "--pack", "bt", "--format", "json"})
	require.NoError(t, root.Execute())

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Tasks []task.Task `json:"tasks"`
			Total int         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.True(t, envelope.OK)
	assert.Equal(t, 1, envelope.Data.Total)
	require.Len(t, envelope.Data.Tasks, 1)
	assert.Equal(t, "task-bt", envelope.Data.Tasks[0].ID)
	assert.Equal(t, "bt", envelope.Data.Tasks[0].PackName)
}

func TestListCommand_InvalidOrder(t *testing.T) {
	f, out := setupMockTaskCmd(t, sampleListTasks(), nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list", "--order", "sideways"})
	err := root.Execute()
	require.Error(t, err)
	assert.Equal(t, `invalid sort order "sideways" (valid options: asc, desc)`, err.Error())
}

func TestListCommand_MultipleStatusesJSON(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "task-run", Name: "a.zip", Status: "running", Progress: 10},
		{TaskID: "task-pause", Name: "b.zip", Status: "paused", Progress: 50},
		{TaskID: "task-done", Name: "c.zip", Status: "completed", Progress: 100},
	}
	f, out := setupMockTaskCmd(t, tasks, nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list", "--status", "running,paused", "--format", "json"})
	require.NoError(t, root.Execute())

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Tasks []task.Task `json:"tasks"`
			Total int         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.True(t, envelope.OK)
	assert.Equal(t, 2, envelope.Data.Total)
	require.Len(t, envelope.Data.Tasks, 2)
	ids := []string{envelope.Data.Tasks[0].ID, envelope.Data.Tasks[1].ID}
	assert.ElementsMatch(t, []string{"task-run", "task-pause"}, ids)
}

func TestListCommand_RepeatedStatusFlagsJSON(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "task-run", Name: "a.zip", Status: "running"},
		{TaskID: "task-pause", Name: "b.zip", Status: "paused"},
		{TaskID: "task-done", Name: "c.zip", Status: "completed"},
	}
	f, out := setupMockTaskCmd(t, tasks, nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "list", "--status", "running", "--status", "paused", "--format", "json"})
	require.NoError(t, root.Execute())

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Tasks []task.Task `json:"tasks"`
			Total int         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.Equal(t, 2, envelope.Data.Total)
}
