package config

import (
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdConfig(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `View and modify CLI configuration options.
Configuration lives under the ghostdl-unofficial subdirectory of the OS config directory:
  Linux:   $XDG_CONFIG_HOME or ~/.config
  macOS:   ~/Library/Application Support
  Windows: %AppData%
Run 'ghostdl config list' for the exact path. GD_CLI_CONFIG_DIR overrides the directory (may be relative; used as given).
If both the OS config directory and the home directory cannot be determined, the command returns an error.
Keys can be overridden by environment variables with the GD_ prefix (e.g., GD_SERVER_URL).

Keys:
  server.url       BrowserService WebSocket URL (default ws://127.0.0.1:14370)
  server.insecure  Skip TLS certificate verification for wss:// (default false);
                   same effect as global --insecure for that process
  output.format    Default output format`,
	}

	cmd.AddCommand(NewCmdGet(f))
	cmd.AddCommand(NewCmdSet(f))
	cmd.AddCommand(NewCmdUnset(f))
	cmd.AddCommand(NewCmdList(f))

	return cmd
}
