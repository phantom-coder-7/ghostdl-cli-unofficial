package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func TestParseHeaders_Valid(t *testing.T) {
	m, err := ParseHeaders([]string{"Referer: https://example.com", "User-Agent: ExampleAgent"})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", m["Referer"])
	assert.Equal(t, "ExampleAgent", m["User-Agent"])
}

func TestParseHeaders_Malformed(t *testing.T) {
	_, err := ParseHeaders([]string{"NoColonHeader"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NoColonHeader")
}

func TestValidateTaskSource_Valid(t *testing.T) {
	src, err := ValidateTaskSource("resource")
	require.NoError(t, err)
	assert.Equal(t, ws.SourceResource, src)
}

func TestValidateTaskSource_Invalid(t *testing.T) {
	_, err := ValidateTaskSource("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source")
	assert.Contains(t, err.Error(), "download, resource")
}

func TestValidateTaskSource_UnsupportedMergeAndPageMedia(t *testing.T) {
	for _, src := range []string{"resource_merge", "page_media"} {
		_, err := ValidateTaskSource(src)
		require.Error(t, err, src)
		assert.Contains(t, err.Error(), "unsupported source")
	}
}

func TestBuildCreateTaskMessage_OmitsEmptyFields(t *testing.T) {
	msg, err := BuildCreateTaskMessage(&task.CreateRequest{
		URL:    "https://example.com/a.zip",
		Source: "resource",
	})
	require.NoError(t, err)
	assert.Equal(t, ws.TypeCreateTask, msg.Type)
	assert.Equal(t, ws.SourceResource, msg.Source)
	assert.NotEmpty(t, msg.RequestID)
	assert.NotContains(t, string(msg.Payload), "preBlockNum")
	assert.NotContains(t, string(msg.Payload), "filename")
}

func TestBuildCreateTaskMessage_WithOptions(t *testing.T) {
	msg, err := BuildCreateTaskMessage(&task.CreateRequest{
		URL:      "https://example.com/a.zip",
		Filename: "a.zip",
		Path:     "/downloads",
		Headers:  map[string]string{"Referer": "https://example.com"},
		Threads:  8,
		Title:    "Example",
		Source:   "download",
		Draft:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, "Example", msg.Title)
	assert.True(t, msg.Draft)
	body := string(msg.Payload)
	assert.Contains(t, body, `"url"`)
	assert.Contains(t, body, `"filename":"a.zip"`)
	assert.Contains(t, body, `"preBlockNum":8`)
	assert.Contains(t, body, `"Referer"`)
}

func TestCreateTask_Created(t *testing.T) {
	srv := newMockGDServerConfig(t, mockGDConfig{
		onCreateTask: func(req ws.CreateTaskMsg) ws.CreateTaskResultMsg {
			return ws.CreateTaskResultMsg{
				Type:      ws.TypeCreateTaskResult,
				RequestID: req.RequestID,
				Status:    "created",
				TaskID:    "tsk_new",
			}
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	result, err := CreateTask(conn, &task.CreateRequest{
		URL: "https://example.com/file.zip", Source: "resource",
	})
	require.NoError(t, err)
	assert.Equal(t, "created", result.Status)
	assert.Equal(t, "tsk_new", result.TaskID)
}

func TestCreateTask_Drafted(t *testing.T) {
	srv := newMockGDServerConfig(t, mockGDConfig{
		onCreateTask: func(req ws.CreateTaskMsg) ws.CreateTaskResultMsg {
			return ws.CreateTaskResultMsg{
				Type:      ws.TypeCreateTaskResult,
				RequestID: req.RequestID,
				Status:    "drafted",
				TaskID:    "tsk_draft",
				Message:   "awaiting approval",
			}
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	result, err := CreateTask(conn, &task.CreateRequest{
		URL: "https://example.com/file.zip", Source: "resource", Draft: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "drafted", result.Status)
}

func TestCreateTask_Rejected(t *testing.T) {
	srv := newMockGDServerConfig(t, mockGDConfig{
		onCreateTask: func(req ws.CreateTaskMsg) ws.CreateTaskResultMsg {
			return ws.CreateTaskResultMsg{
				Type:      ws.TypeCreateTaskResult,
				RequestID: req.RequestID,
				Status:    "rejected",
				Message:   "无效的请求",
			}
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, err = CreateTask(conn, &task.CreateRequest{
		URL: "https://example.com/file.zip", Source: "resource",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "[create_rejected]")
	assert.NotContains(t, err.Error(), "task_action_failed")
}

func TestCreateTask_TopLevelError(t *testing.T) {
	srv := newMockGDServerConfig(t, mockGDConfig{
		createTaskError: &ws.ErrorMsg{
			Type:    ws.TypeError,
			Code:    "bad_request",
			Message: "无效的请求",
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, err = CreateTask(conn, &task.CreateRequest{
		URL: "https://example.com/file.zip", Source: "resource",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request")
}

func TestCreateTask_UnexpectedStatusSanitizesError(t *testing.T) {
	hostile := "weird" + "\x1b[31m" + "status"
	srv := newMockGDServerConfig(t, mockGDConfig{
		onCreateTask: func(req ws.CreateTaskMsg) ws.CreateTaskResultMsg {
			return ws.CreateTaskResultMsg{
				Type:      ws.TypeCreateTaskResult,
				RequestID: req.RequestID,
				Status:    hostile,
			}
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, err = CreateTask(conn, &task.CreateRequest{
		URL: "https://example.com/file.zip", Source: "resource",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected create_task_result status")
	assert.NotContains(t, err.Error(), "\x1b")
	assert.Contains(t, err.Error(), "\uFFFD")
}
