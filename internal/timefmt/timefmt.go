package timefmt

import "time"

// Location returns UTC when utc is true, otherwise the process local zone.
func Location(utc bool) *time.Location {
	if utc {
		return time.UTC
	}
	return time.Local
}

// Canonical formats t as RFC3339 in UTC (suffix Z).
func Canonical(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// Display parses s as RFC3339 and formats it in loc. Empty s, parse failure, or
// naive strings without a zone are returned unchanged. Nil loc is treated as Local.
func Display(s string, loc *time.Location) string {
	if s == "" {
		return s
	}
	if loc == nil {
		loc = time.Local
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.In(loc).Format(time.RFC3339)
}
