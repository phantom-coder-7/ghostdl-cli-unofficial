package task

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/timefmt"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	var (
		statusFlags []string
		packFlags   []string
		extFlags    []string
		query       string
		sortBy      string
		order       string
		limit       int
		offset      int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List existing tasks",
		Long: `List download tasks from Ghost Downloader.

Filter by status: repeat --status or use comma-separated values.
Filter by pack: repeat --pack or use comma-separated values (e.g. --pack bt for BitTorrent).
Filter by extension: repeat --ext or use comma-separated values.
Search with --query across id, name, url, pack name, and file extension.
Sort with --sort and --order.
Without --sort and --order, Ghost Downloader's order (newest first) is kept.`,
		Example: `  # All tasks
  ghostdl task list

  # Filter by status
  ghostdl task list --status running

  # Search and filter
  ghostdl task list --query report
  ghostdl task list --status running,paused
  ghostdl task list --query zip --status completed --sort name
  ghostdl task list --sort size --order desc
  ghostdl task list --order asc

  # BitTorrent tasks
  ghostdl task list --pack bt
  ghostdl task list --pack bt --sort size --order desc

  # JSON output for pipeline processing
  ghostdl task list --format json | jq '.data.tasks | length'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			showDetail, _ := cmd.Flags().GetBool("detail")
			useUTC, _ := cmd.Flags().GetBool("utc")
			loc := timefmt.Location(useUTC)

			var statuses []task.Status
			for _, flagVal := range statusFlags {
				for _, part := range strings.Split(flagVal, ",") {
					part = strings.TrimSpace(part)
					if part != "" {
						statuses = append(statuses, task.Status(part))
					}
				}
			}

			var packs []string
			for _, flagVal := range packFlags {
				for _, part := range strings.Split(flagVal, ",") {
					part = strings.TrimSpace(part)
					if part != "" {
						packs = append(packs, part)
					}
				}
			}

			var exts []string
			for _, flagVal := range extFlags {
				for _, part := range strings.Split(flagVal, ",") {
					part = strings.TrimSpace(part)
					if part != "" {
						exts = append(exts, part)
					}
				}
			}

			listReq := &task.ListRequest{
				Statuses: statuses,
				Packs:    packs,
				Exts:     exts,
				Query:    query,
				Sort:     sortBy,
				Order:    order,
				Limit:    limit,
				Offset:   offset,
			}
			if err := api.ValidateListRequest(listReq); err != nil {
				return err
			}

			conn, cleanup, err := dialTaskConn(cmd.Context(), f)
			if err != nil {
				return err
			}
			defer cleanup()

			resp, err := api.ListTasks(conn, listReq, loc)
			if err != nil {
				return cmdutil.PreferInterrupt(cmd.Context(), err)
			}

			rowsPrinted := len(resp.Tasks)
			if format == "table" || format == "list" {
				if offset > 0 || (limit > 0 && resp.Total > rowsPrinted) {
					fmt.Fprintf(f.IOStreams.ErrOut,
						"Showing %d of %d tasks (use --limit 0 to show all)\n",
						rowsPrinted, resp.Total,
					)
				}
			}

			formatter := output.NewFormatterWithView(f.IOStreams.Out, format, showDetail)
			if format == "json" {
				return formatter.Print(resp)
			}
			return formatter.Print(resp.Tasks)
		},
	}

	cmd.Flags().StringArrayVar(&statusFlags, "status", nil, "Filter by status: waiting, running, paused, completed, failed (repeatable or comma-separated)")
	cmd.Flags().StringArrayVar(&packFlags, "pack", nil, "Filter by pack name (exact match, case-insensitive; repeatable or comma-separated; e.g. bt for BitTorrent)")
	cmd.Flags().StringArrayVar(&extFlags, "ext", nil, "Filter by file extension (exact match, case-insensitive; repeatable or comma-separated; leading dot optional)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Case-insensitive search in id, name, url, pack name, and file extension")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort field: created, id, name, pack (packName alias), progress, received (receivedBytes alias), size, speed, status")
	cmd.Flags().StringVar(&order, "order", "", "Sort order: asc or desc")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number per page")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset")

	return cmd
}
