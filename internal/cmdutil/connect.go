package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/binname"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func ClassifyConnectError(serverURL string, err error) *output.ActionableError {
	url := display.Sanitize(serverURL)
	switch {
	case errors.Is(err, ws.ErrUnauthorized):
		return output.NewActionableError(
			"unauthorized",
			"Authentication failed — token is invalid or was regenerated.",
			fmt.Sprintf("Run %s auth login to re-pair with Ghost Downloader.", binname.Command),
		)
	case errors.Is(err, ws.ErrProtocolMismatch):
		return output.NewActionableError(
			"protocol_mismatch",
			"Authentication failed — protocol version mismatch.",
			"Update ghostdl-unofficial to a newer version.",
		)
	case strings.Contains(err.Error(), "failed to connect"):
		return output.NewActionableError(
			"unreachable",
			fmt.Sprintf("Cannot reach Ghost Downloader at %s", url),
			"Is Ghost Downloader running? Is Enable Browser Extension turned on? Settings → Browser Extension → Enable Browser Extension",
		)
	default:
		msg := err.Error()
		return output.NewActionableError("connect_failed", msg, "")
	}
}

func NewNotLoggedInError() *output.ActionableError {
	return output.NewActionableError(
		"not_logged_in",
		"Not logged in",
		fmt.Sprintf("Run %s auth login to pair with Ghost Downloader.", binname.Command),
	)
}
