package cmdutil

import (
	"context"
	"errors"
)

// ErrInterrupted replaces a socket error when the command context is already canceled.
var ErrInterrupted = errors.New("interrupted")

func PreferInterrupt(ctx context.Context, err error) error {
	if err == nil || ctx == nil || ctx.Err() == nil {
		return err
	}
	return ErrInterrupted
}
