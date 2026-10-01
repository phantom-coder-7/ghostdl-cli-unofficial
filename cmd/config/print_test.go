package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatSortedMap_NestedKeysAlphabetical(t *testing.T) {
	m := map[string]any{
		"server": map[string]any{
			"url":      "ws://example",
			"insecure": false,
		},
		"output": map[string]any{
			"format": "json",
		},
	}

	assert.Equal(t, "map[format:json]", formatConfigValue(m["output"]))
	assert.Equal(t, "map[insecure:false url:ws://example]", formatConfigValue(m["server"]))
	assert.Equal(t, "map[output:map[format:json] server:map[insecure:false url:ws://example]]", formatSortedMap(m))
}

func TestPrintSettings_StableOrder(t *testing.T) {
	settings := map[string]any{
		"zeta": 1,
		"alpha": map[string]any{
			"z": true,
			"a": "x",
		},
		"mid": "ok",
	}

	var buf bytes.Buffer
	printSettings(&buf, settings, "/tmp/config.yaml")
	want := "" +
		"  alpha                          = map[a:x z:true]\n" +
		"  mid                            = ok\n" +
		"  zeta                           = 1\n" +
		"\n" +
		"Config file: /tmp/config.yaml\n"
	assert.Equal(t, want, buf.String())
}
