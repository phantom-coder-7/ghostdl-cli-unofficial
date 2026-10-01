package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/config"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/version"
)

func TestRootHelpDoesNotIncludeBuildInformation(t *testing.T) {
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})

	require.NoError(t, root.Help())

	help := out.String()
	assert.NotContains(t, help, "build_date:")
	assert.NotContains(t, help, "commit:")
	assert.NotContains(t, help, "--version")
	assert.NotContains(t, help, "Run 'ghostdl auth login' first")
}

func TestVersionSubcommandPrintsBuildInformation(t *testing.T) {
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version"})

	require.NoError(t, root.Execute())

	output := out.String()
	assert.Contains(t, output, "name: "+version.Name)
	assert.Contains(t, output, "homepage: "+version.Homepage)
	assert.Contains(t, output, "build_date:")
	assert.Contains(t, output, "commit:")
	assert.Contains(t, output, "os: "+runtime.GOOS)
	assert.Contains(t, output, "arch: "+runtime.GOARCH)
	assert.NotContains(t, output, "dirty:")
}

func TestRootDoesNotRegisterVersionFlag(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)

	assert.Nil(t, root.Flags().Lookup("version"))
	assert.Nil(t, root.PersistentFlags().Lookup("version"))

	var hasVersionFlag bool
	root.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "version" {
			hasVersionFlag = true
		}
	})
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "version" {
			hasVersionFlag = true
		}
	})
	assert.False(t, hasVersionFlag, "version flag should not be registered")

	verboseFlag := root.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)

	root.SetArgs([]string{"--version"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag")
	assert.NotContains(t, out.String(), "name:")
	assert.NotContains(t, out.String(), "build_date:")
}

func TestVerboseFlagAccepted(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)

	for _, args := range [][]string{
		{"version", "-v"},
		{"version", "--verbose"},
	} {
		out.Reset()
		errOut.Reset()
		root.SetArgs(args)
		require.NoError(t, root.Execute(), "args: %v", args)
		assert.Contains(t, errOut.String(), "skipping login check", "args: %v", args)
		assert.NotContains(t, out.String(), "level=DEBUG", "debug must not leak to stdout")
		assert.NotContains(t, out.String(), "skipping login check", "debug must not leak to stdout")
	}
}

func TestVerboseFlagIsNotVersion(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"-v", "version"})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "name: "+version.Name)
	assert.Contains(t, errOut.String(), "skipping login check")
}

func TestLogFileFlagAccepted(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cli.log")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"version", "--log-file", logPath})

	require.NoError(t, root.Execute())
	assert.Contains(t, errOut.String(), "skipping login check")

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, errOut.String(), string(data))
}

func TestQuietByDefaultNoDebugOnStderr(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"version"})

	require.NoError(t, root.Execute())
	assert.NotContains(t, errOut.String(), "skipping login check")
	assert.NotContains(t, errOut.String(), "level=DEBUG")
}

func TestVerboseDoesNotAffectStdoutJSON(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"version", "--format", "json", "--verbose"})

	require.NoError(t, root.Execute())
	assert.Contains(t, errOut.String(), "skipping login check")

	var info map[string]string
	require.NoError(t, json.Unmarshal([]byte(out.String()), &info))
	assert.Equal(t, version.Name, info["name"])
}

func TestRootRejectsInvalidFormat(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"task", "get", "task-123", "--format", "bogus"})

	err := root.Execute()
	require.Error(t, err)
	assert.Equal(t, `invalid output format "bogus" (valid options: table, json, ndjson, csv, list)`, err.Error())
}

func TestRootClosesLogFileOnPreRunError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cli.log")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	logCleanup = nil

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"task", "get", "task-123", "--format", "bogus", "--log-file", logPath})

	err := root.Execute()
	require.Error(t, err)
	assert.Equal(t, `invalid output format "bogus" (valid options: table, json, ndjson, csv, list)`, err.Error())
	assert.Nil(t, logCleanup, "log file handle should be released on PreRunE error")

	_, statErr := os.Stat(logPath)
	require.NoError(t, statErr, "log file should exist after setup opened it")

	f2, openErr := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0o644)
	require.NoError(t, openErr, "log file should not remain open in this process")
	require.NoError(t, f2.Close())
}

