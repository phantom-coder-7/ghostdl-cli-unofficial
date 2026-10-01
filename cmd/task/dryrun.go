package task

import (
	"github.com/spf13/cobra"
)

func isDryRun(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("dry-run")
	return v
}
