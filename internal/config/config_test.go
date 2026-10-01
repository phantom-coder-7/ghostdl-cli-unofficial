package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fileOnlySettings reads config.yaml with a file-only viper (no defaults, no env).
func fileOnlySettings(t *testing.T, path string) map[string]any {
	t.Helper()
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadInConfig())
	return v.AllSettings()
}

// newTestManager creates a Manager backed by a temporary directory so tests
// do not touch the real user config file.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	v, err := newViper(path)
	require.NoError(t, err)
	return &Manager{v: v, path: path}
}

func TestDir_ConfigDirEnvWins(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(ConfigDirEnv, dir)

	got, err := Dir()
	require.NoError(t, err)
	assert.Equal(t, dir, got)
}

func TestDir_UnixFallbackWhenXDGRelative(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only: XDG_CONFIG_HOME relative forces UserConfigDir failure")
	}
	home := t.TempDir()
	t.Setenv(ConfigDirEnv, "")
	t.Setenv("XDG_CONFIG_HOME", "relative-not-abs")
	t.Setenv("HOME", home)

	got, err := Dir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".config", "ghostdl-unofficial"), got)
}

func TestDir_EmptyHomeAndXDGReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("on windows UserHomeDir reads USERPROFILE, not HOME")
	}
	t.Setenv(ConfigDirEnv, "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	_, err := Dir()
	require.Error(t, err)
}

func TestStaleKeys_LoadAndUnset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("task:\n  max_count: 42\nwatch:\n  interval: 9\n"), 0644))

	v, err := newViper(path)
	require.NoError(t, err)
	m := &Manager{v: v, path: path}
	assert.Equal(t, "42", m.GetString("task.max_count"))
	assert.Equal(t, "9", m.GetString("watch.interval"))

	require.NoError(t, m.Unset("task.max_count"))
	require.NoError(t, m.Unset("watch.interval"))
	assert.Equal(t, "", m.GetString("task.max_count"))
	assert.Equal(t, "", m.GetString("watch.interval"))
}

func TestSet_PersistsValue(t *testing.T) {
	m := newTestManager(t)

	require.NoError(t, m.Set("server.url", "ws://192.168.1.1:9000"))
	assert.Equal(t, "ws://192.168.1.1:9000", m.GetString("server.url"))
}

func TestUnset_RestoresDefault(t *testing.T) {
	m := newTestManager(t)

	require.NoError(t, m.Set("server.url", "ws://192.168.1.1:9000"))
	assert.Equal(t, "ws://192.168.1.1:9000", m.GetString("server.url"))

	require.NoError(t, m.Unset("server.url"))

	assert.Equal(t, defaults["server.url"], m.GetString("server.url"))
}

func TestUnset_KeyRemovedFromFile(t *testing.T) {
	m := newTestManager(t)

	require.NoError(t, m.Set("output.format", "json"))

	require.NoError(t, m.Unset("output.format"))

	v, err := newViper(m.path)
	require.NoError(t, err)
	m2 := &Manager{v: v, path: m.path}
	assert.Equal(t, defaults["output.format"], m2.GetString("output.format"))
}

func TestUnset_EmptyStringIsNotUnset(t *testing.T) {
	m := newTestManager(t)

	require.NoError(t, m.Set("output.format", ""))
	assert.Equal(t, "", m.GetString("output.format"))

	require.NoError(t, m.Unset("output.format"))
	assert.Equal(t, defaults["output.format"], m.GetString("output.format"))
}

func TestUnset_Noop(t *testing.T) {
	m := newTestManager(t)

	require.NoError(t, m.Unset("server.url"))
	assert.Equal(t, defaults["server.url"], m.GetString("server.url"))
}

func TestUnset_UnknownKey(t *testing.T) {
	m := newTestManager(t)
	require.NoError(t, m.Unset("totally.unknown.key"))
}

func TestUnset_MultipleKeys(t *testing.T) {
	m := newTestManager(t)

	require.NoError(t, m.Set("server.url", "ws://10.0.0.1:9999"))
	require.NoError(t, m.Set("output.format", "json"))

	require.NoError(t, m.Unset("server.url"))

	assert.Equal(t, defaults["server.url"], m.GetString("server.url"))
	assert.Equal(t, "json", m.GetString("output.format"))
}

func TestDeleteNestedKey_TopLevel(t *testing.T) {
	m := map[string]any{"foo": 1, "bar": 2}
	deleteNestedKey(m, []string{"foo"})
	assert.NotContains(t, m, "foo")
	assert.Contains(t, m, "bar")
}

func TestDeleteNestedKey_Nested(t *testing.T) {
	m := map[string]any{
		"server": map[string]any{"url": "ws://...", "port": 9000},
	}
	deleteNestedKey(m, []string{"server", "url"})
	server := m["server"].(map[string]any)
	assert.NotContains(t, server, "url")
	assert.Contains(t, server, "port")
}

func TestDeleteNestedKey_PrunesEmptyParent(t *testing.T) {
	m := map[string]any{
		"server": map[string]any{"url": "ws://..."},
	}
	deleteNestedKey(m, []string{"server", "url"})
	assert.NotContains(t, m, "server")
}

func TestDeleteNestedKey_MissingPath(t *testing.T) {
	m := map[string]any{"other": 1}
	deleteNestedKey(m, []string{"server", "url"})
	assert.Contains(t, m, "other")
}

func TestSet_DoesNotWriteDefaults(t *testing.T) {
	m := newTestManager(t)

	require.NoError(t, m.Set("output.format", "json"))

	settings := fileOnlySettings(t, m.path)
	assert.Equal(t, "json", settings["output"].(map[string]any)["format"])
	assert.NotContains(t, settings, "server")
}

func TestSet_DoesNotWriteEnv(t *testing.T) {
	m := newTestManager(t)
	t.Setenv("GD_SERVER_URL", "ws://env-only.example:9999")

	require.NoError(t, m.Set("output.format", "json"))

	assert.Equal(t, "ws://env-only.example:9999", m.GetString("server.url"))

	settings := fileOnlySettings(t, m.path)
	assert.Equal(t, "json", settings["output"].(map[string]any)["format"])
	assert.NotContains(t, settings, "server")
}

func TestSet_PreservesStaleKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("task:\n  max_count: 42\n"), 0644))

	v, err := newViper(path)
	require.NoError(t, err)
	m := &Manager{v: v, path: path}
	require.NoError(t, m.Set("output.format", "json"))

	settings := fileOnlySettings(t, m.path)
	assert.Equal(t, "json", settings["output"].(map[string]any)["format"])
	assert.EqualValues(t, 42, settings["task"].(map[string]any)["max_count"])
}

func TestUnset_KeepsSiblingsNoDefaults(t *testing.T) {
	m := newTestManager(t)

	require.NoError(t, m.Set("server.url", "ws://10.0.0.1:9999"))
	require.NoError(t, m.Set("output.format", "json"))
	require.NoError(t, m.Unset("server.url"))

	settings := fileOnlySettings(t, m.path)
	assert.Equal(t, "json", settings["output"].(map[string]any)["format"])
	assert.NotContains(t, settings, "server")
}

func TestMalformedConfig_FailsAtConstruction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	malformed := []byte(":\n  - not: valid: yaml: [[[\n")
	require.NoError(t, os.WriteFile(path, malformed, 0644))

	_, err := newViper(path)
	require.Error(t, err)

	after, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, malformed, after)
}

func TestSet_NewFileMode0644(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only: file mode bits")
	}
	m := newTestManager(t)

	require.NoError(t, m.Set("output.format", "json"))

	info, err := os.Stat(m.path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}
