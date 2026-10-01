package api

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/binname"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/timefmt"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

var taskRequestIDCounter atomic.Uint64

// ListTasks fetches all tasks from Ghost Downloader via the BrowserService subscribe_tasks
// protocol, then applies client-side status, pack, extension, and query filtering, optional sorting,
// and offset/limit. Snapshot order is preserved only when Sort and Order are both unset.
func ListTasks(conn *ws.Conn, req *task.ListRequest, loc *time.Location) (*task.ListResponse, error) {
	if err := ValidateListRequest(req); err != nil {
		return nil, err
	}
	snap, err := conn.SubscribeTasksOnce(15 * time.Second)
	if err != nil {
		return nil, err
	}
	return applyListFilters(snap.Tasks, req, loc)
}

// applyListFilters converts GD wire tasks to domain tasks, applies status, pack, extension,
// and query filtering, optional sorting, then applies offset and limit.
func applyListFilters(wireTasks []ws.GDTask, req *task.ListRequest, loc *time.Location) (*task.ListResponse, error) {
	if err := ValidateListRequest(req); err != nil {
		return nil, err
	}
	if loc == nil {
		loc = time.Local
	}
	var filtered []*task.Task
	for i := range wireTasks {
		t := &wireTasks[i]
		st := task.Status(t.Status)
		if !taskMatchesStatus(st, req) {
			continue
		}
		domain := convertGDTask(t, loc)
		if !taskMatchesPack(domain.PackName, req.Packs) {
			continue
		}
		if !taskMatchesExt(domain.FileExt, req.Exts) {
			continue
		}
		if !taskMatchesQuery(domain, req.Query) {
			continue
		}
		filtered = append(filtered, domain)
	}

	field, order, doSort := resolveListSort(req)
	if doSort {
		sortTaskList(filtered, field, order)
	}

	total := len(filtered)

	if req.Offset >= len(filtered) {
		filtered = []*task.Task{}
	} else {
		filtered = filtered[req.Offset:]
	}

	if req.Limit > 0 && len(filtered) > req.Limit {
		filtered = filtered[:req.Limit]
	}

	if filtered == nil {
		filtered = []*task.Task{}
	}

	return &task.ListResponse{Tasks: filtered, Total: total}, nil
}

// convertGDTask maps a GD wire-format task to the domain Task type.
func convertGDTask(t *ws.GDTask, loc *time.Location) *task.Task {
	if loc == nil {
		loc = time.Local
	}
	return &task.Task{
		ID:            t.TaskID,
		Name:          t.Name,
		Status:        task.Status(t.Status),
		Progress:      t.Progress,
		ReceivedBytes: t.ReceivedBytes,
		FileSize:      t.FileSize,
		Speed:         t.Speed,
		CreatedAt:     timefmt.Display(t.CreatedAt, loc),
		CanPause:      t.CanPause,
		CanOpenFile:   t.CanOpenFile,
		CanOpenFolder: t.CanOpenFolder,
		FileExt:       t.FileExt,
		PackName:      t.PackName,
		URL:           t.URL,
	}
}

// findGDTask fetches a task snapshot and returns the task with the given ID.
func findGDTask(conn *ws.Conn, id string) (*ws.GDTask, error) {
	snap, err := conn.SubscribeTasksOnce(15 * time.Second)
	if err != nil {
		return nil, err
	}
	for i := range snap.Tasks {
		if snap.Tasks[i].TaskID == id {
			return &snap.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", id)
}

// GetTask returns a single task by ID from the latest task_snapshot.
func GetTask(conn *ws.Conn, id string, loc *time.Location) (*task.Task, error) {
	wireTask, err := findGDTask(conn, id)
	if err != nil {
		return nil, err
	}
	return convertGDTask(wireTask, loc), nil
}

// GetProgress returns progress for a task by ID, computing ETA client-side.
func GetProgress(conn *ws.Conn, id string) (*task.ProgressResponse, error) {
	wireTask, err := findGDTask(conn, id)
	if err != nil {
		return nil, err
	}
	return ProgressFromGDTask(wireTask), nil
}

// ProgressFromGDTask builds a ProgressResponse from a GD wire task.
func ProgressFromGDTask(wireTask *ws.GDTask) *task.ProgressResponse {
	return &task.ProgressResponse{
		TaskID:   wireTask.TaskID,
		Total:    wireTask.FileSize,
		Consumed: wireTask.ReceivedBytes,
		Progress: wireProgressFraction(wireTask.Progress),
		Status:   task.Status(wireTask.Status),
		ETA:      computeETA(wireTask.FileSize, wireTask.ReceivedBytes, wireTask.Speed),
	}
}

// BrowserService progress is a percent.
func wireProgressFraction(p float64) float64 {
	frac := p / 100.0
	if frac < 0 {
		return 0
	}
	if frac > 1 {
		return 1
	}
	return frac
}

// computeETA returns estimated seconds remaining, or 0 when unknown (speed == 0).
func computeETA(fileSize, receivedBytes, speed int64) int64 {
	if speed <= 0 {
		return 0
	}
	remaining := fileSize - receivedBytes
	if remaining <= 0 {
		return 0
	}
	return remaining / speed
}

// ensureActionSupported returns an error when hello_ack advertised taskActions
// and the requested action is absent.
func ensureActionSupported(conn *ws.Conn, action ws.TaskAction) error {
	actions := conn.TaskActions()
	if len(actions) == 0 {
		return nil
	}
	for _, a := range actions {
		if a == action {
			return nil
		}
	}
	return output.NewActionableError(
		"unsupported_action",
		fmt.Sprintf("action %q is not supported by this Ghost Downloader version", action),
		"Update Ghost Downloader to a version that supports this action",
	)
}

func BuildTaskActionMessage(taskID string, action ws.TaskAction) ws.TaskActionMsg {
	reqID := fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), taskRequestIDCounter.Add(1))
	return ws.TaskActionMsg{
		Type:      ws.TypeTaskAction,
		RequestID: reqID,
		TaskID:    taskID,
		Action:    action,
	}
}

