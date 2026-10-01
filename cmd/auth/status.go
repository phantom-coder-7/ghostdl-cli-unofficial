package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
)

func NewCmdStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show whether this CLI is paired",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := f.AuthStore()
			if err != nil {
				return err
			}

			if !store.IsLoggedIn() {
				fmt.Fprintln(f.IOStreams.Out, "Status: Not logged in")
				fmt.Fprintln(f.IOStreams.Out, "Hint:   run 'ghostdl auth login' to pair with Ghost Downloader")
				return nil
			}

			data, err := store.Get()
			if err != nil {
				fmt.Fprintln(f.IOStreams.Out, "Status: Unknown (failed to read credential)")
				return nil
			}

			masked := ""
			if len(data.Token) > 12 {
				masked = data.Token[:8] + "..." + data.Token[len(data.Token)-4:]
			} else if data.Token != "" {
				masked = "***"
			}

			client, _ := f.HttpClient()
			serverURL := "(Unknown)"
			if client != nil {
				serverURL = client.ServerURL()
			}

			fmt.Fprintf(f.IOStreams.Out, "Status: Logged in\n")
			fmt.Fprintf(f.IOStreams.Out, "Server: %s\n", display.Sanitize(serverURL))
			if masked != "" {
				fmt.Fprintf(f.IOStreams.Out, "Token:  %s\n", display.Sanitize(masked))
			}
			return nil
		},
	}
	return cmd
}
