package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

var tzSG = time.FixedZone("+08:00", 8*60*60)

func TestApplyListFilters(t *testing.T) {
	sample := sampleGDTasks()
	tests := []struct {
		name       string
		tasks      []ws.GDTask
		req        *task.ListRequest
		wantTotal  int
		wantIDs    []string
		wantStatus task.Status // when len(wantIDs)==1 and non-zero, also assert task status
	}{
		{
			name: "no filter", tasks: sample, req: &task.ListRequest{Limit: 20},
			wantTotal: 3, wantIDs: []string{"task-1", "task-2", "task-3"},
		},
		{
			name: "status filter", tasks: sample,
			req:       &task.ListRequest{Status: task.StatusCompleted, Limit: 20},
			wantTotal: 1, wantIDs: []string{"task-2"}, wantStatus: task.StatusCompleted,
		},
		{
			name: "status filter no match", tasks: sample,
			req:       &task.ListRequest{Status: task.StatusFailed, Limit: 20},
			wantTotal: 0,
		},
		{
			name: "limit", tasks: sample, req: &task.ListRequest{Limit: 2},
			wantTotal: 3, wantIDs: []string{"task-1", "task-2"},
		},
		{
			name: "offset", tasks: sample, req: &task.ListRequest{Limit: 20, Offset: 1},
			wantTotal: 3, wantIDs: []string{"task-2", "task-3"},
		},
		{
			name: "offset and limit", tasks: sample, req: &task.ListRequest{Limit: 1, Offset: 1},
			wantTotal: 3, wantIDs: []string{"task-2"},
		},
		{
			name: "offset beyond end", tasks: sample, req: &task.ListRequest{Limit: 20, Offset: 10},
			wantTotal: 3,
		},
		{
			name: "zero limit means no cap", tasks: sample, req: &task.ListRequest{Limit: 0},
			wantTotal: 3, wantIDs: []string{"task-1", "task-2", "task-3"},
		},
		{
			name: "empty tasks", tasks: []ws.GDTask{}, req: &task.ListRequest{Limit: 20},
			wantTotal: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := applyListFilters(tt.tasks, tt.req, time.Local)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, resp.Total)
			assert.Len(t, resp.Tasks, len(tt.wantIDs))
			for i, id := range tt.wantIDs {
				assert.Equal(t, id, resp.Tasks[i].ID)
			}
			if len(tt.wantIDs) == 1 && tt.wantStatus != "" {
				assert.Equal(t, tt.wantStatus, resp.Tasks[0].Status)
			}
		})
	}
}

func TestApplyListFilters_FieldMapping(t *testing.T) {
	wireTasks := []ws.GDTask{
		{
			TaskID:        "abc123",
			Name:          "archive.zip",
			Status:        "running",
			Progress:      42.5,
			ReceivedBytes: 44564480,
			FileSize:      104857600,
			Speed:         5242880,
			CreatedAt:     "2025-01-15T10:30:00",
			CanPause:      true,
			CanOpenFile:   false,
			CanOpenFolder: true,
			FileExt:       ".zip",
			PackName:      "pack1",
		},
	}
	resp, err := applyListFilters(wireTasks, &task.ListRequest{Limit: 20}, time.Local)
	require.NoError(t, err)
	require.Len(t, resp.Tasks, 1)
	t0 := resp.Tasks[0]
	assert.Equal(t, "abc123", t0.ID)
	assert.Equal(t, "archive.zip", t0.Name)
	assert.Equal(t, task.StatusRunning, t0.Status)
	assert.Equal(t, 42.5, t0.Progress)
	assert.Equal(t, int64(44564480), t0.ReceivedBytes)
	assert.Equal(t, int64(104857600), t0.FileSize)
	assert.Equal(t, int64(5242880), t0.Speed)
	assert.Equal(t, "2025-01-15T10:30:00", t0.CreatedAt)
	assert.True(t, t0.CanPause)
	assert.False(t, t0.CanOpenFile)
	assert.True(t, t0.CanOpenFolder)
	assert.Equal(t, ".zip", t0.FileExt)
	assert.Equal(t, "pack1", t0.PackName)
}

var taskTestUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type mockGDConfig struct {
	tasks           []ws.GDTask
	capabilities    *ws.Capabilities
	onTaskAction    func(req ws.TaskActionMsg) ws.TaskActionResultMsg
	onCreateTask    func(req ws.CreateTaskMsg) ws.CreateTaskResultMsg
	createTaskError *ws.ErrorMsg
}

