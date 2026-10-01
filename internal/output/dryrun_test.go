package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func TestPrintDryRun_TableFormat(t *testing.T) {
	out := &bytes.Buffer{}
	msg := ws.TaskActionMsg{
		Type:      ws.TypeTaskAction,
		RequestID: "req-1",
		TaskID:    "tsk_abc",
		Action:    ws.ActionRemove,
	}
	require.NoError(t, PrintDryRun(out, "table", msg))
	assert.Contains(t, out.String(), "DRY-RUN")
	assert.Contains(t, out.String(), `"type": "task_action"`)
}

func TestPrintDryRun_JSONFormat(t *testing.T) {
	out := &bytes.Buffer{}
	msg := ws.TaskActionMsg{
		Type:      ws.TypeTaskAction,
		RequestID: "req-1",
		TaskID:    "tsk_abc",
		Action:    ws.ActionRemove,
	}
	require.NoError(t, PrintDryRun(out, "json", msg))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.Equal(t, true, envelope["ok"])
	data, ok := envelope["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "task_action", data["type"])
}

func TestPrintDryRunPreview_NoOp(t *testing.T) {
	out := &bytes.Buffer{}
	require.NoError(t, PrintDryRunPreview(out, "json", false, "already paused", nil))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	data := envelope["data"].(map[string]any)
	assert.Equal(t, false, data["wouldSend"])
	assert.Equal(t, "already paused", data["notice"])
}
