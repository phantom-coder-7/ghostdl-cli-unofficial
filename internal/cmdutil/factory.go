package cmdutil

import (
	"io"
	"os"
	"sync"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/api"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/config"
)

type Factory struct {
	IOStreams *IOStreams

	// Insecure is --insecure, combined with server.insecure inside api.NewClient.
	Insecure bool

	// OutputFormat is the resolved --format for error printing from main.
	OutputFormat string

	Config     func() (*config.Manager, error)
	AuthStore  func() (*auth.Store, error)
	HttpClient func() (*api.Client, error)
}

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

func NewFactory() *Factory {
	f := &Factory{
		IOStreams: &IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}

	f.Config = sync.OnceValues(func() (*config.Manager, error) {
		return config.NewManager()
	})

	f.AuthStore = sync.OnceValues(auth.NewStore)

	f.HttpClient = sync.OnceValues(func() (*api.Client, error) {
		cfg, err := f.Config()
		if err != nil {
			return nil, err
		}
		return api.NewClient(cfg, f.Insecure)
	})

	return f
}
