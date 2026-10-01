package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/cmd"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/output"
)

func main() {
	factory := cmdutil.NewFactory()
	rootCmd := cmd.NewRootCmd(factory)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		output.WriteCLIError(os.Stdout, os.Stderr, factory.OutputFormat, err)
		os.Exit(1)
	}
}
