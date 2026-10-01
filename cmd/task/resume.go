package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
)

func NewCmdResume(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume <task-id>",
		Short: "Resume a paused download task",
		Long:  "Resume a paused task. If the task is already running, this is a no-op.",
		Args:  cobra.ExactArgs(1),
		Example: `  ghostdl task resume tsk_abc123
  ghostdl task resume tsk_abc123 --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")

			conn, cleanup, err := dialTaskConn(cmd.Context(), f)
			if err != nil {
				return err
			}
			defer cleanup()

			if isDryRun(cmd) {
				performed, notice, msg, err := api.PlanResume(conn, args[0])
				if err != nil {
					return cmdutil.PreferInterrupt(cmd.Context(), err)
				}
				return output.PrintDryRunPreview(f.IOStreams.Out, format, performed, notice, msg)
			}

			performed, notice, err := api.ResumeTask(conn, args[0])
			if err != nil {
				return cmdutil.PreferInterrupt(cmd.Context(), err)
			}
			taskID := args[0]
			if performed {
				return printMutationResult(f.IOStreams.Out, format, taskID, "resume", "resumed",
					fmt.Sprintf("Task %s resumed\n", taskID))
			}
			return printMutationResult(f.IOStreams.Out, format, taskID, "resume", notice,
				fmt.Sprintf("Task %s is %s\n", taskID, notice))
		},
	}
	return cmd
}
