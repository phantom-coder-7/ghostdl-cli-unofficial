package overview_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/version"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"
	overviewpkg "github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/overview"
)

func overviewFixture() []ws.GDTask {
	return []ws.GDTask{
		{TaskID: "w1", Name: "wait.zip", Status: "waiting"},
		{TaskID: "r1", Name: "run.zip", Status: "running", Speed: 1000, ReceivedBytes: 500},
		{TaskID: "r2", Name: "run2.zip", Status: "running", Speed: 250000, ReceivedBytes: 524287500, PackName: "bt"},
		{TaskID: "p1", Name: "pause.zip", Status: "paused"},
		{TaskID: "c1", Name: "done1.zip", Status: "completed", CreatedAt: "2024-01-01T10:00:00Z"},
		{TaskID: "c2", Name: "done2.zip", Status: "completed", CreatedAt: "2024-01-02T10:00:00Z"},
		{TaskID: "c3", Name: "done3.zip", Status: "completed", CreatedAt: "2024-01-03T10:00:00Z"},
		{TaskID: "c4", Name: "done4.zip", Status: "completed", CreatedAt: "2024-01-04T10:00:00Z"},
		{TaskID: "c5", Name: "done5.zip", Status: "completed", CreatedAt: "2024-01-05T10:00:00Z"},
		{TaskID: "c6", Name: "done6.zip", Status: "completed", CreatedAt: "2024-01-06T10:00:00Z"},
		{TaskID: "c7", Name: "done7.zip", Status: "completed", CreatedAt: "2024-01-07T10:00:00Z"},
		{TaskID: "c8", Name: "done8.zip", Status: "completed", CreatedAt: "2024-01-08T10:00:00Z"},
		{TaskID: "c9", Name: "done9.zip", Status: "completed", CreatedAt: "2024-01-09T10:00:00Z"},
		{TaskID: "c10", Name: "done10.zip", Status: "completed", CreatedAt: "2024-01-10T10:00:00Z"},
		{TaskID: "f1", Name: "fail1.zip", Status: "failed", CreatedAt: "2024-01-01T09:00:00Z"},
	}
}

func setupOverviewMock(t *testing.T, tasks []ws.GDTask, subscribeCount *atomic.Int32) (*cmdutil.Factory, *bytes.Buffer) {
	t.Helper()
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type: ws.TypeHelloAck, ProtocolVersion: ws.ProtocolVersion, AppVersion: "3.1.0",
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
				if subscribeCount != nil {
					subscribeCount.Add(1)
				}
				require.NoError(t, conn.WriteJSON(ws.TaskSnapshotMsg{
					Type: ws.TypeTaskSnapshot, Tasks: tasks,
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

func TestOverview_NotLoggedIn(t *testing.T) {
	keyring.MockInit()
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
	root.SetArgs([]string{"overview"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestOverview_ConnectFailureDefault(t *testing.T) {
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(map[string]string{"type": "something_unexpected"}))
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{Out: out, ErrOut: &bytes.Buffer{}},
	}
	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) { return auth.NewStore() })
	f.HttpClient = sync.OnceValues(func() (*api.Client, error) {
		return api.NewClientForURL(wstest.URL(srv)), nil
	})

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"overview"})
	err = root.Execute()
	require.Error(t, err)
	assert.NotContains(t, out.String(), "Overview connect failed")
	assert.Contains(t, err.Error(), "connect_failed")
}

func TestOverview_Unauthorized(t *testing.T) {
	keyring.MockInit()
	loginJSON, _ := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "bad"})
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(ws.ErrorMsg{Type: ws.TypeError, Code: "unauthorized"}))
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{Out: out, ErrOut: &bytes.Buffer{}},
	}
	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) { return auth.NewStore() })
	f.HttpClient = sync.OnceValues(func() (*api.Client, error) {
		return api.NewClientForURL(wstest.URL(srv)), nil
	})

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"overview"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Authentication failed — token is invalid or was regenerated.")
	assert.NotContains(t, out.String(), "Authentication failed")
}

func TestOverview_SuccessJSON(t *testing.T) {
	var subCount atomic.Int32
	f, out := setupOverviewMock(t, overviewFixture(), &subCount)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"overview", "--format", "json"})
	require.NoError(t, root.Execute())

	assert.Equal(t, int32(1), subCount.Load())

	var envelope struct {
		OK   bool                 `json:"ok"`
		Data overviewpkg.Response `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.True(t, envelope.OK)

	assert.Equal(t, 1, envelope.Data.Counts.Waiting)
	assert.Equal(t, 2, envelope.Data.Counts.Running)
	assert.Equal(t, 1, envelope.Data.Counts.Paused)
	assert.Equal(t, 10, envelope.Data.Counts.Completed)
	assert.Equal(t, 1, envelope.Data.Counts.Failed)

	assert.Equal(t, int64(251000), envelope.Data.DownloadRateBytesPerSec)
	assert.Len(t, envelope.Data.InProgress, 4)
	assert.Len(t, envelope.Data.Failed, 1)
	assert.Len(t, envelope.Data.RecentCompleted, 5)

	assert.Equal(t, version.Display(), envelope.Data.Auth.CLIVersion)
}

func TestOverview_SuccessHumanCLIVersion(t *testing.T) {
	f, out := setupOverviewMock(t, overviewFixture(), nil)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"overview"})
	require.NoError(t, root.Execute())

	assert.Contains(t, out.String(), "  CLI:     "+version.Display())
}

func TestOverview_LimitTwo(t *testing.T) {
	var subCount atomic.Int32
	f, out := setupOverviewMock(t, overviewFixture(), &subCount)

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"overview", "--limit", "2", "--format", "json"})
	require.NoError(t, root.Execute())

	var envelope struct {
		OK   bool                 `json:"ok"`
		Data overviewpkg.Response `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))

	assert.Len(t, envelope.Data.InProgress, 4)
	assert.Len(t, envelope.Data.Failed, 1)
	assert.Len(t, envelope.Data.RecentCompleted, 2)
	assert.Equal(t, 10, envelope.Data.Counts.Completed)
}
