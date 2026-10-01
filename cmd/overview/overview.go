package overview

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/timefmt"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/version"
	overviewpkg "github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/overview"
)

func NewCmdOverview(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Print auth, task counts, in-progress and recent tasks, and download traffic",
		Long: `Open one WebSocket session to Ghost Downloader, read hello and a single task snapshot,
then print counts, aggregate download traffic from running tasks, and three task slices
(in progress, failed, recent completed). Use ghostdl task list for full filtered listings.`,
		Example: `  ghostdl overview
  ghostdl overview --limit 10 --format json
  ghostdl overview --detail`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			showDetail, _ := cmd.Flags().GetBool("detail")
			useUTC, _ := cmd.Flags().GetBool("utc")
			loc := timefmt.Location(useUTC)

			if limit < 0 {
				return fmt.Errorf("limit must be zero or greater")
			}

			store, err := f.AuthStore()
			if err != nil {
				return err
			}
			loginData, err := store.Get()
			if err != nil {
				return err
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

			conn, err := client.Connect(ctx, loginData)
			if err != nil {
				if ctx.Err() != nil {
					return cmdutil.PreferInterrupt(ctx, err)
				}
				return cmdutil.ClassifyConnectError(serverURL, err)
			}
			stopAfterFunc := context.AfterFunc(ctx, func() { _ = conn.Close() })
			defer func() {
				stopAfterFunc()
				_ = conn.Close()
			}()

			snap, err := conn.SubscribeTasksOnce(15 * time.Second)
			if err != nil {
				return cmdutil.PreferInterrupt(ctx, err)
			}

			resp, err := api.BuildOverview(snap.Tasks, limit, loc)
			if err != nil {
				return err
			}

			resp.Auth = overviewpkg.AuthBlock{
				ServerURL:  serverURL,
				AppVersion: conn.AppVersion(),
				CLIVersion: version.Display(),
			}

			formatter := output.NewFormatterWithView(f.IOStreams.Out, format, showDetail)
			switch format {
			case "json", "ndjson":
				return formatter.Print(resp)
			default:
				return printOverviewHuman(f, formatter, resp)
			}
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum failed and recent completed tasks to show")

	return cmd
}
