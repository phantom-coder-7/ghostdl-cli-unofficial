package log

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupQuietByDefault(t *testing.T) {
	errOut := &bytes.Buffer{}
	cleanup, err := Setup(false, "", errOut)
	require.NoError(t, err)
	defer cleanup()

	Default().Debug("should not appear")
	assert.Empty(t, errOut.String())
}

func TestSetupVerboseWritesToErrOut(t *testing.T) {
	errOut := &bytes.Buffer{}
	cleanup, err := Setup(true, "", errOut)
	require.NoError(t, err)
	defer cleanup()

	Default().Debug("hello debug", "cmd", "test")
	assert.Contains(t, errOut.String(), "hello debug")
	assert.Contains(t, errOut.String(), "cmd=test")
}

func TestSetupLogFileImpliesVerbose(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cli.log")

	errOut := &bytes.Buffer{}
	cleanup, err := Setup(false, logPath, errOut)
	require.NoError(t, err)
	defer cleanup()

	Default().Debug("file implied verbose")
	assert.Contains(t, errOut.String(), "file implied verbose")

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "file implied verbose")
}

func TestSetupLogFileWritesSameLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cli.log")

	errOut := &bytes.Buffer{}
	cleanup, err := Setup(true, logPath, errOut)
	require.NoError(t, err)
	defer cleanup()

	Default().Debug("tee test", "status", "ok")

	errOutStr := errOut.String()
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, errOutStr, string(data))
}

func TestSetDefault(t *testing.T) {
	buf := &bytes.Buffer{}
	SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))

	Default().Info("custom logger")
	assert.Contains(t, buf.String(), "custom logger")
}

func TestSetupLogFileOpenError(t *testing.T) {
	_, err := Setup(true, "/nonexistent/dir/cli.log", &bytes.Buffer{})
	require.Error(t, err)
}
