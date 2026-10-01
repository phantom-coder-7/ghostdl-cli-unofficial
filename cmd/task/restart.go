package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func NewCmdRestart(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart <task-id>",
		Short: "Restart a download task (redownload)",
		Long:  "Restart a task by requesting a redownload from Ghost Downloader.",
		Args:  cobra.ExactArgs(1),
		Example: `  ghostdl task restart tsk_abc123
  ghostdl task restart tsk_abc123 --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")

			if isDryRun(cmd) {
				msg := api.BuildTaskActionMessage(args[0], ws.ActionRedownload)
				return output.PrintDryRun(f.IOStreams.Out, format, msg)
			}

			conn, cleanup, err := dialTaskConn(cmd.Context(), f)
			if err != nil {
				return err
			}
			defer cleanup()

			if err := api.TaskAction(conn, args[0], ws.ActionRedownload); err != nil {
				return cmdutil.PreferInterrupt(cmd.Context(), err)
			}
			taskID := args[0]
			return printMutationResult(f.IOStreams.Out, format, taskID, "restart", "restart requested",
				fmt.Sprintf("Task %s restart requested\n", taskID))
		},
	}
	return cmd
}
