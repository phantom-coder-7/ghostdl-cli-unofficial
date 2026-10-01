package config

import (
	"fmt"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdGet(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [key]",
		Short: "View a configuration option (list all if key is not specified)",
		Long:  "View a configuration option. With no key, lists all options in sorted (alphabetical) order.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				printSettings(f.IOStreams.Out, cfg.List(), cfg.Path())
				return nil
			}

			fmt.Fprintln(f.IOStreams.Out, cfg.GetString(args[0]))
			return nil
		},
	}
	return cmd
}
