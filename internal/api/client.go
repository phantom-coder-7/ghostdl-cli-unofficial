package api

import (
	"context"
	"fmt"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/auth"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/config"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/ws"
)

type Client struct {
	serverURL string
	insecure  bool
}

// NewClient reads server.url from config. flagInsecure is --insecure; either
// that or server.insecure skips TLS verify on wss://.
func NewClient(cfg *config.Manager, flagInsecure bool) (*Client, error) {
	serverURL := cfg.GetString("server.url")
	if serverURL == "" {
		return nil, fmt.Errorf("configuration missing server.url")
	}
	return &Client{
		serverURL: serverURL,
		insecure:  flagInsecure || cfg.GetBool("server.insecure"),
	}, nil
}

func NewClientForURL(serverURL string) *Client {
	return &Client{serverURL: serverURL}
}

func (c *Client) Connect(ctx context.Context, loginData *auth.LoginData) (*ws.Conn, error) {
	return ws.Dial(ctx, c.serverURL, loginData.Token, ws.WithInsecureSkipVerify(c.insecure))
}

func (c *Client) ValidateLogin(ctx context.Context, loginData *auth.LoginData) error {
	conn, err := c.Connect(ctx, loginData)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// Ping does a hello handshake and closes the socket so it is safe for reachability checks.
func (c *Client) Ping(ctx context.Context, loginData *auth.LoginData) (*ws.HelloAckMsg, error) {
	return ws.Ping(ctx, c.serverURL, loginData.Token, ws.WithInsecureSkipVerify(c.insecure))
}

func (c *Client) ServerURL() string {
	return c.serverURL
}

func (c *Client) InsecureSkipVerify() bool {
	return c.insecure
}
