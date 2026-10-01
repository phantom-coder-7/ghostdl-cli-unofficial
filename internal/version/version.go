package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

const (
	// Name is the program name shown in help and version output.
	Name = "ghostdl-unofficial"

	// Homepage is the project homepage shown in help and version output.
	Homepage = "https://github.com/phantom-coder-7/ghostdl-cli-unofficial"
)

var (
	// Injected at link time via -ldflags; empty for unstamped local builds.
	ver       string
	commit    string
	buildDate string
	dirty     string
)

func init() {
	applyBuildInfoFallback()
}

func applyBuildInfoFallback() {
	if commit != "" && buildDate != "" {
		normalizeCommit()
		if dirty == "" {
			fillDirtyFromBuildInfo()
		}
		normalizeDirty()
		return
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		setUnknownDefaults()
		normalizeDirty()
		return
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if commit == "" {
				commit = setting.Value
			}
		case "vcs.time":
			if buildDate == "" {
				buildDate = setting.Value
			}
		case "vcs.modified":
			if dirty == "" && setting.Value == "true" {
				dirty = "true"
			}
		}
	}

	setUnknownDefaults()
	if dirty == "" {
		fillDirtyFromBuildInfo()
	}
	normalizeDirty()
	normalizeCommit()
}

func fillDirtyFromBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.modified" && setting.Value == "true" {
			dirty = "true"
			return
		}
	}
}

func normalizeDirty() {
	if dirty != "true" {
		dirty = "false"
	}
}

func formattedCommit() string {
	c := commit
	if dirty == "true" && !strings.HasSuffix(c, "+dirty") {
		c += "+dirty"
	}
	return c
}

func setUnknownDefaults() {
	if commit == "" {
		commit = "unknown"
	}
	if buildDate == "" {
		buildDate = "unknown"
	}
}

func normalizeCommit() {
	if commit == "" || commit == "unknown" {
		return
	}
	if len(commit) > 7 {
		commit = commit[:7]
	}
}

// Semver returns the bare product semver when ver matches semver rules, otherwise "".
func Semver() string {
	if !isSemverTag(ver) {
		return ""
	}
	if len(ver) > 0 && (ver[0] == 'v' || ver[0] == 'V') {
		return ver[1:]
	}
	return ver
}

// Display is the compact version string for overview and similar surfaces.
// Prefer the semver tag when present; otherwise the short commit. "+dirty"
// is appended when the tree is dirty, matching Info()["commit"].
func Display() string {
	if v := Semver(); v != "" {
		if dirty == "true" {
			return v + "+dirty"
		}
		return v
	}
	return formattedCommit()
}

func Info() map[string]string {
	info := map[string]string{
		"name":       Name,
		"homepage":   Homepage,
		"build_date": buildDate,
		"commit":     formattedCommit(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}
	if v := Semver(); v != "" {
		info["version"] = v
	}
	return info
}

func InfoLines() []string {
	info := Info()
	keys := []string{"name", "homepage", "version", "build_date", "commit", "os", "arch"}
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		if val, ok := info[key]; ok {
			lines = append(lines, fmt.Sprintf("%s: %s", key, val))
		}
	}
	return lines
}

// String returns all build information as a single block.
func String() string {
	return strings.Join(InfoLines(), "\n") + "\n"
}
