package ws

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGDTaskUnmarshalCreatedAt(t *testing.T) {
	noZoneSuffix := regexp.MustCompile(`T\d{2}:\d{2}:\d{2}$`)

	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "unix seconds",
			json: `{"taskId":"t1","name":"file.zip","status":"running","progress":0.5,"createdAt":1753228800}`,
			want: "2025-07-23T00:00:00Z",
		},
		{
			name: "unix milliseconds",
			json: `{"taskId":"t2","name":"file.zip","status":"done","progress":1.0,"createdAt":1753228800000}`,
			want: "2025-07-23T00:00:00Z",
		},
		{
			name: "iso string with Z",
			json: `{"taskId":"t3","name":"file.zip","status":"done","progress":1.0,"createdAt":"2025-07-23T00:00:00Z"}`,
			want: "2025-07-23T00:00:00Z",
		},
		{
			name: "iso string with offset",
			json: `{"taskId":"t3b","name":"file.zip","status":"done","progress":1.0,"createdAt":"2025-07-23T08:00:00+08:00"}`,
			want: "2025-07-23T00:00:00Z",
		},
		{
			name: "naive iso string",
			json: `{"taskId":"t3c","name":"file.zip","status":"done","progress":1.0,"createdAt":"2025-07-23T00:00:00"}`,
			want: "2025-07-23T00:00:00",
		},
		{
			name: "missing createdAt",
			json: `{"taskId":"t4","name":"file.zip","status":"done","progress":1.0}`,
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var task GDTask
			require.NoError(t, json.Unmarshal([]byte(tc.json), &task))
			assert.Equal(t, tc.want, task.CreatedAt)
			if tc.want != "" && tc.name != "naive iso string" {
				assert.False(t, noZoneSuffix.MatchString(task.CreatedAt),
					"CreatedAt %q must include Z or offset, not bare time", task.CreatedAt)
			}
		})
	}
}

func TestGDTaskUnmarshalURL(t *testing.T) {
	t.Run("url present", func(t *testing.T) {
		const raw = `{"taskId":"t1","name":"file.zip","status":"running","progress":0.5,"url":"https://example.com/file.zip"}`
		var task GDTask
		require.NoError(t, json.Unmarshal([]byte(raw), &task))
		assert.Equal(t, "https://example.com/file.zip", task.URL)
	})

	t.Run("url absent", func(t *testing.T) {
		const raw = `{"taskId":"t2","name":"file.zip","status":"running","progress":0.5}`
		var task GDTask
		require.NoError(t, json.Unmarshal([]byte(raw), &task))
		assert.Empty(t, task.URL)
	})
}
