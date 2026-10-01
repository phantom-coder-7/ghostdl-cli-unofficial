package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/timefmt"
)

const ProtocolVersion = 2

const (
	// Auth — flat JSON, BrowserService protocol
	TypeHello       = "hello"
	TypeHelloAck    = "hello_ack"
	TypePairRequest = "pair_request"
	TypePairResult  = "pair_result"
	TypeError       = "error"

	// GD BrowserService task protocol
	TypeSubscribeTasks   = "subscribe_tasks"
	TypeTaskSnapshot     = "task_snapshot"
	TypeCreateTask       = "create_task"
	TypeCreateTaskResult = "create_task_result"
	TypeTaskAction       = "task_action"
	TypeTaskActionResult = "task_action_result"
)

// HelloMsg authenticates a session using a previously obtained pairing token.
type HelloMsg struct {
	Type            string `json:"type"`
	RequestID       string `json:"requestId"`
	ProtocolVersion int    `json:"protocolVersion"`
	Token           string `json:"token"`
}

// HelloAckMsg is the server's positive response to hello.
type HelloAckMsg struct {
	Type            string          `json:"type"`
	ProtocolVersion int             `json:"protocolVersion"`
	AppVersion      string          `json:"appVersion"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
}

// PairRequestMsg requests a new pairing token. GD shows a user-approval dialog.
type PairRequestMsg struct {
	Type             string `json:"type"`
	RequestID        string `json:"requestId"`
	ProtocolVersion  int    `json:"protocolVersion"`
	ExtensionVersion string `json:"extensionVersion,omitempty"`
	ClientKind       string `json:"clientKind,omitempty"`
}

// PairResultMsg is the server's response to a pair_request.
type PairResultMsg struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	Token     string `json:"token,omitempty"`
}

// ErrorMsg is sent by the server on auth failure or request errors.
type ErrorMsg struct {
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// SubscribeTasksMsg subscribes to task snapshot push messages from GD.
type SubscribeTasksMsg struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
}

// GDTask is a single task entry as sent inside a task_snapshot message.
type GDTask struct {
	TaskID        string  `json:"taskId"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	Progress      float64 `json:"progress"`
	ReceivedBytes int64   `json:"receivedBytes"`
	FileSize      int64   `json:"fileSize"`
	Speed         int64   `json:"speed"`
	CreatedAt     string  `json:"createdAt"`
	CanPause      bool    `json:"canPause"`
	CanOpenFile   bool    `json:"canOpenFile"`
	CanOpenFolder bool    `json:"canOpenFolder"`
	FileExt       string  `json:"fileExt"`
	PackName      string  `json:"packName"`
	URL           string  `json:"url,omitempty"`
}

// UnmarshalJSON handles createdAt on the BrowserService wire as a JSON number
// (Unix seconds). ISO-8601 strings and millisecond timestamps are still accepted
// for backward compatibility. Parsed values are normalized to RFC3339 UTC; display
// zone is chosen later when mapping to domain tasks.
func (t *GDTask) UnmarshalJSON(data []byte) error {
	// Alias breaks the recursion that would happen if we called json.Unmarshal
	// on *GDTask directly inside this method.
	type alias GDTask
	var raw struct {
		alias
		CreatedAt json.RawMessage `json:"createdAt"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*t = GDTask(raw.alias)

	if len(raw.CreatedAt) == 0 {
		return nil
	}
	if raw.CreatedAt[0] == '"' {
		var s string
		if err := json.Unmarshal(raw.CreatedAt, &s); err != nil {
			return err
		}
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			t.CreatedAt = timefmt.Canonical(parsed)
		} else {
			t.CreatedAt = s
		}
		return nil
	}
	// JSON number — canonical wire format is Unix seconds; some payloads may still
	// send milliseconds. Heuristic: values below 1e10 are seconds (1970–2286);
	// values at or above 1e10 are treated as milliseconds.
	var n int64
	if err := json.Unmarshal(raw.CreatedAt, &n); err != nil {
		return fmt.Errorf("createdAt: cannot parse %s as string or number: %w", raw.CreatedAt, err)
	}
	var ts time.Time
	if n < 10_000_000_000 {
		ts = time.Unix(n, 0)
	} else {
		ts = time.UnixMilli(n)
	}
	t.CreatedAt = timefmt.Canonical(ts)
	return nil
}

// TaskSnapshotMsg is the server push message containing all current tasks.
// The server sends this after subscribe_tasks and again whenever tasks change.
type TaskSnapshotMsg struct {
	Type  string   `json:"type"`
	Tasks []GDTask `json:"tasks"`
}

// TaskAction identifies a lifecycle or UI action on a task.
type TaskAction string

const (
	ActionTogglePause TaskAction = "toggle_pause"
	ActionCancel      TaskAction = "cancel" // deletes files
	ActionRemove      TaskAction = "remove" // keeps files
	ActionRedownload  TaskAction = "redownload"
	ActionOpenFile    TaskAction = "open_file"
	ActionOpenFolder  TaskAction = "open_folder"
)

// TaskActionMsg requests an action on a task.
type TaskActionMsg struct {
	Type      string     `json:"type"`
	RequestID string     `json:"requestId"`
	TaskID    string     `json:"taskId"`
	Action    TaskAction `json:"action"`
}

// TaskActionResultMsg is the server's response to a task_action request.
type TaskActionResultMsg struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	OK        bool   `json:"ok"`
	Message   string `json:"message,omitempty"`
	TaskID    string `json:"taskId,omitempty"`
}

// TaskSource identifies how a task was created.
type TaskSource string

const (
	SourceDownload      TaskSource = "download"
	SourceResource      TaskSource = "resource"
	SourceResourceMerge TaskSource = "resource_merge"
	SourcePageMedia     TaskSource = "page_media"
)

// CreateTaskMsg requests creation of a new task.
type CreateTaskMsg struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId"`
	Source    TaskSource      `json:"source"`
	Title     string          `json:"title,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Draft     bool            `json:"draft,omitempty"`
}

// CreateTaskResultMsg is the server's response to a create_task request.
type CreateTaskResultMsg struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	Status    string `json:"status"` // created | drafted | rejected
	TaskID    string `json:"taskId,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Capabilities describes server features advertised in hello_ack.
type Capabilities struct {
	TaskSnapshots bool         `json:"taskSnapshots"`
	TaskActions   []TaskAction `json:"taskActions"`
}
