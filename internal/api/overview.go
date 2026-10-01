package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/overview"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

// BuildOverview computes status counts, traffic aggregates, and three task slices from a snapshot.
func BuildOverview(wireTasks []ws.GDTask, limit int, loc *time.Location) (*overview.Response, error) {
	if limit < 0 {
		return nil, fmt.Errorf("limit must be zero or greater")
	}

	counts := overview.StatusCounts{}
	var downloadRate, receivedRunning int64
	var btRunning int

	for i := range wireTasks {
		st := wireTasks[i].Status
		switch task.Status(st) {
		case task.StatusWaiting:
			counts.Waiting++
		case task.StatusRunning:
			counts.Running++
			downloadRate += wireTasks[i].Speed
			receivedRunning += wireTasks[i].ReceivedBytes
			if strings.EqualFold(wireTasks[i].PackName, "bt") {
				btRunning++
			}
		case task.StatusPaused:
			counts.Paused++
		case task.StatusCompleted:
			counts.Completed++
		case task.StatusFailed:
			counts.Failed++
		}
	}

	inProgressReq := &task.ListRequest{
		Statuses: []task.Status{task.StatusWaiting, task.StatusRunning, task.StatusPaused},
	}
	inProgressResp, err := applyListFilters(wireTasks, inProgressReq, loc)
	if err != nil {
		return nil, err
	}

	failed, err := overviewLimitedSlice(wireTasks, &task.ListRequest{
		Statuses: []task.Status{task.StatusFailed},
		Sort:     "created",
		Order:    "desc",
	}, limit, loc)
	if err != nil {
		return nil, err
	}

	recentCompleted, err := overviewLimitedSlice(wireTasks, &task.ListRequest{
		Statuses: []task.Status{task.StatusCompleted},
		Sort:     "created",
		Order:    "desc",
	}, limit, loc)
	if err != nil {
		return nil, err
	}

	return &overview.Response{
		Counts:                  counts,
		DownloadRateBytesPerSec: downloadRate,
		ReceivedBytesRunning:    receivedRunning,
		BtRunningCount:          btRunning,
		InProgress:              inProgressResp.Tasks,
		Failed:                  failed,
		RecentCompleted:         recentCompleted,
	}, nil
}

func overviewLimitedSlice(wireTasks []ws.GDTask, req *task.ListRequest, limit int, loc *time.Location) ([]*task.Task, error) {
	if limit == 0 {
		return []*task.Task{}, nil
	}
	reqCopy := *req
	reqCopy.Limit = limit
	resp, err := applyListFilters(wireTasks, &reqCopy, loc)
	if err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}
