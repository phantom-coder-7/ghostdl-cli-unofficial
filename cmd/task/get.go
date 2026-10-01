package task

import (
	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/timefmt"
)

func NewCmdGet(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <task-id>",
		Short: "Show a task",
		Args:  cobra.ExactArgs(1),
		Example: `  ghostdl task get task-k9m3
  ghostdl task get task-k9m3 --format json
  # Vertical label/value blocks
  ghostdl task get task-k9m3 --format list
  # All fields instead of the overview subset
  ghostdl task get task-k9m3 --detail`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			showDetail, _ := cmd.Flags().GetBool("detail")
			useUTC, _ := cmd.Flags().GetBool("utc")
			loc := timefmt.Location(useUTC)
			conn, cleanup, err := dialTaskConn(cmd.Context(), f)
			if err != nil {
				return err
			}
			defer cleanup()

			t, err := api.GetTask(conn, args[0], loc)
			if err != nil {
				return cmdutil.PreferInterrupt(cmd.Context(), err)
			}

			return output.NewFormatterWithView(f.IOStreams.Out, format, showDetail).Print(t)
		},
	}
	return cmd
}
