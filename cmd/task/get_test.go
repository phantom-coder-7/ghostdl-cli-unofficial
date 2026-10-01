package task_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
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
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

var getTestUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func setupGetTest(t *testing.T) (*cmdutil.Factory, *bytes.Buffer) {
	t.Helper()
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := getTestUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		caps, _ := json.Marshal(ws.Capabilities{TaskSnapshots: true})
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
			if base.Type == ws.TypeSubscribeTasks {
				require.NoError(t, conn.WriteJSON(ws.TaskSnapshotMsg{
					Type: ws.TypeTaskSnapshot,
					Tasks: []ws.GDTask{{
						TaskID:        "task-run",
						Name:          "file.zip",
						Status:        "running",
						Progress:      50.0,
						ReceivedBytes: 512,
						FileSize:      1024,
						Speed:         100,
						CreatedAt:     "2025-01-15T08:00:00Z",
						CanPause:      true,
						CanOpenFile:   false,
						CanOpenFolder: true,
						FileExt:       ".zip",
						PackName:      "pack-a",
						URL:           "https://example.com/file.zip",
					}},
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

func countTableColumns(out string) int {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		return 0
	}
	if strings.Contains(lines[0], "\t") {
		return len(strings.Split(lines[0], "\t"))
	}
	return len(strings.Fields(lines[0]))
}

func TestGetCommand_OverviewColumnCount(t *testing.T) {
	f, out := setupGetTest(t)
	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "get", "task-run"})
	require.NoError(t, root.Execute())
	assert.Equal(t, 5, countTableColumns(out.String()))
	assert.NotContains(t, out.String(), "Speed")
}

func TestGetCommand_CreatedAtLocalShape(t *testing.T) {
	f, out := setupGetTest(t)
	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "get", "task-run", "--format", "json", "--detail"})
	require.NoError(t, root.Execute())

	var envelope struct {
		Data task.Task `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	createdAt := envelope.Data.CreatedAt
	assert.Regexp(t, regexp.MustCompile(`T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})`), createdAt)
}

func TestGetCommand_CreatedAtUTC(t *testing.T) {
	f, out := setupGetTest(t)
	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "get", "task-run", "--format", "json", "--detail", "--utc"})
	require.NoError(t, root.Execute())

	var envelope struct {
		Data task.Task `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	createdAt := envelope.Data.CreatedAt
	assert.Contains(t, createdAt, "Z")
	assert.NotRegexp(t, regexp.MustCompile(`\+\d{2}:\d{2}`), createdAt)
	assert.True(t, strings.HasSuffix(createdAt, "Z"))
}

func TestGetCommand_DetailColumnCount(t *testing.T) {
	f, out := setupGetTest(t)
	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"task", "get", "task-run", "--detail"})
	require.NoError(t, root.Execute())
	outStr := out.String()
	assert.Contains(t, outStr, "Speed")
	assert.Contains(t, outStr, "Received")
	assert.Contains(t, outStr, "URL")
	assert.Contains(t, outStr, "https://example.com/file.zip")
}
