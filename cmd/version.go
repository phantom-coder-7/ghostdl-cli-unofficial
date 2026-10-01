package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and build information",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			formatVal, _ := cmd.Flags().GetString("format")
			showDetail, _ := cmd.Flags().GetBool("detail")
			switch formatVal {
			case "json":
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(version.Info())
			case "ndjson":
				b, err := json.Marshal(version.Info())
				if err != nil {
					return err
				}
				fmt.Fprintln(out, string(b))
				return nil
			case "csv", "list":
				return output.NewFormatterWithView(out, formatVal, showDetail).Print(version.Info())
			default:
				fmt.Fprint(out, version.String())
				return nil
			}
		},
	}
}
