package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func NewCmdOpen(f *cmdutil.Factory) *cobra.Command {
	var folder bool

	cmd := &cobra.Command{
		Use:   "open <task-id>",
		Short: "Open a task's downloaded file or folder",
		Long:  "Open a task's downloaded file or folder via Ghost Downloader (desktop app must be running).",
		Args:  cobra.ExactArgs(1),
		Example: `  # Open the downloaded file
  ghostdl task open tsk_abc123

  # Open the download folder
  ghostdl task open tsk_abc123 --folder

  # Preview the wire message
  ghostdl task open tsk_abc123 --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			taskID := args[0]
			action := ws.ActionOpenFile
			successMsg := "Task %s opened\n"
			if folder {
				action = ws.ActionOpenFolder
				successMsg = "Task %s folder opened\n"
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
			outcome := "opened"
			if folder {
				outcome = "folder opened"
			}
			return printMutationResult(f.IOStreams.Out, format, taskID, "open", outcome,
				fmt.Sprintf(successMsg, taskID))
		},
	}

	cmd.Flags().BoolVar(&folder, "folder", false, "Open the download folder instead of the file")

	return cmd
}
