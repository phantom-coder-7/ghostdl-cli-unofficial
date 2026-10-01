package overview

import (
	"fmt"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/binname"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	overviewpkg "github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/overview"
)

const inProgressHumanCap = 20

func printOverviewHuman(f *cmdutil.Factory, formatter *output.Formatter, resp *overviewpkg.Response) error {
	out := f.IOStreams.Out

	fmt.Fprintln(out, "Overview")
	fmt.Fprintf(out, "  Server:  %s\n", display.Sanitize(resp.Auth.ServerURL))
	if resp.Auth.AppVersion != "" {
		fmt.Fprintf(out, "  App:     Ghost Downloader %s\n", display.Sanitize(resp.Auth.AppVersion))
	}
	fmt.Fprintf(out, "  CLI:     %s\n", resp.Auth.CLIVersion)
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Counts:  waiting=%d  running=%d  paused=%d  completed=%d  failed=%d\n",
		resp.Counts.Waiting, resp.Counts.Running, resp.Counts.Paused, resp.Counts.Completed, resp.Counts.Failed)
	fmt.Fprintf(out, "Traffic: download %d B/s  ·  received (running) %d B  ·  bt running=%d\n",
		resp.DownloadRateBytesPerSec, resp.ReceivedBytesRunning, resp.BtRunningCount)
	fmt.Println()

	inProgressN := len(resp.InProgress)
	fmt.Fprintf(out, "In progress (%d)\n", inProgressN)
	displayInProgress := resp.InProgress
	var truncated int
	if len(displayInProgress) > inProgressHumanCap {
		displayInProgress = displayInProgress[:inProgressHumanCap]
		truncated = inProgressN - inProgressHumanCap
	}
	if err := formatter.Print(displayInProgress); err != nil {
		return err
	}
	if truncated > 0 {
		fmt.Fprintf(out, "… and %d more — use: %s task list --status waiting,running,paused\n", truncated, binname.Command)
	}
	fmt.Println()

	failedShown := len(resp.Failed)
	fmt.Fprintf(out, "Failed (showing %d of %d)\n", failedShown, resp.Counts.Failed)
	if err := formatter.Print(resp.Failed); err != nil {
		return err
	}
	fmt.Println()

	completedShown := len(resp.RecentCompleted)
	fmt.Fprintf(out, "Recent completed (showing %d of %d)\n", completedShown, resp.Counts.Completed)
	if err := formatter.Print(resp.RecentCompleted); err != nil {
		return err
	}

	return nil
}
