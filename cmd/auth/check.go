package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
)

func NewCmdCheck(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Verify the server is reachable and the stored token is valid",
		Long: `Open a WebSocket connection to Ghost Downloader and perform a hello handshake.

Exits 0 on success. Exits 1 if the server is unreachable, the token is missing,
or authentication fails.`,
		Example: `  # Check connectivity using stored credentials and the default server:
  ghostdl auth check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := f.AuthStore()
			if err != nil {
				return err
			}

			if !store.IsLoggedIn() {
				return cmdutil.NewNotLoggedInError()
			}

			loginData, err := store.Get()
			if err != nil {
				return fmt.Errorf("failed to read stored credentials: %w", err)
			}

			client, err := f.HttpClient()
			if err != nil {
				return err
			}
			serverURL := client.ServerURL()

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			ack, err := client.Ping(ctx, loginData)
			if err != nil {
				if ctx.Err() != nil {
					return cmdutil.PreferInterrupt(ctx, err)
				}
				return cmdutil.ClassifyConnectError(serverURL, err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Server is up and authenticated.\n")
			fmt.Fprintf(f.IOStreams.Out, "  Server:  %s\n", display.Sanitize(serverURL))
			if ack.AppVersion != "" {
				fmt.Fprintf(f.IOStreams.Out, "  App:     Ghost Downloader %s\n", display.Sanitize(ack.AppVersion))
			}
			return nil
		},
	}
	return cmd
}
