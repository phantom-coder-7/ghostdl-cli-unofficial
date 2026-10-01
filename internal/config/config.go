package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// ConfigDirEnv overrides the application config directory (used in tests).
const ConfigDirEnv = "GD_CLI_CONFIG_DIR"

// Dir returns the ghostdl-unofficial config directory (contains config.yaml).
func Dir() (string, error) {
	if d := os.Getenv(ConfigDirEnv); d != "" {
		return d, nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", homeErr
		}
		switch runtime.GOOS {
		case "windows":
			configDir = filepath.Join(home, "AppData", "Roaming")
		case "darwin", "ios":
			configDir = filepath.Join(home, "Library", "Application Support")
		default:
			configDir = filepath.Join(home, ".config")
		}
		if !filepath.IsAbs(configDir) {
			return "", fmt.Errorf("config directory is not absolute: %s", configDir)
		}
	}
	return filepath.Join(configDir, "ghostdl-unofficial"), nil
}

var defaults = map[string]any{
	"server.url":      "ws://127.0.0.1:14370",
	"server.insecure": false,
	"output.format":   "table",
}

// Configuration priority: environment variables (GD_*) > config file (YAML) > default values
type Manager struct {
	v    *viper.Viper
	path string
}

func newViper(configPath string) (*viper.Viper, error) {
	v := viper.New()
	v.SetEnvPrefix("GD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	for key, val := range defaults {
		v.SetDefault(key, val)
	}
	if err := v.ReadInConfig(); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return v, nil
}

func NewManager() (*Manager, error) {
	configDir, err := Dir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(configDir, "config.yaml")

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	v, err := newViper(configPath)
	if err != nil {
		return nil, err
	}
	return &Manager{v: v, path: configPath}, nil
}

func (m *Manager) GetString(key string) string { return m.v.GetString(key) }
func (m *Manager) GetBool(key string) bool     { return m.v.GetBool(key) }

// updateFile is the only path that writes config.yaml. It reads file-only
// settings (no defaults, no env), applies mutate, and writes via viper.
func (m *Manager) updateFile(mutate func(settings map[string]any)) error {
	fileViper := viper.New()
	fileViper.SetConfigFile(m.path)
	fileViper.SetConfigType("yaml")
	if err := fileViper.ReadInConfig(); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	settings := fileViper.AllSettings()
	if settings == nil {
		settings = map[string]any{}
	}
	mutate(settings)

	writeViper := viper.New()
	writeViper.SetConfigFile(m.path)
	writeViper.SetConfigType("yaml")
	writeViper.SetConfigPermissions(0644)
	if err := writeViper.MergeConfigMap(settings); err != nil {
		return err
	}
	if err := writeViper.WriteConfig(); err != nil {
		return err
	}

	v, err := newViper(m.path)
	if err != nil {
		return err
	}
	m.v = v
	return nil
}

func (m *Manager) Set(key string, value any) error {
	return m.updateFile(func(settings map[string]any) {
		setNestedKey(settings, strings.Split(strings.ToLower(key), "."), value)
	})
}

// Unset removes a key from the persisted config file so the default value applies again.
// Unlike Set with an empty string, Unset completely removes the key from storage.
func (m *Manager) Unset(key string) error {
	return m.updateFile(func(settings map[string]any) {
		deleteNestedKey(settings, strings.Split(strings.ToLower(key), "."))
	})
}

// setNestedKey sets a dot-separated key in a nested map, creating parent maps
// and replacing any non-map value that blocks the path.
func setNestedKey(m map[string]any, parts []string, value any) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		m[parts[0]] = value
		return
	}
	sub, ok := m[parts[0]].(map[string]any)
	if !ok {
		sub = map[string]any{}
		m[parts[0]] = sub
	}
	setNestedKey(sub, parts[1:], value)
}

// deleteNestedKey removes a dot-separated key from a nested map, pruning empty parent maps.
func deleteNestedKey(m map[string]any, parts []string) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		delete(m, parts[0])
		return
	}
	sub, ok := m[parts[0]].(map[string]any)
	if !ok {
		return
	}
	deleteNestedKey(sub, parts[1:])
	if len(sub) == 0 {
		delete(m, parts[0])
	}
}

func (m *Manager) List() map[string]any { return m.v.AllSettings() }
func (m *Manager) Path() string         { return m.path }
