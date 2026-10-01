package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

func NewCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var (
		url      string
		filename string
		path     string
		headers  []string
		threads  int
		title    string
		source   string
		draft    bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new download task",
		Long: `Create a download task in Ghost Downloader.

--dry-run prints the wire message without sending it.`,
		Example: `  ghostdl task create --url https://example.com/file.zip
  ghostdl task create --url https://example.com/file.zip --filename archive.zip --threads 8
  ghostdl task create --url https://example.com/file.zip --header "Referer: https://example.com/" --dry-run
  ghostdl task create --url https://example.com/file.zip --source resource --draft`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			showDetail, _ := cmd.Flags().GetBool("detail")

			headerMap, err := api.ParseHeaders(headers)
			if err != nil {
				return err
			}

			req := &task.CreateRequest{
				URL:      url,
				Filename: filename,
				Path:     path,
				Headers:  headerMap,
				Threads:  threads,
				Title:    title,
				Source:   source,
				Draft:    draft,
			}

			msg, err := api.BuildCreateTaskMessage(req)
			if err != nil {
				return err
			}

			if isDryRun(cmd) {
				return output.PrintDryRun(f.IOStreams.Out, format, msg)
			}

			conn, cleanup, err := dialTaskConn(cmd.Context(), f)
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := api.CreateTask(conn, req)
			if err != nil {
				return cmdutil.PreferInterrupt(cmd.Context(), err)
			}

			switch result.Status {
			case "created":
				if format != "table" {
					return output.NewFormatterWithView(f.IOStreams.Out, format, showDetail).Print(result)
				}
				fmt.Fprintf(f.IOStreams.Out, "Task created: %s\n", display.Sanitize(result.TaskID))
				return nil
			case "drafted":
				if format != "table" {
					return output.NewFormatterWithView(f.IOStreams.Out, format, showDetail).Print(result)
				}
				fmt.Fprintf(f.IOStreams.Out,
					"Task drafted — Ghost Downloader is holding it for approval in the desktop UI\n")
				if result.TaskID != "" {
					fmt.Fprintf(f.IOStreams.Out, "Task ID: %s\n", display.Sanitize(result.TaskID))
				}
				if result.Message != "" {
					fmt.Fprintf(f.IOStreams.Out, "%s\n", display.Sanitize(result.Message))
				}
				return nil
			default:
				return fmt.Errorf("unexpected create result status %q", display.Sanitize(result.Status))
			}
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Download URL (required)")
	cmd.Flags().StringVar(&filename, "filename", "", "Target filename")
	cmd.Flags().StringVar(&path, "path", "", "Output folder path")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "HTTP header in \"key: value\" form (repeatable)")
	cmd.Flags().IntVar(&threads, "threads", 0, "Download threads (maps to preBlockNum)")
	cmd.Flags().StringVar(&title, "title", "", "Task title")
	cmd.Flags().StringVar(&source, "source", "resource", "Task source: download or resource")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create as draft awaiting approval in Ghost Downloader")
	cobra.CheckErr(cmd.MarkFlagRequired("url"))

	return cmd
}
