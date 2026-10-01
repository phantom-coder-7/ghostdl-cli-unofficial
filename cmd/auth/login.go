package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func NewCmdLogin(f *cmdutil.Factory) *cobra.Command {
	var token string
	var force bool
	var allowCredentialFile bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Ghost Downloader",
		Long: `Link this CLI to Ghost Downloader's BrowserService on localhost.

Two ways to pair — use whichever fits the moment:

  Pairing request (interactive)
    With no flags, a stored token that still works prints already logged in
    and returns. Otherwise the CLI sends a pairing request and
    Ghost Downloader shows an approval dialog. Click Allow within 60 seconds.
    Best when you're at the machine and don't mind a popup. --force opens
    that dialog again even when the stored token still works.

  Copied pairing token (no popup)
    Copy the value from Ghost Downloader → Settings → Browser Extension →
    Pairing Token and pass it with --token. Skips the approval dialog and
    replaces the stored credential — handy for scripts, headless setups, or
    when you'd rather not chase a popup. --force does not apply to this path.

Both paths store one credential in the OS keychain when that service is
available. Pass --allow-credential-file to write the user-only credential
file login-data when the OS keychain isn't reachable; that file's weaker
than the OS keychain. If the keychain's there but refuses access, login
errors — it won't write the file. This CLI holds a single session: it
talks to one Ghost Downloader instance at a time. Logout is how you drop
the credential. A rejected token still re-pairs on the next plain login.

Requires Ghost Downloader running with Browser Extension enabled. Default
server: ws://127.0.0.1:14370 (override with ghostdl config set server.url).`,
		Example: `  # Request approval; a still-valid token prints already logged in:
  ghostdl auth login

  # Open the approval dialog again when the stored token still works:
  ghostdl auth login --force

  # Paste a token from Settings → Browser Extension (replaces the credential):
  ghostdl auth login --token <pairing-token>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			store, err := f.AuthStore()
			if err != nil {
				return err
			}
			store.AllowCredentialFile = allowCredentialFile

			client, err := f.HttpClient()
			if err != nil {
				return err
			}
			serverURL := client.ServerURL()

			var pairToken string

			if token != "" {
				if err := store.MissingKeychainError(); err != nil {
					return err
				}
				pairToken = token
				loginData := &auth.LoginData{Kind: auth.KindToken, Token: pairToken}
				if err := client.ValidateLogin(ctx, loginData); err != nil {
					if ctx.Err() != nil {
						return cmdutil.PreferInterrupt(ctx, err)
					}
					return fmt.Errorf("token verification failed: %w\n  Server: %s", err, display.Sanitize(serverURL))
				}
			} else {
				if !force {
					if existing, getErr := store.Get(); getErr == nil && existing != nil && existing.Token != "" {
						if err := client.ValidateLogin(ctx, existing); err != nil {
							if ctx.Err() != nil {
								return cmdutil.PreferInterrupt(ctx, err)
							}
						} else {
							fmt.Fprintf(f.IOStreams.Out, "Already logged in\n  Server: %s\n", display.Sanitize(serverURL))
							return nil
						}
					}
				}

				if err := store.MissingKeychainError(); err != nil {
					return err
				}

				fmt.Fprintf(f.IOStreams.Out, "Waiting for approval in Ghost Downloader (60 s timeout)…\n")
				fmt.Fprintf(f.IOStreams.Out, "  Server: %s\n", display.Sanitize(serverURL))
				fmt.Fprintf(f.IOStreams.Out, "  → Open Ghost Downloader and approve the pairing request\n")

				pairToken, err = ws.PairRequest(ctx, serverURL, ws.WithInsecureSkipVerify(client.InsecureSkipVerify()))
				if err != nil {
					if ctx.Err() != nil {
						return cmdutil.PreferInterrupt(ctx, err)
					}
					return fmt.Errorf("pairing failed: %w", err)
				}
			}

			loginData := &auth.LoginData{
				Kind:  auth.KindToken,
				Token: pairToken,
			}
			if err := store.Save(loginData); err != nil {
				if errors.Is(err, auth.ErrCredentialFileOptIn) {
					return err
				}
				return fmt.Errorf("failed to save credential: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Login successful\n  Server: %s\n", display.Sanitize(serverURL))
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "",
		"Pairing token from Ghost Downloader → Settings → Browser Extension → Pairing Token (skips approval dialog)")
	cmd.Flags().BoolVar(&force, "force", false,
		"Pair again even when the stored token still works")
	cmd.Flags().BoolVar(&allowCredentialFile, "allow-credential-file", false,
		"Write the user-only credential file login-data when the OS keychain cannot be reached (weaker than the OS keychain)")

	return cmd
}
