package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
)

type ActionableError struct {
	Type          string `json:"type"`
	Message       string `json:"message"`
	Hint          string `json:"hint,omitempty"`
	ConsoleURL    string `json:"console_url,omitempty"`
	ServerMessage string `json:"server_message,omitempty"`
}

func (e *ActionableError) Error() string {
	s := fmt.Sprintf("[%s] %s", e.Type, display.Sanitize(e.Message))
	if e.ServerMessage != "" && e.ServerMessage != e.Message {
		s += fmt.Sprintf("\nServer detail: %s", display.Sanitize(e.ServerMessage))
	}
	if e.Hint != "" {
		s += fmt.Sprintf("\nHint: %s", display.Sanitize(e.Hint))
	}
	return s
}

func NewActionableError(typ, message, hint string) *ActionableError {
	return &ActionableError{
		Type:    typ,
		Message: message,
		Hint:    hint,
	}
}

var gdServerMessages = map[string]string{
	"任务不存在":     "task not found",
	"不支持的操作":    "action not supported by this Ghost Downloader version",
	"任务已完成":     "task already completed",
	"当前任务不支持暂停": "this task cannot be paused",
	"文件尚未生成":    "file has not been created yet",
	"目录不存在":     "directory does not exist",
}

func mapGDMessageText(serverMsg string) string {
	if mapped, ok := gdServerMessages[serverMsg]; ok {
		return mapped
	}
	return serverMsg
}

func MapGDServerMessage(serverMsg string) *ActionableError {
	return &ActionableError{
		Type:          "task_action_failed",
		Message:       mapGDMessageText(serverMsg),
		ServerMessage: serverMsg,
	}
}

// MapGDCreateMessage keeps Ghost Downloader fetch timeouts distinct from a CLI
// connection failure — the desktop app reached out to the URL and timed out.
func MapGDCreateMessage(serverMsg string) *ActionableError {
	err := &ActionableError{
		Type:          "create_rejected",
		Message:       mapGDMessageText(serverMsg),
		ServerMessage: serverMsg,
	}
	if isServerSideFetchTimeout(serverMsg) {
		err.Message = "Ghost Downloader could not download from the URL (server-side network timeout)"
		err.Hint = "The CLI connected successfully; Ghost Downloader timed out fetching the URL. Try a smaller file or check network access from the desktop app."
	}
	return err
}

func isServerSideFetchTimeout(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "is_timeout") ||
		strings.Contains(lower, "timedout") ||
		strings.Contains(msg, "TimedOut")
}

type cliErrorEnvelope struct {
	OK    bool             `json:"ok"`
	Error *ActionableError `json:"error"`
}

func WriteCLIError(stdout, stderr io.Writer, format string, err error) {
	if err == nil {
		return
	}
	if format == "json" {
		ae := asActionableError(err)
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(cliErrorEnvelope{OK: false, Error: ae})
		fmt.Fprintf(stderr, "Error: [%s] %s\n", ae.Type, display.Sanitize(ae.Message))
		return
	}
	fmt.Fprintf(stderr, "Error: %v\n", err)
}

func asActionableError(err error) *ActionableError {
	var ae *ActionableError
	if errors.As(err, &ae) {
		return ae
	}
	return &ActionableError{
		Type:    "error",
		Message: err.Error(),
	}
}
