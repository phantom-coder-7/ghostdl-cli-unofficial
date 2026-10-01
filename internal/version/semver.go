package version

import (
	"regexp"
	"strings"
)

var (
	// Without a leading "v", require at least major.minor (e.g. 1.0, 1.10.123).
	semverNoV = regexp.MustCompile(`^\d+\.\d+(\.\d+)*(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

	// With a leading "v", allow vMAJOR alone (e.g. v1, v1.0, v1.10.123).
	semverWithV = regexp.MustCompile(`^v\d+(\.\d+)*(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
)

func isSemverTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false
	}
	if tag[0] == 'v' {
		return semverWithV.MatchString(tag)
	}
	return semverNoV.MatchString(tag)
}
