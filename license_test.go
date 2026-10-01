package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
)

func TestLicenseCommandPrintsEmbeddedLicenseWithoutLogin(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(moduleRoot(t), "LICENSE"))
	require.NoError(t, err)
	require.Equal(t, string(raw), licenseText)
	require.Contains(t, licenseText, "GNU AFFERO GENERAL PUBLIC LICENSE")
	require.Contains(t, licenseText, "ghostdl-cli-unofficial")

	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		},
	}

	root := cmd.NewRootCmd(f)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"license"})

	require.NoError(t, root.Execute())

	output := out.String()
	assert.Contains(t, output, "GNU AFFERO GENERAL PUBLIC LICENSE")
	assert.Contains(t, output, "ghostdl-cli-unofficial")
	assert.Equal(t, licenseText, output)
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Dir(filename)
}