func defaultMockCapabilities() *ws.Capabilities {
	return &ws.Capabilities{
		TaskSnapshots: true,
		TaskActions: []ws.TaskAction{
			ws.ActionTogglePause, ws.ActionCancel, ws.ActionRemove,
			ws.ActionRedownload, ws.ActionOpenFile, ws.ActionOpenFolder,
		},
	}
}

// newMockGDServer starts a test WebSocket server that mimics Ghost Downloader.
func newMockGDServer(t *testing.T, wireTasks []ws.GDTask) *httptest.Server {
	t.Helper()
	return newMockGDServerConfig(t, mockGDConfig{tasks: wireTasks, capabilities: defaultMockCapabilities()})
}

func newMockGDServerConfig(t *testing.T, cfg mockGDConfig) *httptest.Server {
	t.Helper()
	if cfg.capabilities == nil {
		cfg.capabilities = defaultMockCapabilities()
	}
	capsJSON, err := json.Marshal(cfg.capabilities)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := taskTestUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		assert.Equal(t, ws.TypeHello, hello.Type)
		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type:            ws.TypeHelloAck,
			ProtocolVersion: ws.ProtocolVersion,
			AppVersion:      "3.5.0",
			Capabilities:    capsJSON,
		}))

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var base struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(msgBytes, &base); err != nil {
				continue
			}

			switch base.Type {
			case ws.TypeSubscribeTasks:
				require.NoError(t, conn.WriteJSON(ws.TaskSnapshotMsg{
					Type:  ws.TypeTaskSnapshot,
					Tasks: cfg.tasks,
				}))
			case ws.TypeTaskAction:
				var req ws.TaskActionMsg
				require.NoError(t, json.Unmarshal(msgBytes, &req))
				if cfg.onTaskAction != nil {
					require.NoError(t, conn.WriteJSON(cfg.onTaskAction(req)))
				} else {
					require.NoError(t, conn.WriteJSON(ws.TaskActionResultMsg{
						Type:      ws.TypeTaskActionResult,
						RequestID: req.RequestID,
						OK:        true,
						TaskID:    req.TaskID,
					}))
				}
			case ws.TypeCreateTask:
				var req ws.CreateTaskMsg
				require.NoError(t, json.Unmarshal(msgBytes, &req))
				if cfg.createTaskError != nil {
					errMsg := *cfg.createTaskError
					errMsg.RequestID = req.RequestID
					require.NoError(t, conn.WriteJSON(errMsg))
					continue
				}
				if cfg.onCreateTask != nil {
					require.NoError(t, conn.WriteJSON(cfg.onCreateTask(req)))
				} else {
					require.NoError(t, conn.WriteJSON(ws.CreateTaskResultMsg{
						Type:      ws.TypeCreateTaskResult,
						RequestID: req.RequestID,
						Status:    "created",
						TaskID:    "tsk_default",
					}))
				}
			}
		}
	}))
	return srv
}

