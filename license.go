package main

import (
	_ "embed"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
)

//go:embed LICENSE
var licenseText string

func init() {
	cmd.SetLicenseText(licenseText)
}
