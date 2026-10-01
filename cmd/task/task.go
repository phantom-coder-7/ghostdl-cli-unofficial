package task

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/cmdutil"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

func dialTaskConn(ctx context.Context, f *cmdutil.Factory) (*ws.Conn, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	store, err := f.AuthStore()
	if err != nil {
		return nil, nil, err
	}

	loginData, err := store.Get()
	if err != nil {
		return nil, nil, cmdutil.NewNotLoggedInError()
	}

	client, err := f.HttpClient()
	if err != nil {
		return nil, nil, err
	}

	conn, err := client.Connect(ctx, loginData)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, cmdutil.PreferInterrupt(ctx, err)
		}
		return nil, nil, cmdutil.ClassifyConnectError(client.ServerURL(), err)
	}

	stopAfterFunc := context.AfterFunc(ctx, func() { _ = conn.Close() })
	cleanup := func() {
		stopAfterFunc()
		_ = conn.Close()
	}
	return conn, cleanup, nil
}

func NewCmdTask(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage download tasks",
		Long: `List, create, and control download tasks in Ghost Downloader.

Writes accept --dry-run to print the wire message without sending it.`,
	}

	cmd.AddCommand(NewCmdCreate(f))
	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdGet(f))
	cmd.AddCommand(NewCmdProgress(f))
	cmd.AddCommand(NewCmdPause(f))
	cmd.AddCommand(NewCmdResume(f))
	cmd.AddCommand(NewCmdDelete(f))
	cmd.AddCommand(NewCmdRestart(f))
	cmd.AddCommand(NewCmdOpen(f))

	return cmd
}
