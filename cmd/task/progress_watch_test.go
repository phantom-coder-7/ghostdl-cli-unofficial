package task

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"
)

func TestWatchProgress_RendersMultipleSnapshots(t *testing.T) {
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	snapshots := []ws.TaskSnapshotMsg{
		{Type: ws.TypeTaskSnapshot, Tasks: []ws.GDTask{{
			TaskID: "task-watch", Status: "running", Progress: 25,
			ReceivedBytes: 250, FileSize: 1000, Speed: 100,
		}}},
		{Type: ws.TypeTaskSnapshot, Tasks: []ws.GDTask{{
			TaskID: "task-watch", Status: "running", Progress: 75,
			ReceivedBytes: 750, FileSize: 1000, Speed: 100,
		}}},
		{Type: ws.TypeTaskSnapshot, Tasks: []ws.GDTask{{
			TaskID: "task-watch", Status: "completed", Progress: 100,
			ReceivedBytes: 1000, FileSize: 1000, Speed: 0,
		}}},
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type: ws.TypeHelloAck, ProtocolVersion: ws.ProtocolVersion,
		}))

		var sub ws.SubscribeTasksMsg
		require.NoError(t, conn.ReadJSON(&sub))
		for _, snap := range snapshots {
			require.NoError(t, conn.WriteJSON(snap))
		}
		conn.ReadMessage()
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

	done := make(chan error, 1)
	go func() {
		done <- watchProgress(context.Background(), f, "task-watch")
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not complete")
	}

	output := out.String()
	assert.Contains(t, output, "Task completed")
	assert.GreaterOrEqual(t, strings.Count(output, "task-watch"), 2)
}

func TestWatchProgress_FailedTask(t *testing.T) {
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	snapshots := []ws.TaskSnapshotMsg{{
		Type: ws.TypeTaskSnapshot, Tasks: []ws.GDTask{{
			TaskID: "task-fail", Status: "failed", Progress: 10,
			ReceivedBytes: 100, FileSize: 1000, Speed: 0,
		}},
	}}

	f, out := setupWatchServer(t, snapshots)
	err = watchProgress(context.Background(), f, "task-fail")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task failed")
	assert.NotContains(t, out.String(), "Task failed")
}

func TestWatchProgress_Interrupt(t *testing.T) {
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	snapshots := []ws.TaskSnapshotMsg{{
		Type: ws.TypeTaskSnapshot, Tasks: []ws.GDTask{{
			TaskID: "task-watch", Status: "running", Progress: 25,
			ReceivedBytes: 250, FileSize: 1000, Speed: 100,
		}},
	}}

	f, out := setupWatchServer(t, snapshots)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- watchProgress(ctx, f, "task-watch")
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not exit on cancel")
	}
	assert.Contains(t, out.String(), "Interrupted")
}

func TestWatchProgress_DeadSocket(t *testing.T) {
	keyring.MockInit()
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{Type: ws.TypeHelloAck, ProtocolVersion: ws.ProtocolVersion}))
		var sub ws.SubscribeTasksMsg
		require.NoError(t, conn.ReadJSON(&sub))
		conn.Close()
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	f := &cmdutil.Factory{IOStreams: &cmdutil.IOStreams{Out: out, ErrOut: &bytes.Buffer{}}}
	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) { return auth.NewStore() })
	f.HttpClient = sync.OnceValues(func() (*api.Client, error) {
		return api.NewClientForURL(wstest.URL(srv)), nil
	})

	err = watchProgress(context.Background(), f, "task-watch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection closed while watching")
}

func setupWatchServer(t *testing.T, snapshots []ws.TaskSnapshotMsg) (*cmdutil.Factory, *bytes.Buffer) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{Type: ws.TypeHelloAck, ProtocolVersion: ws.ProtocolVersion}))
		var sub ws.SubscribeTasksMsg
		require.NoError(t, conn.ReadJSON(&sub))
		for _, snap := range snapshots {
			require.NoError(t, conn.WriteJSON(snap))
		}
		select {}
	}))
	t.Cleanup(srv.Close)

	out := &bytes.Buffer{}
	f := &cmdutil.Factory{IOStreams: &cmdutil.IOStreams{Out: out, ErrOut: &bytes.Buffer{}}}
	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) { return auth.NewStore() })
	f.HttpClient = sync.OnceValues(func() (*api.Client, error) {
		return api.NewClientForURL(wstest.URL(srv)), nil
	})
	return f, out
}
