package config

import (
	"fmt"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdSet(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration option",
		Example: `  ghostdl config set server.url ws://127.0.0.1:14370
  ghostdl config set server.insecure true
  ghostdl config set output.format json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			if err := cfg.Set(args[0], args[1]); err != nil {
				return fmt.Errorf("failed to write configuration: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "%s = %s\n", args[0], args[1])
			return nil
		},
	}
	return cmd
}
