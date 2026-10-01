package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
)

func NewCmdLogout(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout and clear local credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := f.AuthStore()
			if err != nil {
				return err
			}

			if !store.IsLoggedIn() {
				fmt.Fprintln(f.IOStreams.Out, "Already in logged out state")
				return nil
			}

			if err := store.Delete(); err != nil {
				return fmt.Errorf("Failed to clear credentials: %w", err)
			}

			fmt.Fprintln(f.IOStreams.Out, "Logged out")
			return nil
		},
	}
	return cmd
}
