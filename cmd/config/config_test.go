package config_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configcmd "github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd/config"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/config"
)

func newConfigFactory(t *testing.T) (*cmdutil.Factory, *bytes.Buffer) {
	t.Setenv(config.ConfigDirEnv, t.TempDir())
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}
	f.Config = sync.OnceValues(func() (*config.Manager, error) {
		return config.NewManager()
	})
	return f, out
}

func TestConfigList_ShowsDefaults(t *testing.T) {
	f, out := newConfigFactory(t)

	cmd := configcmd.NewCmdList(f)
	require.NoError(t, cmd.Execute())

	outStr := out.String()
	assert.Contains(t, outStr, "server")
	assert.Contains(t, outStr, "ws://127.0.0.1:14370")
	assert.Contains(t, outStr, "config.yaml")
	assertSortedConfigListing(t, outStr)
}

func TestConfigList_KeysSortedAlphabetical(t *testing.T) {
	f, out := newConfigFactory(t)

	cmd := configcmd.NewCmdList(f)
	require.NoError(t, cmd.Execute())
	assertSortedConfigListing(t, out.String())

	out2 := &bytes.Buffer{}
	f.IOStreams.Out = out2
	require.NoError(t, cmd.Execute())
	assert.Equal(t, out.String(), out2.String())
}

func TestConfigGet_DefaultAndSet(t *testing.T) {
	f, out := newConfigFactory(t)

	getCmd := configcmd.NewCmdGet(f)
	getCmd.SetArgs([]string{"output.format"})
	require.NoError(t, getCmd.Execute())
	assert.Equal(t, "table\n", out.String())

	out.Reset()
	setCmd := configcmd.NewCmdSet(f)
	setCmd.SetArgs([]string{"output.format", "json"})
	require.NoError(t, setCmd.Execute())
	assert.Contains(t, out.String(), "output.format = json")

	out.Reset()
	require.NoError(t, getCmd.Execute())
	assert.Equal(t, "json\n", out.String())
}

func TestConfigUnset_RestoresDefault(t *testing.T) {
	f, out := newConfigFactory(t)

	setCmd := configcmd.NewCmdSet(f)
	setCmd.SetArgs([]string{"server.url", "ws://10.0.0.1:9999"})
	require.NoError(t, setCmd.Execute())

	out.Reset()
	unsetCmd := configcmd.NewCmdUnset(f)
	unsetCmd.SetArgs([]string{"server.url"})
	require.NoError(t, unsetCmd.Execute())
	assert.Contains(t, out.String(), "server.url unset")
	assert.Contains(t, out.String(), "ws://127.0.0.1:14370")

	cfg, err := f.Config()
	require.NoError(t, err)
	assert.Equal(t, "ws://127.0.0.1:14370", cfg.GetString("server.url"))
}

func TestConfigGet_NoArgsListsAll(t *testing.T) {
	f, out := newConfigFactory(t)

	cmd := configcmd.NewCmdGet(f)
	require.NoError(t, cmd.Execute())
	outStr := out.String()
	assert.Contains(t, outStr, "ws://127.0.0.1:14370")
	assert.Contains(t, outStr, "Config file:")
	assertSortedConfigListing(t, outStr)
}

func TestConfigGet_NoArgsSameOrderAsList(t *testing.T) {
	f, out := newConfigFactory(t)

	listCmd := configcmd.NewCmdList(f)
	require.NoError(t, listCmd.Execute())
	listOut := out.String()

	out.Reset()
	getCmd := configcmd.NewCmdGet(f)
	require.NoError(t, getCmd.Execute())
	assert.Equal(t, listOut, out.String())
}

// assertSortedConfigListing locks default key order: top-level output before
// server; nested server keys insecure before url (Go map %v shape).
func assertSortedConfigListing(t *testing.T, outStr string) {
	t.Helper()

	outputIdx := strings.Index(outStr, "  output")
	serverIdx := strings.Index(outStr, "  server")
	require.GreaterOrEqual(t, outputIdx, 0, "missing output key line")
	require.GreaterOrEqual(t, serverIdx, 0, "missing server key line")
	assert.Less(t, outputIdx, serverIdx, "top-level keys must be alphabetical: output before server")

	assert.Contains(t, outStr, "  output                         = map[format:table]")
	assert.Contains(t, outStr, "  server                         = map[insecure:false url:ws://127.0.0.1:14370]")
}
