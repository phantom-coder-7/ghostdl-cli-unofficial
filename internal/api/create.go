package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

// createTaskPayload is the payload object inside create_task.
// Empty fields are omitted from the wire JSON.
type createTaskPayload struct {
	URL         string            `json:"url,omitempty"`
	Filename    string            `json:"filename,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Path        string            `json:"path,omitempty"`
	PreBlockNum int               `json:"preBlockNum,omitempty"`
}

var validTaskSources = []ws.TaskSource{
	ws.SourceDownload,
	ws.SourceResource,
}

var unsupportedTaskSources = map[ws.TaskSource]struct{}{
	ws.SourceResourceMerge: {},
	ws.SourcePageMedia:     {},
}

// ParseHeaders converts repeatable "key: value" flags into a header map.
func ParseHeaders(headers []string) (map[string]string, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		idx := strings.Index(h, ":")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid header %q: expected \"key: value\" format", h)
		}
		key := strings.TrimSpace(h[:idx])
		value := strings.TrimSpace(h[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("invalid header %q: header name is empty", h)
		}
		out[key] = value
	}
	return out, nil
}

// ValidateTaskSource checks the source against values the CLI can send on the wire.
func ValidateTaskSource(source string) (ws.TaskSource, error) {
	src := ws.TaskSource(source)
	if _, unsupported := unsupportedTaskSources[src]; unsupported {
		return "", output.NewActionableError(
			"unsupported_source",
			fmt.Sprintf("unsupported source %q: this CLI supports only download and resource (not resource_merge or page_media)", source),
			"Use --source download or --source resource.",
		)
	}
	for _, valid := range validTaskSources {
		if src == valid {
			return src, nil
		}
	}
	return "", fmt.Errorf(
		"invalid source %q: valid values are download, resource",
		source,
	)
}

// BuildCreateTaskMessage constructs the wire create_task message without sending it.
func BuildCreateTaskMessage(req *task.CreateRequest) (ws.CreateTaskMsg, error) {
	if req.URL == "" {
		return ws.CreateTaskMsg{}, fmt.Errorf("url is required")
	}

	source, err := ValidateTaskSource(string(req.Source))
	if err != nil {
		return ws.CreateTaskMsg{}, err
	}

	payload := createTaskPayload{URL: req.URL}
	if req.Filename != "" {
		payload.Filename = req.Filename
	}
	if req.Path != "" {
		payload.Path = req.Path
	}
	if len(req.Headers) > 0 {
		payload.Headers = req.Headers
	}
	if req.Threads > 0 {
		payload.PreBlockNum = req.Threads
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return ws.CreateTaskMsg{}, fmt.Errorf("failed to marshal create_task payload: %w", err)
	}

	reqID := fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), taskRequestIDCounter.Add(1))
	msg := ws.CreateTaskMsg{
		Type:      ws.TypeCreateTask,
		RequestID: reqID,
		Source:    source,
		Payload:   payloadBytes,
	}
	if req.Title != "" {
		msg.Title = req.Title
	}
	if req.Draft {
		msg.Draft = true
	}
	return msg, nil
}

// CreateTask sends create_task and handles created, drafted, and rejected results.
func CreateTask(conn *ws.Conn, req *task.CreateRequest) (*task.CreateResult, error) {
	msg, err := BuildCreateTaskMessage(req)
	if err != nil {
		return nil, err
	}

	raw, err := conn.SendGD(msg, msg.RequestID, ws.TypeCreateTaskResult, 30*time.Second)
	if err != nil {
		return nil, err
	}

	var result ws.CreateTaskResultMsg
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create_task_result: %w", err)
	}

	out := &task.CreateResult{
		Status:  result.Status,
		TaskID:  result.TaskID,
		Message: result.Message,
	}

	switch result.Status {
	case "created", "drafted":
		return out, nil
	case "rejected":
		if result.Message != "" {
			return out, output.MapGDCreateMessage(result.Message)
		}
		return out, output.NewActionableError("create_rejected", "Ghost Downloader rejected the create request", "")
	default:
		return out, fmt.Errorf("unexpected create_task_result status %q", display.Sanitize(result.Status))
	}
}
