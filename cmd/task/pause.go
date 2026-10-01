package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
)

func NewCmdPause(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause <task-id>",
		Short: "Pause a running download task",
		Long:  "Pause a running task. If the task is already paused, this is a no-op.",
		Args:  cobra.ExactArgs(1),
		Example: `  ghostdl task pause tsk_abc123
  ghostdl task pause tsk_abc123 --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")

			conn, cleanup, err := dialTaskConn(cmd.Context(), f)
			if err != nil {
				return err
			}
			defer cleanup()

			if isDryRun(cmd) {
				performed, notice, msg, err := api.PlanPause(conn, args[0])
				if err != nil {
					return cmdutil.PreferInterrupt(cmd.Context(), err)
				}
				return output.PrintDryRunPreview(f.IOStreams.Out, format, performed, notice, msg)
			}

			performed, notice, err := api.PauseTask(conn, args[0])
			if err != nil {
				return cmdutil.PreferInterrupt(cmd.Context(), err)
			}
			taskID := args[0]
			if performed {
				return printMutationResult(f.IOStreams.Out, format, taskID, "pause", "paused",
					fmt.Sprintf("Task %s paused\n", taskID))
			}
			return printMutationResult(f.IOStreams.Out, format, taskID, "pause", notice,
				fmt.Sprintf("Task %s is %s\n", taskID, notice))
		},
	}
	return cmd
}