func TestListTasks_AllTasks(t *testing.T) {
	srv := newMockGDServer(t, sampleGDTasks())
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	resp, err := ListTasks(conn, &task.ListRequest{Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Len(t, resp.Tasks, 3)
}

func TestListTasks_StatusFilter(t *testing.T) {
	srv := newMockGDServer(t, sampleGDTasks())
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	resp, err := ListTasks(conn, &task.ListRequest{
		Status: task.StatusRunning,
		Limit:  20,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "task-1", resp.Tasks[0].ID)
}

func TestListTasks_Pagination(t *testing.T) {
	srv := newMockGDServer(t, sampleGDTasks())
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	resp, err := ListTasks(conn, &task.ListRequest{Limit: 1, Offset: 1}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "task-2", resp.Tasks[0].ID)
}

func TestListTasks_EmptySnapshot(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	resp, err := ListTasks(conn, &task.ListRequest{Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Tasks)
}

// TestApplyListFilters_NumericCreatedAt verifies that a GDTask whose createdAt
// was a JSON number (Unix seconds) round-trips through applyListFilters correctly.
func TestApplyListFilters_NumericCreatedAt(t *testing.T) {
	// 1736928000 s = 2025-01-15T08:00:00Z
	const raw = `[{"taskId":"task-num","name":"num.zip","status":"running","progress":10,` +
		`"receivedBytes":0,"fileSize":1024,"speed":0,"createdAt":1736928000}]`

	var wireTasks []ws.GDTask
	require.NoError(t, json.Unmarshal([]byte(raw), &wireTasks))

	resp, err := applyListFilters(wireTasks, &task.ListRequest{Limit: 20}, tzSG)
	require.NoError(t, err)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "task-num", resp.Tasks[0].ID)
	assert.Equal(t, "2025-01-15T16:00:00+08:00", resp.Tasks[0].CreatedAt)

	respUTC, err := applyListFilters(wireTasks, &task.ListRequest{Limit: 20}, time.UTC)
	require.NoError(t, err)
	require.Len(t, respUTC.Tasks, 1)
	assert.Equal(t, "2025-01-15T08:00:00Z", respUTC.Tasks[0].CreatedAt)
}

func TestConvertGDTask_CreatedAtWireFormats(t *testing.T) {
	base := ws.GDTask{TaskID: "t", Name: "f.zip", Status: "running"}

	t.Run("millisecond wire canonical", func(t *testing.T) {
		const raw = `{"taskId":"t","name":"f.zip","status":"running","createdAt":1736928000000}`
		var wireTask ws.GDTask
		require.NoError(t, json.Unmarshal([]byte(raw), &wireTask))
		got := convertGDTask(&wireTask, time.UTC)
		assert.Equal(t, "2025-01-15T08:00:00Z", got.CreatedAt)
	})

	t.Run("iso with Z", func(t *testing.T) {
		wireTask := base
		wireTask.CreatedAt = "2025-01-15T08:00:00Z"
		assert.Equal(t, "2025-01-15T16:00:00+08:00", convertGDTask(&wireTask, tzSG).CreatedAt)
	})

	t.Run("iso with offset", func(t *testing.T) {
		wireTask := base
		wireTask.CreatedAt = "2025-01-15T16:00:00+08:00"
		assert.Equal(t, "2025-01-15T08:00:00Z", convertGDTask(&wireTask, time.UTC).CreatedAt)
	})

	t.Run("missing createdAt", func(t *testing.T) {
		got := convertGDTask(&base, time.UTC)
		assert.Empty(t, got.CreatedAt)
	})
}

// TestListTasks_NumericCreatedAt runs an end-to-end test where the mock GD server
// sends createdAt as a JSON number (Unix seconds), matching the current wire format.
func TestListTasks_NumericCreatedAt(t *testing.T) {
	// Raw task_snapshot JSON with numeric createdAt (1736928000 s = 2025-01-15T08:00:00Z).
	const snapshotJSON = `{"type":"task_snapshot","tasks":[` +
		`{"taskId":"task-num","name":"num.zip","status":"running","progress":10,` +
		`"receivedBytes":0,"fileSize":1024,"speed":0,"createdAt":1736928000}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := taskTestUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type:            ws.TypeHelloAck,
			ProtocolVersion: ws.ProtocolVersion,
			AppVersion:      "4.1.1",
		}))

		var sub ws.SubscribeTasksMsg
		require.NoError(t, conn.ReadJSON(&sub))

		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(snapshotJSON)))

		conn.ReadMessage()
	}))
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	resp, err := ListTasks(conn, &task.ListRequest{Limit: 20}, time.UTC)
	require.NoError(t, err)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "task-num", resp.Tasks[0].ID)
	assert.Equal(t, "2025-01-15T08:00:00Z", resp.Tasks[0].CreatedAt)
}

// sampleGDTasks returns three tasks with different statuses, newest first.
func sampleGDTasks() []ws.GDTask {
	return []ws.GDTask{
		{
			TaskID:    "task-1",
			Name:      "file1.zip",
			Status:    "running",
			Progress:  50.0,
			FileSize:  1024 * 1024,
			CanPause:  true,
			CreatedAt: "2025-01-15T10:00:00",
		},
		{
			TaskID:    "task-2",
			Name:      "file2.mp4",
			Status:    "completed",
			Progress:  100.0,
			FileSize:  512 * 1024,
			CanPause:  true,
			CreatedAt: "2025-01-14T09:00:00",
		},
		{
			TaskID:    "task-3",
			Name:      "file3.txt",
			Status:    "paused",
			Progress:  25.0,
			FileSize:  256 * 1024,
			CanPause:  true,
			CreatedAt: "2025-01-13T08:00:00",
		},
	}
}

func TestConvertGDTask_MapsURL(t *testing.T) {
	got := convertGDTask(&ws.GDTask{
		TaskID: "task-1",
		Name:   "file.zip",
		Status: "running",
		URL:    "https://example.com/file.zip",
	}, time.Local)
	assert.Equal(t, "https://example.com/file.zip", got.URL)

	empty := convertGDTask(&ws.GDTask{TaskID: "task-2", Name: "other.zip", Status: "waiting"}, time.Local)
	assert.Empty(t, empty.URL)
}

func TestGetTask_Found(t *testing.T) {
	srv := newMockGDServer(t, sampleGDTasks())
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	got, err := GetTask(conn, "task-1", time.Local)
	require.NoError(t, err)
	assert.Equal(t, "task-1", got.ID)
	assert.Equal(t, task.StatusRunning, got.Status)
}

func TestGetTask_NotFound(t *testing.T) {
	srv := newMockGDServer(t, sampleGDTasks())
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, err = GetTask(conn, "missing-id", time.Local)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found: missing-id")
}

func TestGetProgress_FoundWithETA(t *testing.T) {
	tasks := []ws.GDTask{{
		TaskID:        "task-eta",
		Name:          "big.zip",
		Status:        "running",
		Progress:      50.0,
		ReceivedBytes: 500,
		FileSize:      1500,
		Speed:         100,
	}}
	srv := newMockGDServer(t, tasks)
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	progress, err := GetProgress(conn, "task-eta")
	require.NoError(t, err)
	assert.Equal(t, "task-eta", progress.TaskID)
	assert.Equal(t, int64(1500), progress.Total)
	assert.Equal(t, int64(500), progress.Consumed)
	assert.Equal(t, int64(10), progress.ETA)
	assert.InDelta(t, 0.5, progress.Progress, 0.001)
	assert.Equal(t, task.StatusRunning, progress.Status)
}

func TestGetProgress_ZeroSpeedUnknownETA(t *testing.T) {
	tasks := []ws.GDTask{{
		TaskID:        "task-slow",
		Status:        "running",
		ReceivedBytes: 100,
		FileSize:      1000,
		Speed:         0,
	}}
	srv := newMockGDServer(t, tasks)
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	progress, err := GetProgress(conn, "task-slow")
	require.NoError(t, err)
	assert.Equal(t, int64(0), progress.ETA)
}

func TestGetProgress_NotFound(t *testing.T) {
	srv := newMockGDServer(t, sampleGDTasks())
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, err = GetProgress(conn, "missing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

// TestGetProgress_SnapshotWithoutRequestID is a regression test for the live-server
// timeout bug: task_snapshot never carries requestId, so GetProgress must use
// SubscribeTasksOnce rather than SendGD correlation.
func TestGetProgress_SnapshotWithoutRequestID(t *testing.T) {
	const snapshotJSON = `{"type":"task_snapshot","tasks":[` +
		`{"taskId":"task-live","name":"live.zip","status":"running","progress":25,` +
		`"receivedBytes":250,"fileSize":1000,"speed":50}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := taskTestUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type:            ws.TypeHelloAck,
			ProtocolVersion: ws.ProtocolVersion,
			AppVersion:      "4.3.7",
		}))

		var sub ws.SubscribeTasksMsg
		require.NoError(t, conn.ReadJSON(&sub))
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(snapshotJSON)))
		conn.ReadMessage()
	}))
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	start := time.Now()
	progress, err := GetProgress(conn, "task-live")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, "task-live", progress.TaskID)
	assert.Less(t, elapsed, 2*time.Second, "GetProgress must not wait on SendGD's 30s timeout")
}

