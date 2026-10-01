package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func overviewFixtureTasks() []ws.GDTask {
	return []ws.GDTask{
		{TaskID: "w1", Name: "wait.zip", Status: "waiting", CreatedAt: "2024-01-03T10:00:00Z"},
		{TaskID: "r1", Name: "run-http.zip", Status: "running", Speed: 100, ReceivedBytes: 1000, PackName: "http", CreatedAt: "2024-01-02T10:00:00Z"},
		{TaskID: "r2", Name: "run-bt.bin", Status: "running", Speed: 200, ReceivedBytes: 2000, PackName: "bt", CreatedAt: "2024-01-01T10:00:00Z"},
		{TaskID: "p1", Name: "pause.zip", Status: "paused", CreatedAt: "2024-01-04T10:00:00Z"},
		{TaskID: "c-old", Name: "done-old.zip", Status: "completed", CreatedAt: "2024-01-01T09:00:00Z"},
		{TaskID: "c-mid", Name: "done-mid.zip", Status: "completed", CreatedAt: "2024-01-02T09:00:00Z"},
		{TaskID: "c-new", Name: "done-new.zip", Status: "completed", CreatedAt: "2024-01-03T09:00:00Z"},
		{TaskID: "f-old", Name: "fail-old.zip", Status: "failed", CreatedAt: "2024-01-01T08:00:00Z"},
		{TaskID: "f-new", Name: "fail-new.zip", Status: "failed", CreatedAt: "2024-01-03T08:00:00Z"},
	}
}

func TestBuildOverview_TrafficAndCounts(t *testing.T) {
	resp, err := BuildOverview(overviewFixtureTasks(), 5, time.UTC)
	require.NoError(t, err)

	assert.Equal(t, 1, resp.Counts.Waiting)
	assert.Equal(t, 2, resp.Counts.Running)
	assert.Equal(t, 1, resp.Counts.Paused)
	assert.Equal(t, 3, resp.Counts.Completed)
	assert.Equal(t, 2, resp.Counts.Failed)

	assert.Equal(t, int64(300), resp.DownloadRateBytesPerSec)
	assert.Equal(t, int64(3000), resp.ReceivedBytesRunning)
	assert.Equal(t, 1, resp.BtRunningCount)
}

func TestBuildOverview_LimitFailedAndCompleted(t *testing.T) {
	resp, err := BuildOverview(overviewFixtureTasks(), 1, time.UTC)
	require.NoError(t, err)

	require.Len(t, resp.Failed, 1)
	assert.Equal(t, "f-new", resp.Failed[0].ID)

	require.Len(t, resp.RecentCompleted, 1)
	assert.Equal(t, "c-new", resp.RecentCompleted[0].ID)

	assert.Equal(t, 2, resp.Counts.Failed)
	assert.Equal(t, 3, resp.Counts.Completed)
}

func TestBuildOverview_LimitZeroEmptiesFailedAndCompleted(t *testing.T) {
	resp, err := BuildOverview(overviewFixtureTasks(), 0, time.UTC)
	require.NoError(t, err)

	assert.Empty(t, resp.Failed)
	assert.Empty(t, resp.RecentCompleted)
	assert.Equal(t, 2, resp.Counts.Failed)
	assert.Equal(t, 3, resp.Counts.Completed)
}

func TestBuildOverview_InProgressOrderMatchesSnapshot(t *testing.T) {
	tasks := overviewFixtureTasks()
	resp, err := BuildOverview(tasks, 5, time.UTC)
	require.NoError(t, err)

	require.Len(t, resp.InProgress, 4)
	assert.Equal(t, "w1", resp.InProgress[0].ID)
	assert.Equal(t, "r1", resp.InProgress[1].ID)
	assert.Equal(t, "r2", resp.InProgress[2].ID)
	assert.Equal(t, "p1", resp.InProgress[3].ID)
}

func TestBuildOverview_NegativeLimit(t *testing.T) {
	_, err := BuildOverview(overviewFixtureTasks(), -1, time.UTC)
	require.Error(t, err)
}

func TestBuildOverview_BtRunningCaseInsensitive(t *testing.T) {
	tasks := []ws.GDTask{
		{TaskID: "r1", Status: "running", Speed: 1, PackName: "BT"},
		{TaskID: "r2", Status: "running", Speed: 1, PackName: "bt"},
	}
	resp, err := BuildOverview(tasks, 5, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.BtRunningCount)
}
