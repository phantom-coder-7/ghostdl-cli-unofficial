package auth_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	authcmd "github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
)

func TestLogoutCmd_WhenLoggedIn(t *testing.T) {
	keyring.MockInit()
	loginJSON, _ := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "secret"})
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	f := makeCheckFactory("ws://127.0.0.1:14370")
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdLogout(f)
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "Logged out")

	store, err := f.AuthStore()
	require.NoError(t, err)
	assert.False(t, store.IsLoggedIn())
}

func TestLogoutCmd_AlreadyLoggedOut(t *testing.T) {
	keyring.MockInit()

	f := makeCheckFactory("ws://127.0.0.1:14370")
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdLogout(f)
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "Already in logged out state")
}
