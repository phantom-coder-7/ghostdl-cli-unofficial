package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	authcmd "github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/wstest"

	gwebsocket "github.com/gorilla/websocket"
)

var checkUpgrader = gwebsocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// makeCheckFactory builds a test Factory wired to the given server URL.
// Call keyring.MockInit() and populate the mock keyring before calling this.
func makeCheckFactory(serverURL string) *cmdutil.Factory {
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}

	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) {
		return auth.NewStore()
	})

	f.HttpClient = sync.OnceValues(func() (*api.Client, error) {
		return api.NewClientForURL(serverURL), nil
	})

	return f
}

// TestCheckCmd_NotLoggedIn verifies the command exits with an error when no credentials are stored.
func TestCheckCmd_NotLoggedIn(t *testing.T) {
	keyring.MockInit()

	f := makeCheckFactory("ws://127.0.0.1:14370")
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdCheck(f)
	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghostdl auth login")
	assert.NotContains(t, out.String(), "Not logged in")
}

// TestCheckCmd_Success verifies the command prints server info when hello succeeds.
func TestCheckCmd_Success(t *testing.T) {
	keyring.MockInit()

	loginJSON, _ := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "valid-token"})
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := checkUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		var hello ws.HelloMsg
		require.NoError(t, conn.ReadJSON(&hello))
		assert.Equal(t, "valid-token", hello.Token)

		require.NoError(t, conn.WriteJSON(ws.HelloAckMsg{
			Type:            ws.TypeHelloAck,
			ProtocolVersion: ws.ProtocolVersion,
			AppVersion:      "3.2.1",
		}))
	}))
	defer srv.Close()

	f := makeCheckFactory(wstest.URL(srv))
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdCheck(f)
	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "authenticated")
	assert.Contains(t, out.String(), "3.2.1")
}

// TestCheckCmd_Unauthorized verifies the command prints a re-pair hint on bad token.
func TestCheckCmd_Unauthorized(t *testing.T) {
	keyring.MockInit()

	loginJSON, _ := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "stale-token"})
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

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
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdCheck(f)
	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Authentication failed")
	assert.Contains(t, err.Error(), "ghostdl auth login")
	assert.NotContains(t, out.String(), "Authentication failed")
}

// TestCheckCmd_Unreachable verifies the command prints a helpful message when the server is down.
func TestCheckCmd_Unreachable(t *testing.T) {
	keyring.MockInit()

	loginJSON, _ := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "any-token"})
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := wstest.URL(srv)
	srv.Close()

	f := makeCheckFactory(url)
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdCheck(f)
	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot reach")
	assert.NotContains(t, out.String(), "Cannot reach")
}
