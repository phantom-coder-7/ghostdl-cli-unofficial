package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	authcmd "github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"
)

func TestLoginCmd_TokenSuccess(t *testing.T) {
	keyring.MockInit()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := checkUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		assert.Equal(t, "pair-token-abc", hello.Token)

		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type:            ws.TypeHelloAck,
			ProtocolVersion: ws.ProtocolVersion,
			AppVersion:      "3.0.0",
		}))
	}))
	defer srv.Close()

	f := makeCheckFactory(wstest.URL(srv))
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdLogin(f)
	cmd.SetArgs([]string{"--token", "pair-token-abc"})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, out.String(), "Login successful")
	assert.Contains(t, out.String(), wstest.URL(srv))

	stored, err := keyring.Get("ghostdl-unofficial", "login-data")
	require.NoError(t, err)
	var data auth.LoginData
	require.NoError(t, json.Unmarshal([]byte(stored), &data))
	assert.Equal(t, "pair-token-abc", data.Token)
}

func TestLoginCmd_TokenUnauthorized(t *testing.T) {
	keyring.MockInit()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := checkUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))

		require.NoError(t, conn.WriteJSON(ws.ErrorMsg{
			Type: ws.TypeError,
			Code: "unauthorized",
		}))
	}))
	defer srv.Close()

	f := makeCheckFactory(wstest.URL(srv))
	cmd := authcmd.NewCmdLogin(f)
	cmd.SetArgs([]string{"--token", "bad-token"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token verification failed")
}

func TestLoginCmd_AlreadyLoggedIn(t *testing.T) {
	keyring.MockInit()

	const storedToken = "stored-token"
	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: storedToken})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	var conns atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conns.Add(1)
		conn, err := checkUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		assert.Equal(t, ws.TypeHello, hello.Type)
		assert.Equal(t, storedToken, hello.Token)

		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type:            ws.TypeHelloAck,
			ProtocolVersion: ws.ProtocolVersion,
			AppVersion:      "3.0.0",
		}))
	}))
	defer srv.Close()

	f := makeCheckFactory(wstest.URL(srv))
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdLogin(f)
	require.NoError(t, cmd.Execute())

	assert.Equal(t, int32(1), conns.Load())
	assert.Contains(t, out.String(), "Already logged in")
	assert.Contains(t, out.String(), "Server: "+wstest.URL(srv))
	assert.NotContains(t, out.String(), "Waiting for approval")
	assert.NotContains(t, out.String(), "Login successful")

	stored, err := keyring.Get("ghostdl-unofficial", "login-data")
	require.NoError(t, err)
	assert.Equal(t, string(loginJSON), stored)
}

func TestLoginCmd_InteractivePairNoToken(t *testing.T) {
	keyring.MockInit()

	var (
		mu    sync.Mutex
		types []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := checkUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var req ws.PairRequestMsg
		require.NoError(t, conn.ReadJSON(&req))
		mu.Lock()
		types = append(types, req.Type)
		mu.Unlock()
		if req.Type != ws.TypePairRequest {
			return
		}

		require.NoError(t, conn.WriteJSON(ws.PairResultMsg{
			Type:      ws.TypePairResult,
			RequestID: req.RequestID,
			OK:        true,
			Message:   "配对成功",
			Token:     "new-pair-token",
		}))
	}))
	defer srv.Close()

	f := makeCheckFactory(wstest.URL(srv))
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdLogin(f)
	require.NoError(t, cmd.Execute())

	mu.Lock()
	got := append([]string(nil), types...)
	mu.Unlock()
	assert.Equal(t, []string{ws.TypePairRequest}, got)
	assert.Contains(t, out.String(), "Login successful")
	assert.Contains(t, out.String(), "Waiting for approval")
	assert.NotContains(t, out.String(), "Already logged in")

	stored, err := keyring.Get("ghostdl-unofficial", "login-data")
	require.NoError(t, err)
	var data auth.LoginData
	require.NoError(t, json.Unmarshal([]byte(stored), &data))
	assert.Equal(t, "new-pair-token", data.Token)
}