func TestRootAcceptsValidFormatBeforeAuth(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"version", "--format", "json"})

	require.NoError(t, root.Execute())
}

func TestVersionSubcommandCSVFormat(t *testing.T) {
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version", "--format", "csv"})

	require.NoError(t, root.Execute())

	records, err := csv.NewReader(strings.NewReader(out.String())).ReadAll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(records), 2)
	assert.Equal(t, []string{"key", "value"}, records[0])
}

func TestUTCFlagRegistered(t *testing.T) {
	keyring.MockInit()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: errOut,
		},
	}
	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) { return auth.NewStore() })

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(errOut)

	utcFlag := root.PersistentFlags().Lookup("utc")
	require.NotNil(t, utcFlag)
	assert.Equal(t, "", utcFlag.Shorthand)
	assert.Equal(t, "false", utcFlag.DefValue)

	root.SetArgs([]string{"task", "get", "task-run", "--utc"})
	err := root.Execute()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "unknown flag")
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRootEnablesSilenceErrors(t *testing.T) {
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}

	root := NewRootCmd(f)
	assert.True(t, root.SilenceErrors, "SilenceErrors prevents Cobra from duplicating errors printed by main")
}

func TestFormatFromConfigFile(t *testing.T) {
	t.Setenv(config.ConfigDirEnv, t.TempDir())
	t.Setenv("GD_OUTPUT_FORMAT", "")

	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    &bytes.Buffer{},
			ErrOut: &bytes.Buffer{},
		},
	}
	f.Config = sync.OnceValues(func() (*config.Manager, error) {
		return config.NewManager()
	})
	cfg, err := f.Config()
	require.NoError(t, err)
	require.NoError(t, cfg.Set("output.format", "json"))

	out := &bytes.Buffer{}
	f.IOStreams.Out = out
	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version"})

	require.NoError(t, root.Execute())

	var info map[string]string
	require.NoError(t, json.Unmarshal(out.Bytes(), &info))
	assert.Equal(t, version.Name, info["name"])
}

func TestConfigRunsWithoutLogin(t *testing.T) {
	keyring.MockInit()
	t.Setenv(config.ConfigDirEnv, t.TempDir())

	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}
	f.AuthStore = sync.OnceValues(func() (*auth.Store, error) { return auth.NewStore() })
	f.Config = sync.OnceValues(func() (*config.Manager, error) {
		return config.NewManager()
	})

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"config", "set", "server.url", "ws://172.17.0.2:14370"})
	require.NoError(t, root.Execute())

	out.Reset()
	root.SetArgs([]string{"config", "get", "server.url"})
	require.NoError(t, root.Execute())
	assert.Equal(t, "ws://172.17.0.2:14370\n", out.String())
}

func TestFormatFlagWinsOverConfig(t *testing.T) {
	t.Setenv(config.ConfigDirEnv, t.TempDir())
	t.Setenv("GD_OUTPUT_FORMAT", "")

	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    &bytes.Buffer{},
			ErrOut: &bytes.Buffer{},
		},
	}
	f.Config = sync.OnceValues(func() (*config.Manager, error) {
		return config.NewManager()
	})
	cfg, err := f.Config()
	require.NoError(t, err)
	require.NoError(t, cfg.Set("output.format", "json"))

	out := &bytes.Buffer{}
	f.IOStreams.Out = out
	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version", "--format", "csv"})

	require.NoError(t, root.Execute())

	records, err := csv.NewReader(strings.NewReader(out.String())).ReadAll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(records), 2)
	assert.Equal(t, []string{"key", "value"}, records[0])
}

func TestFormatFromEnv(t *testing.T) {
	t.Setenv(config.ConfigDirEnv, t.TempDir())
	t.Setenv("GD_OUTPUT_FORMAT", "json")

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

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version"})

	require.NoError(t, root.Execute())

	var info map[string]string
	require.NoError(t, json.Unmarshal(out.Bytes(), &info))
	assert.Equal(t, version.Name, info["name"])
}

func TestFormatNilConfigStaysTable(t *testing.T) {
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}

	root := NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"version"})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "name: "+version.Name)
	assert.NotContains(t, out.String(), `"name"`)
}
