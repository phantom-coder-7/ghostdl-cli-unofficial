package task

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func TestBuildBar_HalfFilled(t *testing.T) {
	want := strings.Repeat(barFilled, 20) + strings.Repeat(barEmpty, 20)
	assert.Equal(t, want, buildBar(0.5, 40))
}

func TestRenderProgressLine_NoANSIAndPadding(t *testing.T) {
	out := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &cmdutil.IOStreams{Out: out, ErrOut: &bytes.Buffer{}},
	}

	long := &task.ProgressResponse{
		TaskID:   "task-long-name",
		Consumed: 999999,
		Total:    999999,
		Progress: 0.99,
		ETA:      3661,
		Status:   task.StatusRunning,
	}
	short := &task.ProgressResponse{
		TaskID:   "t",
		Consumed: 1,
		Total:    100,
		Progress: 0.01,
		ETA:      5,
		Status:   task.StatusRunning,
	}

	w := renderProgressLine(f, long, 0)
	longOut := out.String()
	require.NotContains(t, longOut, "\x1b")
	assert.True(t, strings.HasPrefix(longOut, "\r"))

	out.Reset()
	renderProgressLine(f, short, w)
	shortOut := out.String()
	require.NotContains(t, shortOut, "\x1b")
	assert.True(t, strings.HasPrefix(shortOut, "\r"))
	assert.GreaterOrEqual(t, len(shortOut)-1, w, "shorter line should be padded to previous width")
}
