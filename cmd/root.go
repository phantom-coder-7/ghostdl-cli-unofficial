package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd/config"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd/overview"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd/task"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/binname"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	clilog "github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/log"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
)

var (
	format     string
	detail     bool
	utc        bool
	dryRun     bool
	insecure   bool
	verbose    bool
	logFile    string
	logCleanup func()
)

func NewRootCmd(f *cmdutil.Factory) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   binname.Command,
		Short: "Unofficial Ghost Downloader CLI",
		Long:  `ghostdl-unofficial — control Ghost Downloader from the command line.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			f.Insecure = insecure
			f.OutputFormat = format

			cleanup, setupErr := clilog.Setup(verbose || logFile != "", logFile, f.IOStreams.ErrOut)
			if setupErr != nil {
				return fmt.Errorf("failed to open log file: %w", setupErr)
			}
			logCleanup = cleanup
			defer func() {
				if err != nil && logCleanup != nil {
					logCleanup()
					logCleanup = nil
				}
			}()

			if !cmd.Flags().Changed("format") && f.Config != nil {
				if cfg, loadErr := f.Config(); loadErr == nil {
					if v := cfg.GetString("output.format"); v != "" {
						format = v
					}
				}
			}
			f.OutputFormat = format

			if err = output.ValidateFormat(format); err != nil {
				return err
			}

			if parent := cmd.Parent(); parent != nil {
				if parent.Name() == "auth" || cmd.Name() == "auth" {
					clilog.Default().Debug("skipping login check", "cmd", cmd.Name(), "reason", "auth command")
					return nil
				}
				if parent.Name() == "config" || cmd.Name() == "config" {
					clilog.Default().Debug("skipping login check", "cmd", cmd.Name(), "reason", "config command")
					return nil
				}
			}
			if cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "version" || cmd.Name() == "license" {
				clilog.Default().Debug("skipping login check", "cmd", cmd.Name(), "reason", "exempt command")
				return nil
			}

			store, err := f.AuthStore()
			if err != nil {
				return err
			}
			if !store.IsLoggedIn() {
				clilog.Default().Debug("login check failed", "cmd", cmd.Name(), "status", "not logged in")
				return fmt.Errorf("not logged in — run: %s auth login", binname.Command)
			}
			clilog.Default().Debug("login check passed", "cmd", cmd.Name(), "status", "logged in")
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if logCleanup != nil {
				logCleanup()
				logCleanup = nil
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table",
		"Output format: table, json, ndjson, csv, list")
	rootCmd.PersistentFlags().BoolVar(&detail, "detail", false,
		"Show all fields (overview shows a subset for table, csv, and list)")
	rootCmd.PersistentFlags().BoolVar(&utc, "utc", false,
		"Print timestamps in UTC instead of your local time zone")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false,
		"Print the request that would be sent, without sending it")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false,
		"Skip TLS certificate verification for wss:// URLs (also: config server.insecure); no effect on ws://")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Enable debug logs on stderr")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "",
		"Write debug logs to a file (implies --verbose)")

	rootCmd.AddCommand(auth.NewCmdAuth(f))
	rootCmd.AddCommand(overview.NewCmdOverview(f))
	rootCmd.AddCommand(task.NewCmdTask(f))
	rootCmd.AddCommand(config.NewCmdConfig(f))
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newLicenseCmd())

	return rootCmd
}
