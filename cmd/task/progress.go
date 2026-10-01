package task

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func NewCmdProgress(f *cmdutil.Factory) *cobra.Command {
	var watch bool

	cmd := &cobra.Command{
		Use:   "progress <task-id>",
		Short: "Show download progress",
		Long: `Show how far a task has downloaded, remaining size, and ETA.

--watch keeps the line updated from the task snapshot stream until you interrupt it.`,
		Example: `  # One shot
  ghostdl task progress task-k9m3

  # Keep updating until you interrupt
  ghostdl task progress task-k9m3 --watch`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			showDetail, _ := cmd.Flags().GetBool("detail")
			taskID := args[0]

			if watch {
				return watchProgress(cmd.Context(), f, taskID)
			}

			conn, cleanup, err := dialTaskConn(cmd.Context(), f)
			if err != nil {
				return err
			}
			defer cleanup()

			progress, err := api.GetProgress(conn, taskID)
			if err != nil {
				return cmdutil.PreferInterrupt(cmd.Context(), err)
			}

			return output.NewFormatterWithView(f.IOStreams.Out, format, showDetail).Print(progress)
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Keep updating from the task snapshot stream")

	return cmd
}

func watchProgress(ctx context.Context, f *cmdutil.Factory, taskID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	conn, cleanup, err := dialTaskConn(ctx, f)
	if err != nil {
		if ctx.Err() != nil {
			return watchInterrupted(f.IOStreams.Out)
		}
		return err
	}
	defer cleanup()

	stream, err := conn.SubscribeTasksStream(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return watchInterrupted(f.IOStreams.Out)
		}
		return err
	}

	fmt.Fprintf(f.IOStreams.Out,
		"Watching task %s (Ctrl+C to quit)\n\n",
		display.Sanitize(taskID),
	)

	first := true
	prevWidth := 0
	for {
		select {
		case <-ctx.Done():
			return watchInterrupted(f.IOStreams.Out)
		case snap, ok := <-stream:
			if !ok {
				if ctx.Err() != nil {
					return watchInterrupted(f.IOStreams.Out)
				}
				return fmt.Errorf("connection closed while watching task progress")
			}

			var matched bool
			for i := range snap.Tasks {
				if snap.Tasks[i].TaskID != taskID {
					continue
				}
				matched = true
				first = false
				progress := api.ProgressFromGDTask(&snap.Tasks[i])
				prevWidth = renderProgressLine(f, progress, prevWidth)

				switch progress.Status {
				case task.StatusCompleted:
					fmt.Fprintln(f.IOStreams.Out, "\nTask completed")
					return nil
				case task.StatusFailed:
					return fmt.Errorf("task failed: %s", display.Sanitize(taskID))
				}
				break
			}

			if !matched && first {
				return fmt.Errorf("task not found: %s", taskID)
			}
			if !matched {
				return fmt.Errorf("task disappeared: %s", taskID)
			}
		}
	}
}

func watchInterrupted(out io.Writer) error {
	fmt.Fprintln(out, "\nInterrupted")
	return nil
}

func renderProgressLine(f *cmdutil.Factory, progress *task.ProgressResponse, prevWidth int) int {
	bar := buildBar(progress.Progress, 40)
	eta := formatETA(progress.ETA)
	idRunes := []rune(display.Sanitize(progress.TaskID))
	if len(idRunes) > 12 {
		idRunes = idRunes[:12]
	}
	id := string(idRunes)

	line := fmt.Sprintf("  %s | %s | %d/%d (%.1f%%) | ETA: %s",
		id,
		bar,
		progress.Consumed, progress.Total,
		progress.Progress*100,
		eta,
	)
	width := len(line)
	if prevWidth > width {
		line += strings.Repeat(" ", prevWidth-width)
	}
	fmt.Fprintf(f.IOStreams.Out, "\r%s", line)
	return width
}

func buildBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat(barFilled, filled) + strings.Repeat(barEmpty, width-filled)
}

func formatETA(seconds int64) string {
	if seconds <= 0 {
		return "--:--"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%dh%dm", seconds/3600, (seconds%3600)/60)
}
