package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// DryRunPreview describes a mutating operation that would or would not be sent.
type DryRunPreview struct {
	WouldSend bool            `json:"wouldSend"`
	Message   json.RawMessage `json:"message,omitempty"`
	Notice    string          `json:"notice,omitempty"`
}

// PrintDryRun renders the exact wire message that would be sent over the socket.
func PrintDryRun(w io.Writer, format string, msg any) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal dry-run message: %w", err)
	}
	return printDryRunJSON(w, format, raw)
}

// PrintDryRunPreview renders whether a message would be sent, including no-op notices.
func PrintDryRunPreview(w io.Writer, format string, wouldSend bool, notice string, msg any) error {
	preview := DryRunPreview{
		WouldSend: wouldSend,
		Notice:    notice,
	}
	if wouldSend && msg != nil {
		raw, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal dry-run message: %w", err)
		}
		preview.Message = raw
	}
	return NewFormatter(w, format).Print(preview)
}

func printDryRunJSON(w io.Writer, format string, raw []byte) error {
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("failed to parse dry-run message: %w", err)
	}

	switch format {
	case "json":
		return NewFormatter(w, format).Print(parsed)
	default:
		pretty, err := json.MarshalIndent(parsed, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format dry-run message: %w", err)
		}
		fmt.Fprintln(w, "DRY-RUN — would send:")
		fmt.Fprintln(w, string(pretty))
		return nil
	}
}