func TestTaskAction_Success(t *testing.T) {
	srv := newMockGDServer(t, sampleGDTasks())
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	err = TaskAction(conn, "task-1", ws.ActionRemove)
	require.NoError(t, err)
}

func TestTaskAction_FailureMessage(t *testing.T) {
	srv := newMockGDServerConfig(t, mockGDConfig{
		tasks: sampleGDTasks(),
		onTaskAction: func(req ws.TaskActionMsg) ws.TaskActionResultMsg {
			return ws.TaskActionResultMsg{
				Type:      ws.TypeTaskActionResult,
				RequestID: req.RequestID,
				OK:        false,
				Message:   "任务不存在",
			}
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	err = TaskAction(conn, "task-1", ws.ActionRemove)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestTaskAction_UnsupportedAction(t *testing.T) {
	srv := newMockGDServerConfig(t, mockGDConfig{
		tasks: sampleGDTasks(),
		capabilities: &ws.Capabilities{
			TaskSnapshots: true,
			TaskActions:   []ws.TaskAction{ws.ActionTogglePause, ws.ActionCancel},
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	err = TaskAction(conn, "task-1", ws.ActionRemove)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestPauseTask_SendsToggleWhenRunning(t *testing.T) {
	var gotAction ws.TaskAction
	srv := newMockGDServerConfig(t, mockGDConfig{
		tasks: []ws.GDTask{{
			TaskID: "task-run", Status: "running", CanPause: true,
		}},
		onTaskAction: func(req ws.TaskActionMsg) ws.TaskActionResultMsg {
			gotAction = req.Action
			return ws.TaskActionResultMsg{
				Type: ws.TypeTaskActionResult, RequestID: req.RequestID, OK: true,
			}
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	performed, _, err := PauseTask(conn, "task-run")
	require.NoError(t, err)
	assert.True(t, performed)
	assert.Equal(t, ws.ActionTogglePause, gotAction)
}

func TestPauseTask_NoopWhenPaused(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-pause", Status: "paused", CanPause: true,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	performed, notice, err := PauseTask(conn, "task-pause")
	require.NoError(t, err)
	assert.False(t, performed)
	assert.Equal(t, "already paused", notice)
}

func TestPauseTask_RefuseWaiting(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-wait", Status: "waiting", CanPause: true,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = PauseTask(conn, "task-wait")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "waiting to start")
}

func TestPauseTask_RefuseCompleted(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-done", Status: "completed", CanPause: true,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = PauseTask(conn, "task-done")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already completed")
}

func TestPauseTask_RefuseFailed(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-bad", Status: "failed", CanPause: true,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = PauseTask(conn, "task-bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestPauseTask_RefuseCannotPause(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-nop", Status: "running", CanPause: false,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = PauseTask(conn, "task-nop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be paused")
}

func TestResumeTask_SendsToggleWhenPaused(t *testing.T) {
	var gotAction ws.TaskAction
	srv := newMockGDServerConfig(t, mockGDConfig{
		tasks: []ws.GDTask{{
			TaskID: "task-pause", Status: "paused", CanPause: true,
		}},
		onTaskAction: func(req ws.TaskActionMsg) ws.TaskActionResultMsg {
			gotAction = req.Action
			return ws.TaskActionResultMsg{
				Type: ws.TypeTaskActionResult, RequestID: req.RequestID, OK: true,
			}
		},
	})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	performed, _, err := ResumeTask(conn, "task-pause")
	require.NoError(t, err)
	assert.True(t, performed)
	assert.Equal(t, ws.ActionTogglePause, gotAction)
}

func TestResumeTask_NoopWhenRunning(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-run", Status: "running", CanPause: true,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	performed, notice, err := ResumeTask(conn, "task-run")
	require.NoError(t, err)
	assert.False(t, performed)
	assert.Equal(t, "already running", notice)
}

func TestResumeTask_NoopWhenWaiting(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-wait", Status: "waiting", CanPause: true,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	performed, notice, err := ResumeTask(conn, "task-wait")
	require.NoError(t, err)
	assert.False(t, performed)
	assert.Equal(t, "queued and will start automatically", notice)
}

func TestResumeTask_RefuseCompleted(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-done", Status: "completed", CanPause: true,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = ResumeTask(conn, "task-done")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already completed")
}

func TestResumeTask_RefuseCannotPause(t *testing.T) {
	srv := newMockGDServer(t, []ws.GDTask{{
		TaskID: "task-nop", Status: "paused", CanPause: false,
	}})
	defer srv.Close()

	conn, err := ws.Dial(context.Background(), wstest.URL(srv), "test-token")
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = ResumeTask(conn, "task-nop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be paused")
}

func TestComputeETA(t *testing.T) {
	assert.Equal(t, int64(10), computeETA(1500, 500, 100))
	assert.Equal(t, int64(0), computeETA(1500, 500, 0))
	assert.Equal(t, int64(0), computeETA(500, 500, 100))
}

func TestApplyListFilters_QueryMatchesNameCaseInsensitive(t *testing.T) {
	sample := sampleGDTasks()
	resp, err := applyListFilters(sample, &task.ListRequest{Query: "FILE2", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "task-2", resp.Tasks[0].ID)

	respAll, err := applyListFilters(sample, &task.ListRequest{Query: "nomatch", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 0, respAll.Total)
}

func TestApplyListFilters_QueryMatchesURLorID(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "unique-id-99", Name: "a.zip", Status: "running", URL: "https://example.com/a.zip"},
		{TaskID: "other", Name: "b.zip", Status: "running", URL: "https://example.com/report.pdf"},
	}
	resp, err := applyListFilters(tasks, &task.ListRequest{Query: "unique-id", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "unique-id-99", resp.Tasks[0].ID)

	respURL, err := applyListFilters(tasks, &task.ListRequest{Query: "report.pdf", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, respURL.Total)
	assert.Equal(t, "other", respURL.Tasks[0].ID)
}

func TestApplyListFilters_QueryAndStatus(t *testing.T) {
	sample := sampleGDTasks()
	resp, err := applyListFilters(sample, &task.ListRequest{
		Query:    "file",
		Statuses: []task.Status{task.StatusCompleted},
		Limit:    20,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "task-2", resp.Tasks[0].ID)
}

func TestApplyListFilters_MultipleStatusesOR(t *testing.T) {
	sample := sampleGDTasks()
	resp, err := applyListFilters(sample, &task.ListRequest{
		Statuses: []task.Status{task.StatusRunning, task.StatusPaused},
		Limit:    20,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, "task-1", resp.Tasks[0].ID)
	assert.Equal(t, "task-3", resp.Tasks[1].ID)

	respOne, err := applyListFilters(sample, &task.ListRequest{
		Statuses: []task.Status{task.StatusCompleted},
		Limit:    20,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, respOne.Total)
	assert.Equal(t, "task-2", respOne.Tasks[0].ID)
}

func TestApplyListFilters_SortNameAscDesc(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "t1", Name: "Charlie", Status: "running"},
		{TaskID: "t2", Name: "alpha", Status: "running"},
		{TaskID: "t3", Name: "Bravo", Status: "running"},
	}
	respAsc, err := applyListFilters(tasks, &task.ListRequest{Sort: "name", Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"t2", "t3", "t1"}, idsFromTasks(respAsc.Tasks))

	respDesc, err := applyListFilters(tasks, &task.ListRequest{Sort: "name", Order: "desc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"t1", "t3", "t2"}, idsFromTasks(respDesc.Tasks))
}

func TestApplyListFilters_SortSizeDescWithLimit(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "small", Name: "s.zip", Status: "completed", FileSize: 100},
		{TaskID: "large", Name: "l.zip", Status: "completed", FileSize: 9000},
		{TaskID: "mid", Name: "m.zip", Status: "completed", FileSize: 500},
	}
	resp, err := applyListFilters(tasks, &task.ListRequest{
		Statuses: []task.Status{task.StatusCompleted},
		Sort:     "size",
		Order:    "desc",
		Limit:    1,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "large", resp.Tasks[0].ID)
}

func TestApplyListFilters_NoSortPreservesInputOrder(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "newest", Name: "n.zip", Status: "running", CreatedAt: "2025-01-20T10:00:00"},
		{TaskID: "oldest", Name: "o.zip", Status: "running", CreatedAt: "2025-01-01T10:00:00"},
		{TaskID: "middle", Name: "m.zip", Status: "running", CreatedAt: "2025-01-10T10:00:00"},
	}
	resp, err := applyListFilters(tasks, &task.ListRequest{Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"newest", "oldest", "middle"}, idsFromTasks(resp.Tasks))
}

func TestApplyListFilters_OrderAscSortsByCreated(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "newest", Name: "n.zip", Status: "running", CreatedAt: "2025-01-20T10:00:00"},
		{TaskID: "oldest", Name: "o.zip", Status: "running", CreatedAt: "2025-01-01T10:00:00"},
		{TaskID: "middle", Name: "m.zip", Status: "running", CreatedAt: "2025-01-10T10:00:00"},
	}
	resp, err := applyListFilters(tasks, &task.ListRequest{Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"oldest", "middle", "newest"}, idsFromTasks(resp.Tasks))
}

func TestValidateListRequest_InvalidSortAndOrder(t *testing.T) {
	err := ValidateListRequest(&task.ListRequest{Sort: "bogus"})
	require.Error(t, err)
	assert.Equal(t, `invalid sort field "bogus" (valid options: created, id, name, pack, progress, received, size, speed, status)`, err.Error())

	err = ValidateListRequest(&task.ListRequest{Order: "sideways"})
	require.Error(t, err)
	assert.Equal(t, `invalid sort order "sideways" (valid options: asc, desc)`, err.Error())

	_, err = applyListFilters(sampleGDTasks(), &task.ListRequest{Sort: "nope"}, time.Local)
	require.Error(t, err)
	assert.Equal(t, `invalid sort field "nope" (valid options: created, id, name, pack, progress, received, size, speed, status)`, err.Error())
}

func packFilterGDTasks() []ws.GDTask {
	return []ws.GDTask{
		{TaskID: "bt-1", Name: "torrent.bin", Status: "running", PackName: "bt"},
		{TaskID: "http-1", Name: "file.zip", Status: "running", PackName: "http"},
		{TaskID: "bt-2", Name: "other.torrent", Status: "completed", PackName: "BT"},
	}
}

func TestApplyListFilters_PackFilter(t *testing.T) {
	tasks := packFilterGDTasks()
	resp, err := applyListFilters(tasks, &task.ListRequest{Packs: []string{"bt"}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.ElementsMatch(t, []string{"bt-1", "bt-2"}, idsFromTasks(resp.Tasks))

	respFlag, err := applyListFilters(tasks, &task.ListRequest{Packs: []string{"BT"}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 2, respFlag.Total)
}

func TestApplyListFilters_PackFilterMultipleOR(t *testing.T) {
	tasks := packFilterGDTasks()
	resp, err := applyListFilters(tasks, &task.ListRequest{Packs: []string{"http", "bt"}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
}

func TestApplyListFilters_PackAndStatus(t *testing.T) {
	tasks := packFilterGDTasks()
	resp, err := applyListFilters(tasks, &task.ListRequest{
		Packs:    []string{"bt"},
		Statuses: []task.Status{task.StatusCompleted},
		Limit:    20,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "bt-2", resp.Tasks[0].ID)
}

func TestApplyListFilters_UnknownPack(t *testing.T) {
	resp, err := applyListFilters(packFilterGDTasks(), &task.ListRequest{Packs: []string{"magnet"}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Tasks)
}

func extFilterGDTasks() []ws.GDTask {
	return []ws.GDTask{
		{TaskID: "zip-1", Name: "a.zip", Status: "running", FileExt: ".zip", PackName: "http"},
		{TaskID: "zip-2", Name: "b.ZIP", Status: "running", FileExt: "zip", PackName: "http"},
		{TaskID: "bin-1", Name: "c.bin", Status: "running", FileExt: ".bin", PackName: "bt"},
	}
}

func TestApplyListFilters_ExtFilter(t *testing.T) {
	tasks := extFilterGDTasks()
	resp, err := applyListFilters(tasks, &task.ListRequest{Exts: []string{"zip"}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.ElementsMatch(t, []string{"zip-1", "zip-2"}, idsFromTasks(resp.Tasks))

	respDot, err := applyListFilters(tasks, &task.ListRequest{Exts: []string{".ZIP"}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 2, respDot.Total)
}

func TestApplyListFilters_UnknownExt(t *testing.T) {
	resp, err := applyListFilters(extFilterGDTasks(), &task.ListRequest{Exts: []string{"magnet"}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Tasks)
}

func TestApplyListFilters_ExtDotMatchesNothing(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "no-ext", Name: "a", Status: "running", FileExt: ""},
		{TaskID: "zip-1", Name: "b.zip", Status: "running", FileExt: ".zip"},
	}
	resp, err := applyListFilters(tasks, &task.ListRequest{Exts: []string{"."}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Tasks)
}

func TestApplyListFilters_ExtZipOrBin(t *testing.T) {
	resp, err := applyListFilters(extFilterGDTasks(), &task.ListRequest{Exts: []string{"zip", "bin"}, Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.ElementsMatch(t, []string{"zip-1", "zip-2", "bin-1"}, idsFromTasks(resp.Tasks))
}

func TestApplyListFilters_ExtAndPack(t *testing.T) {
	resp, err := applyListFilters(extFilterGDTasks(), &task.ListRequest{
		Exts:  []string{"zip"},
		Packs: []string{"http"},
		Limit: 20,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)

	respBT, err := applyListFilters(extFilterGDTasks(), &task.ListRequest{
		Exts:  []string{"zip"},
		Packs: []string{"bt"},
		Limit: 20,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 0, respBT.Total)
}

func TestApplyListFilters_SortReceivedAscDesc(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "low", Name: "a", Status: "running", ReceivedBytes: 100},
		{TaskID: "high", Name: "b", Status: "running", ReceivedBytes: 900},
		{TaskID: "mid", Name: "c", Status: "running", ReceivedBytes: 500},
	}
	respAsc, err := applyListFilters(tasks, &task.ListRequest{Sort: "received", Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"low", "mid", "high"}, idsFromTasks(respAsc.Tasks))

	respDesc, err := applyListFilters(tasks, &task.ListRequest{Sort: "receivedBytes", Order: "desc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"high", "mid", "low"}, idsFromTasks(respDesc.Tasks))
}

func TestApplyListFilters_SortPackAscDesc(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "t1", Name: "a", Status: "running", PackName: "zebra"},
		{TaskID: "t2", Name: "b", Status: "running", PackName: "Alpha"},
		{TaskID: "t3", Name: "c", Status: "running", PackName: "bt"},
	}
	respAsc, err := applyListFilters(tasks, &task.ListRequest{Sort: "pack", Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"t2", "t3", "t1"}, idsFromTasks(respAsc.Tasks))

	respDesc, err := applyListFilters(tasks, &task.ListRequest{Sort: "packName", Order: "desc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"t1", "t3", "t2"}, idsFromTasks(respDesc.Tasks))
}

func TestApplyListFilters_QueryMatchesPackName(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "t-pack", Name: "download.bin", Status: "running", PackName: "onlyInPackName"},
		{TaskID: "t-other", Name: "download.bin", Status: "running", PackName: "other"},
	}
	resp, err := applyListFilters(tasks, &task.ListRequest{Query: "onlyinpackname", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "t-pack", resp.Tasks[0].ID)
}

func TestApplyListFilters_QueryMatchesFileExt(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "t-ext", Name: "data", Status: "running", FileExt: ".7z"},
		{TaskID: "t-zip", Name: "data", Status: "running", FileExt: ".zip"},
	}
	resp, err := applyListFilters(tasks, &task.ListRequest{Query: ".7z", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "t-ext", resp.Tasks[0].ID)
}

func TestApplyListFilters_UnknownStatusViaStatuses(t *testing.T) {
	_, err := applyListFilters(sampleGDTasks(), &task.ListRequest{
		Statuses: []task.Status{task.Status("downloading")},
		Limit:    20,
	}, time.Local)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid status "downloading"`)
}

func TestApplyListFilters_UnknownStatusViaLegacyStatus(t *testing.T) {
	_, err := applyListFilters(sampleGDTasks(), &task.ListRequest{
		Status: task.Status("downloading"),
		Limit:  20,
	}, time.Local)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid status "downloading"`)
}

func TestProgressFromGDTask_WireValues(t *testing.T) {
	tests := []struct {
		wire float64
		want float64
	}{
		{0, 0},
		{0.5, 0.005},
		{1, 0.01},
		{42.5, 0.425},
		{100, 1},
		{150, 1},
		{-5, 0},
	}
	for _, tc := range tests {
		got := ProgressFromGDTask(&ws.GDTask{Progress: tc.wire}).Progress
		assert.InDelta(t, tc.want, got, 1e-9, "wire=%v", tc.wire)
	}
}

func allStatusGDTasks() []ws.GDTask {
	return []ws.GDTask{
		{TaskID: "st-failed", Name: "f", Status: "failed"},
		{TaskID: "st-wait", Name: "w", Status: "waiting"},
		{TaskID: "st-run", Name: "r", Status: "running"},
		{TaskID: "st-pause", Name: "p", Status: "paused"},
		{TaskID: "st-done", Name: "d", Status: "completed"},
	}
}

func TestApplyListFilters_SortStatusAscDesc(t *testing.T) {
	tasks := allStatusGDTasks()
	respAsc, err := applyListFilters(tasks, &task.ListRequest{Sort: "status", Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"st-wait", "st-run", "st-pause", "st-done", "st-failed"}, idsFromTasks(respAsc.Tasks))

	respDesc, err := applyListFilters(tasks, &task.ListRequest{Sort: "status", Order: "desc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"st-failed", "st-done", "st-pause", "st-run", "st-wait"}, idsFromTasks(respDesc.Tasks))
}

func TestApplyListFilters_StatusesOverridesStatus(t *testing.T) {
	sample := sampleGDTasks()
	resp, err := applyListFilters(sample, &task.ListRequest{
		Status:   task.StatusCompleted,
		Statuses: []task.Status{task.StatusRunning},
		Limit:    20,
	}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, "task-1", resp.Tasks[0].ID)
	assert.Equal(t, task.StatusRunning, resp.Tasks[0].Status)
}

func TestApplyListFilters_SortFieldAliases(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "new", Name: "n", Status: "running", CreatedAt: "2025-02-01T10:00:00", FileSize: 100},
		{TaskID: "old", Name: "o", Status: "running", CreatedAt: "2025-01-01T10:00:00", FileSize: 900},
		{TaskID: "mid", Name: "m", Status: "running", CreatedAt: "2025-01-15T10:00:00", FileSize: 500},
	}
	byCreated, err := applyListFilters(tasks, &task.ListRequest{Sort: "created", Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	byCreatedAt, err := applyListFilters(tasks, &task.ListRequest{Sort: "createdAt", Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, idsFromTasks(byCreated.Tasks), idsFromTasks(byCreatedAt.Tasks))

	bySize, err := applyListFilters(tasks, &task.ListRequest{Sort: "size", Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	byFileSize, err := applyListFilters(tasks, &task.ListRequest{Sort: "FileSize", Order: "asc", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, idsFromTasks(bySize.Tasks), idsFromTasks(byFileSize.Tasks))
}

func TestApplyListFilters_SortOnlyDefaultsAsc(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "t1", Name: "Charlie", Status: "running"},
		{TaskID: "t2", Name: "alpha", Status: "running"},
		{TaskID: "t3", Name: "Bravo", Status: "running"},
	}
	resp, err := applyListFilters(tasks, &task.ListRequest{Sort: "name", Limit: 20}, time.Local)
	require.NoError(t, err)
	assert.Equal(t, []string{"t2", "t3", "t1"}, idsFromTasks(resp.Tasks))
}

func idsFromTasks(tasks []*task.Task) []string {
	ids := make([]string, len(tasks))
	for i, tk := range tasks {
		ids[i] = tk.ID
	}
	return ids
}