func TestLoginCmd_ForceReplacesStoredToken(t *testing.T) {
	keyring.MockInit()

	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "stored-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	var (
		mu    sync.Mutex
		types []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := checkUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var req ws.PairRequestMsg
		require.NoError(t, conn.ReadJSON(&req))
		mu.Lock()
		types = append(types, req.Type)
		mu.Unlock()
		if req.Type != ws.TypePairRequest {
			return
		}

		require.NoError(t, conn.WriteJSON(ws.PairResultMsg{
			Type:      ws.TypePairResult,
			RequestID: req.RequestID,
			OK:        true,
			Message:   "配对成功",
			Token:     "new-pair-token",
		}))
	}))
	defer srv.Close()

	f := makeCheckFactory(wstest.URL(srv))
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdLogin(f)
	cmd.SetArgs([]string{"--force"})
	require.NoError(t, cmd.Execute())

	mu.Lock()
	got := append([]string(nil), types...)
	mu.Unlock()
	assert.Equal(t, []string{ws.TypePairRequest}, got)
	assert.NotContains(t, got, ws.TypeHello)
	assert.Contains(t, out.String(), "Login successful")

	stored, err := keyring.Get("ghostdl-unofficial", "login-data")
	require.NoError(t, err)
	var data auth.LoginData
	require.NoError(t, json.Unmarshal([]byte(stored), &data))
	assert.Equal(t, "new-pair-token", data.Token)
}

func TestLoginCmd_StaleTokenRepairs(t *testing.T) {
	keyring.MockInit()

	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "stale-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	var (
		mu    sync.Mutex
		types []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := checkUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var msg struct {
			Type      string `json:"type"`
			RequestID string `json:"requestId"`
		}
		require.NoError(t, conn.ReadJSON(&msg))
		mu.Lock()
		types = append(types, msg.Type)
		mu.Unlock()

		if msg.Type == ws.TypeHello {
			require.NoError(t, conn.WriteJSON(ws.ErrorMsg{
				Type: ws.TypeError,
				Code: "unauthorized",
			}))
			return
		}

		require.NoError(t, conn.WriteJSON(ws.PairResultMsg{
			Type:      ws.TypePairResult,
			RequestID: msg.RequestID,
			OK:        true,
			Message:   "配对成功",
			Token:     "new-pair-token",
		}))
	}))
	defer srv.Close()

	f := makeCheckFactory(wstest.URL(srv))
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdLogin(f)
	require.NoError(t, cmd.Execute())

	mu.Lock()
	got := append([]string(nil), types...)
	mu.Unlock()
	assert.Equal(t, []string{ws.TypeHello, ws.TypePairRequest}, got)
	assert.Contains(t, out.String(), "Login successful")
	assert.NotContains(t, out.String(), "Already logged in")
	assert.NotContains(t, out.String(), "unauthorized")

	stored, err := keyring.Get("ghostdl-unofficial", "login-data")
	require.NoError(t, err)
	var data auth.LoginData
	require.NoError(t, json.Unmarshal([]byte(stored), &data))
	assert.Equal(t, "new-pair-token", data.Token)
	assert.NotEqual(t, "stale-token", data.Token)
}

func TestLoginCmd_TokenReplacesStored(t *testing.T) {
	keyring.MockInit()

	loginJSON, err := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "stored-token"})
	require.NoError(t, err)
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	var conns atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conns.Add(1)
		conn, err := checkUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		assert.Equal(t, ws.TypeHello, hello.Type)
		assert.Equal(t, "pair-token-abc", hello.Token)

		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type:            ws.TypeHelloAck,
			ProtocolVersion: ws.ProtocolVersion,
			AppVersion:      "3.0.0",
		}))
	}))
	defer srv.Close()

	f := makeCheckFactory(wstest.URL(srv))
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdLogin(f)
	cmd.SetArgs([]string{"--token", "pair-token-abc"})
	require.NoError(t, cmd.Execute())

	assert.Equal(t, int32(1), conns.Load())
	assert.Contains(t, out.String(), "Login successful")
	assert.Contains(t, out.String(), wstest.URL(srv))
	assert.NotContains(t, out.String(), "Waiting for approval")
	assert.NotContains(t, out.String(), "Already logged in")

	stored, err := keyring.Get("ghostdl-unofficial", "login-data")
	require.NoError(t, err)
	var data auth.LoginData
	require.NoError(t, json.Unmarshal([]byte(stored), &data))
	assert.Equal(t, "pair-token-abc", data.Token)
}

func TestLoginCmd_TokenUnreachable(t *testing.T) {
	keyring.MockInit()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := wstest.URL(srv)
	srv.Close()

	f := makeCheckFactory(url)
	cmd := authcmd.NewCmdLogin(f)
	cmd.SetArgs([]string{"--token", "any-token"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token verification failed")
}
