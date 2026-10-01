package task_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"
)

var dryRunUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type receivedTypes struct {
	mu    sync.Mutex
	types []string
}

func (r *receivedTypes) add(t string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types = append(r.types, t)
}

func (r *receivedTypes) hasMutations() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.types {
		if t == ws.TypeCreateTask || t == ws.TypeTaskAction {
			return true
		}
	}
	return false
}

func setupDryRunTest(t *testing.T, tracker *receivedTypes) (*cmdutil.Factory, *bytes.Buffer) {
	t.Helper()
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := dryRunUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		caps, _ := json.Marshal(ws.Capabilities{
			TaskSnapshots: true,
			TaskActions: []ws.TaskAction{
				ws.ActionTogglePause, ws.ActionCancel, ws.ActionRemove, ws.ActionRedownload,
			},
		})
		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type: ws.TypeHelloAck, ProtocolVersion: ws.ProtocolVersion, Capabilities: caps,
		}))

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var base struct {
				Type string `json:"type"`
			}
			require.NoError(t, json.Unmarshal(msgBytes, &base))
			tracker.add(base.Type)

			switch base.Type {
			case ws.TypeSubscribeTasks:
				require.NoError(t, conn.WriteJSON(ws.TaskSnapshotMsg{
					Type: ws.TypeTaskSnapshot,
					Tasks: []ws.GDTask{{
						TaskID: "task-run", Status: "running", CanPause: true,
					}},
				}))
			case ws.TypeTaskAction:
				var req ws.TaskActionMsg
				require.NoError(t, json.Unmarshal(msgBytes, &req))
				require.NoError(t, conn.WriteJSON(ws.TaskActionResultMsg{
					Type: ws.TypeTaskActionResult, RequestID: req.RequestID, OK: true,
				}))
			case ws.TypeCreateTask:
				var req ws.CreateTaskMsg
				require.NoError(t, json.Unmarshal(msgBytes, &req))
				require.NoError(t, conn.WriteJSON(ws.CreateTaskResultMsg{
					Type: ws.TypeCreateTaskResult, RequestID: req.RequestID,
					Status: "created", TaskID: "tsk_new",
				}))
			}
		}
	}))
	t.Cleanup(srv.Close)

	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{Out: out, ErrOut: &bytes.Buffer{}},
	}
	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) { return auth.NewStore() })
	f.HttpClient = sync.OnceValues(func() (*api.Client, error) {
		return api.NewClientForURL(wstest.URL(srv)), nil
	})
	return f, out
}

func TestDryRun_Create_SendsNoMutations(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"task", "create", "--url", "https://example.com/a.zip", "--dry-run",
	})
	require.NoError(t, root.Execute())
	assert.False(t, tracker.hasMutations())
	assert.Contains(t, out.String(), "create_task")
}

func TestDryRun_Pause_SendsNoMutations(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "pause", "task-run", "--dry-run"})
	require.NoError(t, root.Execute())
	assert.False(t, tracker.hasMutations())
}

func TestDryRun_Delete_SendsNoMutations(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "delete", "task-run", "--with-files", "--dry-run"})
	require.NoError(t, root.Execute())
	assert.False(t, tracker.hasMutations())
	assert.Contains(t, out.String(), "cancel")
}

func TestDryRun_Resume_SendsNoMutations(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "resume", "task-run", "--dry-run"})

	require.NoError(t, root.Execute())
	assert.False(t, tracker.hasMutations())
}

func TestDryRun_Restart_SendsNoMutations(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "restart", "task-run", "--dry-run"})
	require.NoError(t, root.Execute())
	assert.False(t, tracker.hasMutations())
}

func TestDryRun_Open_SendsNoMutations(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "open", "task-run", "--dry-run"})
	require.NoError(t, root.Execute())
	assert.False(t, tracker.hasMutations())
	assert.Contains(t, out.String(), "open_file")
}

func TestDryRun_OpenFolder_SendsNoMutations(t *testing.T) {
	tracker := &receivedTypes{}
	f, out := setupDryRunTest(t, tracker)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "open", "task-run", "--folder", "--dry-run"})
	require.NoError(t, root.Execute())
	assert.False(t, tracker.hasMutations())
	assert.Contains(t, out.String(), "open_folder")
}

func TestCreateCommand_InvalidHeader(t *testing.T) {
	keyring.MockInit()
	loginJSON, _ := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "t"})
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{Out: out, ErrOut: &bytes.Buffer{}},
	}
	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) { return auth.NewStore() })
	f.HttpClient = sync.OnceValues(func() (*api.Client, error) {
		return api.NewClientForURL("ws://127.0.0.1:14370"), nil
	})

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"task", "create", "--url", "https://example.com/a.zip",
		"--header", "bad-header", "--dry-run",
	})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad-header")
}
