package task

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var withFiles bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <task-id>",
		Short: "Remove a task from Ghost Downloader",
		Long: `Remove a task from the download queue.

By default, downloaded files are kept on disk and only the task entry is removed.
Use --with-files to delete the downloaded files as well.`,
		Args: cobra.ExactArgs(1),
		Example: `  ghostdl task delete tsk_abc123
  ghostdl task delete tsk_abc123 --with-files --yes
  ghostdl task delete tsk_abc123 --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			taskID := args[0]
			action := ws.ActionRemove
			actionLabel := "removed task entry, kept downloaded files"

			if withFiles {
				action = ws.ActionCancel
				actionLabel = "removed task entry and deleted downloaded files"
				if !isDryRun(cmd) {
					if err := confirmDelete(f, taskID, yes); err != nil {
						return err
					}
				}
			}

			if isDryRun(cmd) {
				msg := api.BuildTaskActionMessage(taskID, action)
				return output.PrintDryRun(f.IOStreams.Out, format, msg)
			}

			conn, cleanup, err := dialTaskConn(cmd.Context(), f)
			if err != nil {
				return err
			}
			defer cleanup()

			if err := api.TaskAction(conn, taskID, action); err != nil {
				return cmdutil.PreferInterrupt(cmd.Context(), err)
			}
			return printMutationResult(f.IOStreams.Out, format, taskID, "delete", actionLabel,
				fmt.Sprintf("Task %s: %s\n", taskID, actionLabel))
		},
	}

	cmd.Flags().BoolVar(&withFiles, "with-files", false, "Also delete downloaded files from disk")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt for --with-files")

	return cmd
}

func confirmDelete(f *cmdutil.Factory, taskID string, yes bool) error {
	if yes {
		return nil
	}
	if !isTerminal(f.IOStreams.In) {
		return fmt.Errorf("--with-files requires --yes when stdin is not a terminal")
	}

	fmt.Fprintf(f.IOStreams.Out,
		"Delete task %s AND its downloaded files? This cannot be undone. [y/N]: ",
		taskID,
	)

	reader := bufio.NewReader(f.IOStreams.In)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("delete cancelled")
	}
	return nil
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
