package timefmt

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocation(t *testing.T) {
	assert.Equal(t, time.UTC, Location(true))
	assert.Equal(t, time.Local, Location(false))
}

func TestCanonicalEndsWithZ(t *testing.T) {
	s := Canonical(time.Unix(1736928000, 0))
	assert.True(t, strings.HasSuffix(s, "Z"))
	assert.Equal(t, "2025-01-15T08:00:00Z", s)
}

func TestDisplay(t *testing.T) {
	sg := time.FixedZone("+08", 8*3600)

	assert.Equal(t, "2026-09-21T07:34:38+08:00", Display("2026-09-20T23:34:38Z", sg))
	assert.Equal(t, "2026-09-20T23:34:38Z", Display("2026-09-21T07:34:38+08:00", time.UTC))
	assert.Equal(t, "", Display("", sg))
	assert.Equal(t, "not a date", Display("not a date", sg))
	assert.Equal(t, "2025-01-15T10:30:00", Display("2025-01-15T10:30:00", sg))
}
