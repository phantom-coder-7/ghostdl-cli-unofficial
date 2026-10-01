// Live integration tests talk to a real Ghost Downloader 3 Desktop App (BrowserService).
//
// They are opt-in and never run during the default unit suite:
//
//	go test ./internal/live -live
//	GD_LIVE=1 go test ./internal/live
//
// Do not pass -live to go test ./... — only this package defines the flag.
package live_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/version"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

var liveEnabled = flag.Bool("live", false, "run integration tests against a real Ghost Downloader Desktop App")

const skipMessage = "skipping live test; run: go test ./internal/live -live (or set GD_LIVE=1)"

const (
	liveMarker  = "ghostdl-live-test"
	liveTestURL = "https://example.com/"
)

// queueMu serialises tests that create, pause, resume, or delete tasks on the real queue.
var queueMu sync.Mutex

func TestMain(m *testing.M) {
	if os.Getenv("GD_LIVE") == "1" {
		*liveEnabled = true
	}
	flag.Parse()
	os.Exit(m.Run())
}

func requireLive(t *testing.T) {
	t.Helper()
	if !*liveEnabled {
		t.Skip(skipMessage)
	}
}

func requireLoggedIn(t *testing.T, f *cmdutil.Factory) {
	t.Helper()
	store, err := f.AuthStore()
	require.NoError(t, err, "failed to open credential store")
	if !store.IsLoggedIn() {
		t.Fatal("not logged in — run: ghostdl auth login first (live tests require a paired CLI)")
	}
}

func newProductionFactory() (*cmdutil.Factory, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	f := cmdutil.NewFactory()
	f.IOStreams = &cmdutil.IOStreams{
		In:     strings.NewReader(""),
		Out:    out,
		ErrOut: errOut,
	}
	return f, out, errOut
}

func executeRoot(t *testing.T, f *cmdutil.Factory, out, errOut *bytes.Buffer, args ...string) error {
	t.Helper()
	root := cmd.NewRootCmd(f)
	root.SetArgs(args)
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetIn(strings.NewReader(""))
	return root.Execute()
}

func resetIO(out, errOut *bytes.Buffer) {
	out.Reset()
	errOut.Reset()
}

func registerDeleteLiveTask(t *testing.T, taskID string) {
	t.Helper()
	t.Cleanup(func() {
		if taskID == "" {
			return
		}
		f, out, errOut := newProductionFactory()
		_ = executeRoot(t, f, out, errOut, "task", "delete", taskID)
	})
}

func createLiveTask(t *testing.T, f *cmdutil.Factory, out, errOut *bytes.Buffer, draft bool) string {
	t.Helper()
	args := []string{
		"task", "create", "--format", "json",
		"--url", liveTestURL,
		"--title", liveMarker,
		"--filename", liveMarker + ".txt",
	}

	if draft {
		args = append(args, "--draft")
	}
	resetIO(out, errOut)
	require.NoError(t, executeRoot(t, f, out, errOut, args...),
		"task create failed: stdout=%q stderr=%q", out.String(), errOut.String())

	var envelope struct {
		OK   bool              `json:"ok"`
		Data task.CreateResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope), "create output: %q", out.String())
	require.True(t, envelope.OK)
	require.Contains(t, []string{"created", "drafted"}, envelope.Data.Status)
	if envelope.Data.TaskID != "" {
		return envelope.Data.TaskID
	}
	return findLiveTaskID(t, f, out, errOut)
}

