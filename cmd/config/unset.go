package config

import (
	"fmt"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdUnset(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a configuration override and restore the default value",
		Long: `Remove a configuration override so the built-in default applies again.

Unlike 'config set <key> ""', which stores an empty string, 'config unset'
completely removes the key from the config file. On the next read the default
value is used instead.`,
		Example: `  ghostdl config unset server.url
  ghostdl config unset server.insecure
  ghostdl config unset output.format`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			key := args[0]
			if err := cfg.Unset(key); err != nil {
				return fmt.Errorf("failed to unset %s: %w", key, err)
			}
			fmt.Fprintf(f.IOStreams.Out, "%s unset (default: %s)\n", key, cfg.GetString(key))
			return nil
		},
	}
	return cmd
}