func TaskAction(conn *ws.Conn, id string, action ws.TaskAction) error {
	return executeTaskAction(conn, BuildTaskActionMessage(id, action))
}

func executeTaskAction(conn *ws.Conn, msg ws.TaskActionMsg) error {
	if err := ensureActionSupported(conn, msg.Action); err != nil {
		return err
	}

	raw, err := conn.SendGD(msg, msg.RequestID, ws.TypeTaskActionResult, 30*time.Second)
	if err != nil {
		return err
	}

	var result ws.TaskActionResultMsg
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("failed to parse task_action_result: %w", err)
	}
	if result.OK {
		return nil
	}
	if result.Message != "" {
		return output.MapGDServerMessage(result.Message)
	}
	return output.NewActionableError("task_action_failed", "task action failed", "")
}

// PlanPause reads the current snapshot and decides whether toggle_pause would be sent.
func PlanPause(conn *ws.Conn, id string) (performed bool, notice string, msg *ws.TaskActionMsg, err error) {
	wireTask, err := findGDTask(conn, id)
	if err != nil {
		return false, "", nil, err
	}
	if err := refuseIfTerminal(wireTask, "pause"); err != nil {
		return false, "", nil, err
	}
	if !wireTask.CanPause {
		return false, "", nil, output.NewActionableError(
			"cannot_pause",
			"this task cannot be paused",
			"Check task status with: "+binname.Command+" task get "+id,
		)
	}

	switch task.Status(wireTask.Status) {
	case task.StatusRunning:
		m := BuildTaskActionMessage(id, ws.ActionTogglePause)
		return true, "", &m, nil
	case task.StatusPaused:
		return false, "already paused", nil, nil
	case task.StatusWaiting:
		return false, "", nil, output.NewActionableError(
			"cannot_pause",
			"task is waiting to start; nothing to pause",
			"Wait for the task to start running, then retry",
		)
	default:
		return false, "", nil, fmt.Errorf("cannot pause task in status %q", display.Sanitize(wireTask.Status))
	}
}

// PauseTask pauses a running task. When no action is needed, performed is false and
// notice explains why (e.g. already paused).
func PauseTask(conn *ws.Conn, id string) (performed bool, notice string, err error) {
	performed, notice, msg, err := PlanPause(conn, id)
	if err != nil || !performed {
		return performed, notice, err
	}
	return true, "", executeTaskAction(conn, *msg)
}

// PlanResume reads the current snapshot and decides whether toggle_pause would be sent.
func PlanResume(conn *ws.Conn, id string) (performed bool, notice string, msg *ws.TaskActionMsg, err error) {
	wireTask, err := findGDTask(conn, id)
	if err != nil {
		return false, "", nil, err
	}
	if err := refuseIfTerminal(wireTask, "resume"); err != nil {
		return false, "", nil, err
	}
	if !wireTask.CanPause {
		return false, "", nil, output.NewActionableError(
			"cannot_pause",
			"this task cannot be paused or resumed",
			"Check task status with: "+binname.Command+" task get "+id,
		)
	}

	switch task.Status(wireTask.Status) {
	case task.StatusPaused:
		m := BuildTaskActionMessage(id, ws.ActionTogglePause)
		return true, "", &m, nil
	case task.StatusRunning:
		return false, "already running", nil, nil
	case task.StatusWaiting:
		return false, "queued and will start automatically", nil, nil
	default:
		return false, "", nil, fmt.Errorf("cannot resume task in status %q", display.Sanitize(wireTask.Status))
	}
}

// ResumeTask resumes a paused task. When no action is needed, performed is false and
// notice explains why (e.g. already running).
func ResumeTask(conn *ws.Conn, id string) (performed bool, notice string, err error) {
	performed, notice, msg, err := PlanResume(conn, id)
	if err != nil || !performed {
		return performed, notice, err
	}
	return true, "", executeTaskAction(conn, *msg)
}

func refuseIfTerminal(wireTask *ws.GDTask, verb string) error {
	switch task.Status(wireTask.Status) {
	case task.StatusCompleted:
		return output.NewActionableError(
			"task_completed",
			"task already completed",
			fmt.Sprintf("Cannot %s a completed task", verb),
		)
	case task.StatusFailed:
		return output.NewActionableError(
			"task_failed",
			"task has failed",
			fmt.Sprintf("Cannot %s a failed task; use %s task restart to retry", verb, binname.Command),
		)
	default:
		return nil
	}
}