func liveListTasks(t *testing.T, f *cmdutil.Factory, out, errOut *bytes.Buffer) []task.Task {
	t.Helper()
	resetIO(out, errOut)
	require.NoError(t, executeRoot(t, f, out, errOut, "task", "list", "--format", "json", "--limit", "100"),
		"task list failed: stdout=%q stderr=%q", out.String(), errOut.String())

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Tasks []task.Task `json:"tasks"`
			Total int         `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	require.True(t, envelope.OK)
	return envelope.Data.Tasks
}

func taskMatchesLiveMarker(tk task.Task) bool {
	markerFile := liveMarker + ".txt"
	if strings.Contains(tk.Name, liveMarker) {
		return true
	}
	if strings.Contains(tk.Name, markerFile) {
		return true
	}
	return false
}

func findLiveTaskID(t *testing.T, f *cmdutil.Factory, out, errOut *bytes.Buffer) string {
	t.Helper()
	for _, tk := range liveListTasks(t, f, out, errOut) {
		if taskMatchesLiveMarker(tk) {
			return tk.ID
		}
	}
	t.Fatalf("could not find %q task in queue after create (list the queue in Ghost Downloader or retry)", liveMarker)
	return ""
}

func createOwnedLiveTask(t *testing.T, f *cmdutil.Factory, out, errOut *bytes.Buffer) string {
	t.Helper()
	queueMu.Lock()
	taskID := createLiveTask(t, f, out, errOut, false)
	queueMu.Unlock()
	registerDeleteLiveTask(t, taskID)
	return taskID
}

func TestLiveVersion(t *testing.T) {
	requireLive(t)

	f, out, errOut := newProductionFactory()
	require.NoError(t, executeRoot(t, f, out, errOut, "version"),
		"version failed: stdout=%q stderr=%q", out.String(), errOut.String())
	assert.Contains(t, out.String(), version.Name)
}

func TestLiveAuthCheck(t *testing.T) {
	requireLive(t)

	f, out, errOut := newProductionFactory()
	requireLoggedIn(t, f)

	resetIO(out, errOut)
	err := executeRoot(t, f, out, errOut, "auth", "check")
	require.NoError(t, err, "auth check failed (is Ghost Downloader running with Browser Extension enabled?): stdout=%q stderr=%q", out.String(), errOut.String())

	got := out.String()
	assert.Contains(t, got, "Server is up and authenticated")
	assert.Contains(t, got, "Server:")
}

func TestLiveAuthStatus(t *testing.T) {
	requireLive(t)

	f, out, errOut := newProductionFactory()
	requireLoggedIn(t, f)

	resetIO(out, errOut)
	err := executeRoot(t, f, out, errOut, "auth", "status")
	require.NoError(t, err, "auth status failed: stdout=%q stderr=%q", out.String(), errOut.String())

	got := out.String()
	assert.Contains(t, got, "Status: Logged in")
	assert.Contains(t, got, "Server:")
}

func TestLiveConfigList(t *testing.T) {
	requireLive(t)

	f, out, errOut := newProductionFactory()
	requireLoggedIn(t, f)

	resetIO(out, errOut)
	require.NoError(t, executeRoot(t, f, out, errOut, "config", "list"),
		"config list failed: stdout=%q stderr=%q", out.String(), errOut.String())

	got := out.String()
	assert.Contains(t, got, "server")
	assert.Contains(t, got, "ws://")
	assert.Contains(t, got, "Config file:")
}

func TestLiveConfigGet(t *testing.T) {
	requireLive(t)

	f, out, errOut := newProductionFactory()
	requireLoggedIn(t, f)

	resetIO(out, errOut)
	require.NoError(t, executeRoot(t, f, out, errOut, "config", "get", "server.url"),
		"config get failed: stdout=%q stderr=%q", out.String(), errOut.String())

	got := strings.TrimSpace(out.String())
	assert.True(t, strings.HasPrefix(got, "ws://"), "server.url should be a WebSocket URL, got %q", got)
}

func TestLiveTaskList(t *testing.T) {
	requireLive(t)

	f, out, errOut := newProductionFactory()
	requireLoggedIn(t, f)

	resetIO(out, errOut)
	err := executeRoot(t, f, out, errOut, "task", "list", "--format", "json")
	require.NoError(t, err, "task list failed (is Ghost Downloader reachable?): stdout=%q stderr=%q", out.String(), errOut.String())

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Tasks []json.RawMessage `json:"tasks"`
			Total int               `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope), "expected JSON task list envelope")
	assert.True(t, envelope.OK)
	assert.GreaterOrEqual(t, envelope.Data.Total, 0)
}

func TestLiveTaskGet(t *testing.T) {
	requireLive(t)

	f, out, errOut := newProductionFactory()
	requireLoggedIn(t, f)

	taskID := createOwnedLiveTask(t, f, out, errOut)

	resetIO(out, errOut)
	require.NoError(t, executeRoot(t, f, out, errOut, "task", "get", taskID, "--format", "json"),
		"task get failed: stdout=%q stderr=%q", out.String(), errOut.String())

	var envelope struct {
		OK   bool      `json:"ok"`
		Data task.Task `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.True(t, envelope.OK)
	assert.Equal(t, taskID, envelope.Data.ID)
}

func TestLiveTaskProgress(t *testing.T) {
	requireLive(t)

	f, out, errOut := newProductionFactory()
	requireLoggedIn(t, f)

	taskID := createOwnedLiveTask(t, f, out, errOut)

	resetIO(out, errOut)
	require.NoError(t, executeRoot(t, f, out, errOut, "task", "progress", taskID, "--format", "json"),
		"task progress failed: stdout=%q stderr=%q", out.String(), errOut.String())

	var envelope struct {
		OK   bool                  `json:"ok"`
		Data task.ProgressResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &envelope))
	assert.True(t, envelope.OK)
	assert.Equal(t, taskID, envelope.Data.TaskID)
}

func TestLiveTaskCreatePauseResumeDelete(t *testing.T) {
	requireLive(t)

	queueMu.Lock()
	defer queueMu.Unlock()

	f, out, errOut := newProductionFactory()
	requireLoggedIn(t, f)

	taskID := createLiveTask(t, f, out, errOut, false)
	registerDeleteLiveTask(t, taskID)

	resetIO(out, errOut)
	pauseErr := executeRoot(t, f, out, errOut, "task", "pause", taskID)
	pauseOut := out.String()
	if pauseErr != nil {
		t.Logf("pause returned error (draft/waiting tasks may refuse pause): %v stdout=%q", pauseErr, pauseOut)
		resetIO(out, errOut)
		require.NoError(t, executeRoot(t, f, out, errOut, "task", "resume", taskID),
			"resume after failed pause: stdout=%q stderr=%q", out.String(), errOut.String())
		assert.Contains(t, out.String(), taskID)
	} else {
		assert.True(t,
			strings.Contains(pauseOut, "paused") || strings.Contains(pauseOut, " is "),
			"unexpected pause output: %q", pauseOut)

		resetIO(out, errOut)
		require.NoError(t, executeRoot(t, f, out, errOut, "task", "resume", taskID),
			"resume failed: stdout=%q stderr=%q", out.String(), errOut.String())
		resumeOut := out.String()
		assert.True(t,
			strings.Contains(resumeOut, "resumed") || strings.Contains(resumeOut, " is "),
			"unexpected resume output: %q", resumeOut)
	}

	resetIO(out, errOut)
	require.NoError(t, executeRoot(t, f, out, errOut, "task", "delete", taskID),
		"delete failed: stdout=%q stderr=%q", out.String(), errOut.String())
	assert.Contains(t, out.String(), taskID)
	assert.Contains(t, out.String(), "removed task entry")
}
