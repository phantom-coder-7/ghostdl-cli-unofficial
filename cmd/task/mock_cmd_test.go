package task_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"
)

type recordedTaskActions struct {
	mu      sync.Mutex
	actions []ws.TaskActionMsg
}

func (r *recordedTaskActions) add(msg ws.TaskActionMsg) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions = append(r.actions, msg)
}

func (r *recordedTaskActions) lastAction() ws.TaskAction {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.actions) == 0 {
		return ""
	}
	return r.actions[len(r.actions)-1].Action
}

func setupMockTaskCmd(t *testing.T, tasks []ws.GDTask, rec *recordedTaskActions) (*cmdutil.Factory, *bytes.Buffer) {
	t.Helper()
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	caps, _ := json.Marshal(ws.Capabilities{
		TaskSnapshots: true,
		TaskActions: []ws.TaskAction{
			ws.ActionTogglePause, ws.ActionCancel, ws.ActionRemove, ws.ActionRedownload,
			ws.ActionOpenFile, ws.ActionOpenFolder,
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
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

			switch base.Type {
			case ws.TypeSubscribeTasks:
				require.NoError(t, conn.WriteJSON(ws.TaskSnapshotMsg{
					Type: ws.TypeTaskSnapshot, Tasks: tasks,
				}))
			case ws.TypeTaskAction:
				var req ws.TaskActionMsg
				require.NoError(t, json.Unmarshal(msgBytes, &req))
				if rec != nil {
					rec.add(req)
				}
				require.NoError(t, conn.WriteJSON(ws.TaskActionResultMsg{
					Type: ws.TypeTaskActionResult, RequestID: req.RequestID, OK: true,
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
