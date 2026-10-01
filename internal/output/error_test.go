package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapGDServerMessage_KnownMessages(t *testing.T) {
	tests := []struct {
		server string
		want   string
	}{
		{"任务不存在", "task not found"},
		{"不支持的操作", "action not supported by this Ghost Downloader version"},
		{"任务已完成", "task already completed"},
		{"当前任务不支持暂停", "this task cannot be paused"},
		{"文件尚未生成", "file has not been created yet"},
		{"目录不存在", "directory does not exist"},
	}

	for _, tc := range tests {
		t.Run(tc.server, func(t *testing.T) {
			err := MapGDServerMessage(tc.server)
			require.NotNil(t, err)
			assert.Equal(t, "task_action_failed", err.Type)
			assert.Equal(t, tc.want, err.Message)
			assert.Equal(t, tc.server, err.ServerMessage)
		})
	}
}

func TestMapGDServerMessage_UnknownMessage(t *testing.T) {
	err := MapGDServerMessage("some unknown error")
	require.NotNil(t, err)
	assert.Equal(t, "task_action_failed", err.Type)
	assert.Equal(t, "some unknown error", err.Message)
	assert.Equal(t, "some unknown error", err.ServerMessage)
}

func TestMapGDCreateMessage_Rejected(t *testing.T) {
	err := MapGDCreateMessage("无效的请求")
	require.NotNil(t, err)
	assert.Equal(t, "create_rejected", err.Type)
	assert.Equal(t, "无效的请求", err.Message)
	assert.NotContains(t, err.Error(), "task_action_failed")
}

func TestMapGDCreateMessage_ServerSideTimeout(t *testing.T) {
	serverMsg := "is_timeout error: wreq::Error { kind: Request, source: TimedOut }"
	err := MapGDCreateMessage(serverMsg)
	require.NotNil(t, err)
	assert.Equal(t, "create_rejected", err.Type)
	assert.Contains(t, err.Message, "server-side network timeout")
	assert.Contains(t, err.Hint, "CLI connected successfully")
	assert.Contains(t, err.Error(), "Server detail:")
	assert.NotContains(t, err.Error(), "task_action_failed")
}

func TestActionableError_TypePrefix(t *testing.T) {
	err := NewActionableError("example", "something failed", "try again")
	assert.True(t, strings.HasPrefix(err.Error(), "[example]"))
}

func TestActionableError_MainPrintsOnce(t *testing.T) {
	err := MapGDCreateMessage("is_timeout error: wreq::Error { source: TimedOut }")
	rendered := fmt.Sprintf("Error: %v\n", err)
	assert.Equal(t, 1, strings.Count(rendered, "create_rejected"))
	assert.Equal(t, 1, strings.Count(rendered, "server-side network timeout"))
}

func TestWriteCLIError_JSONEnvelope(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := NewActionableError("unauthorized", "Authentication failed", "run auth login")
	WriteCLIError(stdout, stderr, "json", err)

	assert.Contains(t, stderr.String(), "Error: [unauthorized] Authentication failed")
	assert.NotContains(t, stderr.String(), "run auth login")

	var doc map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))
	assert.Equal(t, false, doc["ok"])
	errObj := doc["error"].(map[string]any)
	assert.Equal(t, "unauthorized", errObj["type"])
	assert.Equal(t, "run auth login", errObj["hint"])
}

func TestWriteCLIError_TableFormat(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := NewActionableError("unreachable", "Cannot reach server", "start the app")
	WriteCLIError(stdout, stderr, "table", err)
	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), "Error:")
	assert.Contains(t, stderr.String(), "Cannot reach server")
	assert.Contains(t, stderr.String(), "start the app")
}
