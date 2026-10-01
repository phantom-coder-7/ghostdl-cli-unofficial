package version

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSemverTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag  string
		want bool
	}{

		{tag: "1.0", want: true},
		{tag: "1.1", want: true},
		{tag: "1.10.123", want: true},
		{tag: "1.0.0-rc.1", want: true},
		{tag: "1.2.3+build", want: true},

		{tag: "1", want: false},

		{tag: "v1", want: true},
		{tag: "v1.0", want: true},
		{tag: "v1.1", want: true},
		{tag: "v1.10.123", want: true},
		{tag: "v1.0.0-rc.1", want: true},
		{tag: "v1.2.3+build", want: true},

		{tag: "", want: false},
		{tag: "latest", want: false},
		{tag: "release", want: false},
		{tag: "nightly", want: false},
		{tag: "9d51721", want: false},
		{tag: "v1.0.0-dirty", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isSemverTag(tt.tag))
		})
	}
}

func TestSemverFromVer(t *testing.T) {
	original := ver
	t.Cleanup(func() { ver = original })

	ver = "1.2.3"
	assert.Equal(t, "1.2.3", Semver())

	ver = "latest"
	assert.Equal(t, "", Semver())

	ver = "v1"
	assert.Equal(t, "1", Semver())

	ver = "v1.2.3"
	assert.Equal(t, "1.2.3", Semver())
}

func TestInfoLinesIncludesRequiredFields(t *testing.T) {
	originalTag := ver
	originalCommit := commit
	originalDate := buildDate
	originalDirty := dirty
	t.Cleanup(func() {
		ver = originalTag
		commit = originalCommit
		buildDate = originalDate
		dirty = originalDirty
	})

	ver = "1.0.0"
	commit = "abc1234"
	buildDate = "2026-09-21T05:00:00Z"
	dirty = "false"

	lines := InfoLines()
	assert.Contains(t, lines, "name: "+Name)
	assert.Contains(t, lines, "homepage: "+Homepage)
	assert.Contains(t, lines, "version: 1.0.0")
	assert.Contains(t, lines, "build_date: 2026-09-21T05:00:00Z")
	assert.Contains(t, lines, "commit: abc1234")
	assert.Contains(t, lines, "os: "+runtime.GOOS)
	assert.Contains(t, lines, "arch: "+runtime.GOARCH)
	for _, line := range lines {
		assert.NotContains(t, line, "dirty:")
	}
}

func TestInfoCommitDirtySuffix(t *testing.T) {
	originalCommit := commit
	originalDirty := dirty
	t.Cleanup(func() {
		commit = originalCommit
		dirty = originalDirty
	})

	commit = "abc1234"
	dirty = "true"
	assert.Equal(t, "abc1234+dirty", Info()["commit"])
	assert.Contains(t, InfoLines(), "commit: abc1234+dirty")
	_, hasDirty := Info()["dirty"]
	assert.False(t, hasDirty)

	dirty = "false"
	assert.Equal(t, "abc1234", Info()["commit"])
	assert.Contains(t, InfoLines(), "commit: abc1234")
}

func TestDisplay(t *testing.T) {
	originalTag := ver
	originalCommit := commit
	originalDirty := dirty
	t.Cleanup(func() {
		ver = originalTag
		commit = originalCommit
		dirty = originalDirty
	})

	commit = "abc1234"
	dirty = "false"

	ver = "v1.2.3"
	assert.Equal(t, "1.2.3", Display())

	dirty = "true"
	assert.Equal(t, "1.2.3+dirty", Display())

	ver = "1.0.0"
	assert.Equal(t, "1.0.0+dirty", Display())

	ver = "nightly"
	dirty = "false"
	assert.Equal(t, "abc1234", Display())

	dirty = "true"
	assert.Equal(t, "abc1234+dirty", Display())
}

func TestInfoLinesOmitsVersionWithoutSemverTag(t *testing.T) {
	originalTag := ver
	originalCommit := commit
	originalDate := buildDate
	originalDirty := dirty
	t.Cleanup(func() {
		ver = originalTag
		commit = originalCommit
		buildDate = originalDate
		dirty = originalDirty
	})

	ver = "nightly"
	commit = "abc1234"
	buildDate = "2026-09-21T05:00:00Z"

	for _, line := range InfoLines() {
		assert.NotContains(t, line, "version:")
	}
}
