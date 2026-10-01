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

func TestStatusCmd_NotLoggedIn(t *testing.T) {
	keyring.MockInit()

	f := makeCheckFactory("ws://127.0.0.1:14370")
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdStatus(f)
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "Not logged in")
	assert.Contains(t, out.String(), "ghostdl auth login")
}

func TestStatusCmd_LoggedIn(t *testing.T) {
	keyring.MockInit()
	loginJSON, _ := json.Marshal(&auth.LoginData{Kind: auth.KindToken, Token: "abcdefghijklmnop"})
	require.NoError(t, keyring.Set("ghostdl-unofficial", "login-data", string(loginJSON)))

	mockURL := "ws://mock.example/ws"
	f := makeCheckFactory(mockURL)
	out := f.IOStreams.Out.(*bytes.Buffer)

	cmd := authcmd.NewCmdStatus(f)
	require.NoError(t, cmd.Execute())
	outStr := out.String()
	assert.Contains(t, outStr, "Logged in")
	assert.Contains(t, outStr, mockURL)
	assert.Contains(t, outStr, "abcdefgh")
	assert.Contains(t, outStr, "mnop")
}
