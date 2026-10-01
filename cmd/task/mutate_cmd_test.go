package task_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func TestPauseCommand_SendsTogglePause(t *testing.T) {
	tasks := []ws.GDTask{{
		TaskID: "task-run", Status: "running", CanPause: true,
	}}
	rec := &recordedTaskActions{}
	f, out := setupMockTaskCmd(t, tasks, rec)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "pause", "task-run"})
	require.NoError(t, root.Execute())

	assert.Contains(t, out.String(), "paused")
	assert.Equal(t, ws.ActionTogglePause, rec.lastAction())
}

func TestOpenCommand_SendsOpenFile(t *testing.T) {
	tasks := []ws.GDTask{{
		TaskID: "task-open", Status: "completed", Progress: 100, CanOpenFile: true,
	}}
	rec := &recordedTaskActions{}
	f, out := setupMockTaskCmd(t, tasks, rec)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "open", "task-open"})
	require.NoError(t, root.Execute())

	assert.Contains(t, out.String(), "opened")
	assert.Equal(t, ws.ActionOpenFile, rec.lastAction())
}

func TestOpenCommand_FolderSendsOpenFolder(t *testing.T) {
	tasks := []ws.GDTask{{
		TaskID: "task-open", Status: "completed", Progress: 100, CanOpenFolder: true,
	}}
	rec := &recordedTaskActions{}
	f, out := setupMockTaskCmd(t, tasks, rec)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "open", "task-open", "--folder"})
	require.NoError(t, root.Execute())

	assert.Contains(t, out.String(), "folder opened")
	assert.Equal(t, ws.ActionOpenFolder, rec.lastAction())
}

func TestDeleteCommand_RemoveWithoutFiles(t *testing.T) {
	tasks := []ws.GDTask{{
		TaskID: "task-del", Status: "completed", Progress: 100,
	}}
	rec := &recordedTaskActions{}
	f, out := setupMockTaskCmd(t, tasks, rec)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "delete", "task-del"})
	require.NoError(t, root.Execute())

	assert.Contains(t, out.String(), "kept downloaded files")
	assert.Equal(t, ws.ActionRemove, rec.lastAction())
}
