package auth

import (
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication management",
		Long: `Login, logout, and view login status.

Pair with Ghost Downloader's BrowserService — either request approval in the
desktop app or paste a pairing token copied from Settings → Browser Extension:

  ghostdl auth login                  Ask Ghost Downloader to approve (popup dialog)
  ghostdl auth login --token <token>  Paste a copied pairing token (no popup)

The CLI stores one credential and connects to a single Ghost Downloader
instance at a time. A still-valid token prints already logged in; --force
pairs again, --token replaces the credential, and logout drops it.`,
	}

	cmd.AddCommand(NewCmdLogin(f))
	cmd.AddCommand(NewCmdLogout(f))
	cmd.AddCommand(NewCmdStatus(f))
	cmd.AddCommand(NewCmdCheck(f))

	return cmd
}
