package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var licenseText string

// SetLicenseText stores the full license text printed by the license command.
func SetLicenseText(text string) {
	licenseText = text
}

func newLicenseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "license",
		Short: "Print the full license",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), licenseText)
			return err
		},
	}
}
