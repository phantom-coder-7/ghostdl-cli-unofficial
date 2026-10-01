package config

import (
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration options",
		Long:  "List all configuration options. Keys are listed in sorted (alphabetical) order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			printSettings(f.IOStreams.Out, cfg.List(), cfg.Path())
			return nil
		},
	}
	return cmd
}
