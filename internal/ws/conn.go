package ws

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/binname"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/display"
	clilog "github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/log"
	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/version"
)

// Sentinel errors returned by hello/Dial/Ping so callers can distinguish failure modes.
var (
	// ErrUnauthorized is returned when the server rejects the token.
	ErrUnauthorized = errors.New("token is invalid or was regenerated — re-pair with: " + binname.Command + " auth login")
	// ErrProtocolMismatch is returned when the server's protocol version doesn't match.
	ErrProtocolMismatch = errors.New("protocol version mismatch — update ghostdl-unofficial")
)

// cliReply carries a flat-JSON RPC response or error for SendGD waiters.
type cliReply struct {
	raw json.RawMessage
	err error
}

// Conn wraps an authenticated WebSocket connection.
//
// Connection strategies:
//   - Short operations (create/list/get/progress/pause/resume/delete/restart/open): each command dials → authenticates via hello → sends/receives → closes
//   - Long operations (progress --watch): dial → authenticate → consume task_snapshot pushes until interrupt or terminal status
type Conn struct {
	raw       *websocket.Conn
	serverURL string

	mu sync.Mutex

	// cliPending maps requestId → channel for flat-JSON GD RPC responses.
	cliPending map[string]chan cliReply
	cliPendMu  sync.Mutex

	// typeHandlers maps message type → channel for GD BrowserService push messages
	// (e.g. task_snapshot) that arrive without a correlating request ID.
	typeHandlers  map[string]chan json.RawMessage
	typeHandlerMu sync.Mutex

	// taskActions lists actions supported by the connected GD server (from hello_ack).
	taskActions []TaskAction

	// appVersion is the Ghost Downloader app version from hello_ack.
	appVersion string

	// readLoopReady is closed once readLoop begins reading; Dial waits on it so
	// callers can register type handlers before the first push message arrives.
	readLoopReady chan struct{}

	closed bool
}

// DialOption configures optional dial behavior (e.g. TLS verification for wss://).
type DialOption func(*dialOptions)

type dialOptions struct {
	insecureSkipVerify bool
}

// WithInsecureSkipVerify skips TLS certificate verification for wss:// URLs.
// It has no effect on ws:// (plaintext) connections.
func WithInsecureSkipVerify(v bool) DialOption {
	return func(o *dialOptions) { o.insecureSkipVerify = v }
}

func applyDialOptions(opts []DialOption) dialOptions {
	var o dialOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

// Dial connects to the WebSocket server and authenticates with a hello handshake.
// serverURL: ws://127.0.0.1:14370 (plaintext) or wss://… (TLS; verifies system CAs unless
// WithInsecureSkipVerify(true) is set). token: pairing token from PairRequest or GD settings.
func Dial(ctx context.Context, serverURL string, token string, opts ...DialOption) (*Conn, error) {
	start := time.Now()
	clilog.Default().Debug("connecting", "server", serverURL)
	raw, err := dialRaw(ctx, serverURL, applyDialOptions(opts).insecureSkipVerify)
	if err != nil {
		elapsed := time.Since(start)
		clilog.Default().Debug("connect failed", "server", serverURL, "error", err.Error(), "elapsed", elapsed.String())
		return nil, err
	}

	stopCancel := context.AfterFunc(ctx, func() { _ = raw.Close() })
	defer stopCancel()

	c := &Conn{
		raw:           raw,
		serverURL:     serverURL,
		cliPending:    make(map[string]chan cliReply),
		typeHandlers:  make(map[string]chan json.RawMessage),
		readLoopReady: make(chan struct{}),
	}

	ack, err := c.hello(token)
	if err != nil {
		_ = raw.Close()
		elapsed := time.Since(start)
		clilog.Default().Debug("hello failed", "server", serverURL, "error", err.Error(), "elapsed", elapsed.String())
		return nil, err
	}
	stopCancel()
	c.storeCapabilities(ack)
	c.appVersion = ack.AppVersion
	elapsed := time.Since(start)
	clilog.Default().Debug("connected", "server", serverURL, "app_version", ack.AppVersion, "elapsed", elapsed.String())

	go c.readLoop()
	<-c.readLoopReady
	return c, nil
}

// storeCapabilities parses hello_ack capabilities and persists taskActions.
func (c *Conn) storeCapabilities(ack *HelloAckMsg) {
	if len(ack.Capabilities) == 0 {
		return
	}
	var caps Capabilities
	if err := json.Unmarshal(ack.Capabilities, &caps); err != nil {
		return
	}
	c.taskActions = caps.TaskActions
}

// TaskActions returns the task actions advertised by the connected GD server.
func (c *Conn) TaskActions() []TaskAction {
	return c.taskActions
}

// AppVersion returns the Ghost Downloader application version from hello_ack.
func (c *Conn) AppVersion() string {
	return c.appVersion
}

// PairRequest opens a raw connection, sends a pair_request, and waits up to 60 s
// for the user to approve the request in Ghost Downloader.
// The returned token should be stored and passed to Dial on future connections.
func PairRequest(ctx context.Context, serverURL string, opts ...DialOption) (string, error) {
	raw, err := dialRaw(ctx, serverURL, applyDialOptions(opts).insecureSkipVerify)
	if err != nil {
		return "", err
	}
	defer raw.Close()

	stopCancel := context.AfterFunc(ctx, func() { _ = raw.Close() })
	defer stopCancel()

	req := PairRequestMsg{
		Type:             TypePairRequest,
		RequestID:        generateID(),
		ProtocolVersion:  ProtocolVersion,
		ExtensionVersion: version.Display(),
		ClientKind:       "cli",
	}
	if err := raw.WriteJSON(req); err != nil {
		return "", fmt.Errorf("failed to send pair_request: %w", err)
	}

	if err := raw.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return "", fmt.Errorf("failed to set read deadline: %w", err)
	}
	var result PairResultMsg
	if err := raw.ReadJSON(&result); err != nil {
		return "", fmt.Errorf("failed to read pair_result: %w", err)
	}

	if !result.OK {
		return "", fmt.Errorf("pairing denied: %s", display.Sanitize(result.Message))
	}
	if result.Token == "" {
		return "", fmt.Errorf("pairing succeeded but server returned no token")
	}
	return result.Token, nil
}

// hello sends a hello message and reads hello_ack or error.
// Returns the full HelloAckMsg on success so callers can surface appVersion/capabilities.
func (c *Conn) hello(token string) (*HelloAckMsg, error) {
	msg := HelloMsg{
		Type:            TypeHello,
		RequestID:       generateID(),
		ProtocolVersion: ProtocolVersion,
		Token:           token,
	}

	c.mu.Lock()
	err := c.raw.WriteJSON(msg)
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to send hello: %w", err)
	}

	if err := c.raw.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	var raw json.RawMessage
	readErr := c.raw.ReadJSON(&raw)
	if clearErr := c.raw.SetReadDeadline(time.Time{}); clearErr != nil && readErr == nil {
		return nil, fmt.Errorf("failed to clear read deadline: %w", clearErr)
	}
	if readErr != nil {
		return nil, fmt.Errorf("failed to read hello response: %w", readErr)
	}

	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &base); err != nil {
		return nil, fmt.Errorf("failed to parse hello response: %w", err)
	}

	switch base.Type {
	case TypeHelloAck:
		var ack HelloAckMsg
		if err := json.Unmarshal(raw, &ack); err != nil {
			return nil, fmt.Errorf("failed to parse hello_ack: %w", err)
		}
		return &ack, nil
	case TypeError:
		var errMsg ErrorMsg
		if err := json.Unmarshal(raw, &errMsg); err != nil {
			return nil, fmt.Errorf("authentication failed (unparseable error response)")
		}
		switch errMsg.Code {
		case "unauthorized":
			clilog.Default().Debug("hello rejected", "code", errMsg.Code)
			return nil, fmt.Errorf("authentication failed: %w", ErrUnauthorized)
		case "protocol_mismatch":
			clilog.Default().Debug("protocol mismatch", "code", errMsg.Code, "expected", ProtocolVersion)
			return nil, fmt.Errorf("authentication failed: %w", ErrProtocolMismatch)
		default:
			clilog.Default().Debug("hello rejected", "code", errMsg.Code)
			return nil, fmt.Errorf("authentication failed: %s", display.Sanitize(errMsg.Code))
		}
	default:
		return nil, fmt.Errorf("unexpected response type %q (expected %s)", display.Sanitize(base.Type), TypeHelloAck)
	}
}

// Ping opens a connection, performs the hello handshake, and immediately closes.
// It returns the HelloAckMsg on success so callers can inspect appVersion/capabilities.
// Use this for connectivity checks; use Dial for operations that require a persistent session.
func Ping(ctx context.Context, serverURL, token string, opts ...DialOption) (*HelloAckMsg, error) {
	start := time.Now()
	clilog.Default().Debug("ping connect", "server", serverURL)
	raw, err := dialRaw(ctx, serverURL, applyDialOptions(opts).insecureSkipVerify)
	if err != nil {
		elapsed := time.Since(start)
		clilog.Default().Debug("ping connect failed", "server", serverURL, "error", err.Error(), "elapsed", elapsed.String())
		return nil, err
	}
	defer raw.Close()

	stopCancel := context.AfterFunc(ctx, func() { _ = raw.Close() })
	defer stopCancel()

	ack, err := (&Conn{raw: raw, serverURL: serverURL}).hello(token)
	if err != nil {
		elapsed := time.Since(start)
		clilog.Default().Debug("ping hello failed", "server", serverURL, "error", err.Error(), "elapsed", elapsed.String())
		return nil, err
	}
	elapsed := time.Since(start)
	clilog.Default().Debug("ping ok", "server", serverURL, "app_version", ack.AppVersion, "elapsed", elapsed.String())
	return ack, nil
}

// dialRaw creates a raw WebSocket connection without authentication.
// For wss://, TLS certificates are verified against system CAs unless insecureSkipVerify is true.
// For ws://, TLS is not used; insecureSkipVerify is ignored.
func dialRaw(ctx context.Context, serverURL string, insecureSkipVerify bool) (*websocket.Conn, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL %s: %w", display.Sanitize(serverURL), err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	if u.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		}
	}

	clilog.Default().Debug("dial raw websocket", "server", serverURL)
	raw, _, err := dialer.DialContext(ctx, serverURL, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", display.Sanitize(serverURL), err)
	}
	clilog.Default().Debug("dial raw ok", "server", serverURL)
	return raw, nil
}

// Close writes a close frame, then closes the raw socket so a blocked read returns.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	_ = c.raw.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_ = c.raw.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	return c.raw.Close()
}

// readLoop continuously reads messages and dispatches them to pending waiters.
//
// Two dispatch paths:
//  1. requestId-based (GD flat JSON RPC): messages with "requestId" route to cliPending.
//  2. Type-based (GD BrowserService push): messages route to typeHandlers[type].
func (c *Conn) readLoop() {
	close(c.readLoopReady)
	for {
		_, msgBytes, err := c.raw.ReadMessage()
		if err != nil {
			clilog.Default().Debug("read loop closed", "server", c.serverURL, "error", err.Error())
			c.cliPendMu.Lock()
			for _, ch := range c.cliPending {
				close(ch)
			}
			c.cliPending = nil
			c.cliPendMu.Unlock()

			c.typeHandlerMu.Lock()
			for k, ch := range c.typeHandlers {
				close(ch)
				delete(c.typeHandlers, k)
			}
			c.typeHandlerMu.Unlock()
			return
		}

		var envelope struct {
			Type      string `json:"type"`
			RequestID string `json:"requestId"`
		}
		if err := json.Unmarshal(msgBytes, &envelope); err != nil {
			clilog.Default().Debug("skipping invalid websocket frame", "reason", err.Error(), "size", len(msgBytes))
			continue
		}
		clilog.Default().Debug("frame received", "type", envelope.Type, "size", len(msgBytes), "request_id", envelope.RequestID)

		if envelope.RequestID != "" {
			c.cliPendMu.Lock()
			if ch, ok := c.cliPending[envelope.RequestID]; ok {
				reply := cliReply{raw: msgBytes}
				if envelope.Type == TypeError {
					var errMsg ErrorMsg
					if err := json.Unmarshal(msgBytes, &errMsg); err != nil {
						reply.err = fmt.Errorf("server error (unparseable): %w", err)
					} else {
						reply.err = fmt.Errorf("server error: %s: %s",
							display.Sanitize(errMsg.Code),
							display.Sanitize(errMsg.Message))
					}
				}
				select {
				case ch <- reply:
				default:
				}
				c.cliPendMu.Unlock()
				continue
			}
			c.cliPendMu.Unlock()
		}

		if envelope.Type != "" {
			c.typeHandlerMu.Lock()
			if ch, ok := c.typeHandlers[envelope.Type]; ok {
				raw := make([]byte, len(msgBytes))
				copy(raw, msgBytes)
				select {
				case ch <- json.RawMessage(raw):
				default:
				}
			}
			c.typeHandlerMu.Unlock()
		}
	}
}

// SendGD sends a flat-JSON GD BrowserService message and waits for the correlated
// response identified by requestId. A top-level error reply with the matching
// requestId is returned as a failure.
func (c *Conn) SendGD(msg any, requestID string, expectType string, timeout time.Duration) (json.RawMessage, error) {
	start := time.Now()
	clilog.Default().Debug("send request", "expect_type", expectType, "request_id", requestID)

	respCh := make(chan cliReply, 1)
	c.cliPendMu.Lock()
	c.cliPending[requestID] = respCh
	c.cliPendMu.Unlock()

	defer func() {
		c.cliPendMu.Lock()
		delete(c.cliPending, requestID)
		c.cliPendMu.Unlock()
	}()

	c.mu.Lock()
	err := c.raw.WriteJSON(msg)
	c.mu.Unlock()
	if err != nil {
		clilog.Default().Debug("send failed", "request_id", requestID, "error", err.Error())
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	select {
	case resp, ok := <-respCh:
		elapsed := time.Since(start)
		if !ok {
			clilog.Default().Debug("request closed", "request_id", requestID, "elapsed", elapsed.String())
			return nil, fmt.Errorf("connection closed while waiting for response")
		}
		if resp.err != nil {
			clilog.Default().Debug("request error", "request_id", requestID, "elapsed", elapsed.String(), "error", resp.err.Error())
			return nil, resp.err
		}
		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(resp.raw, &base); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		if base.Type != expectType {
			clilog.Default().Debug("unexpected response type", "request_id", requestID, "got", base.Type, "expected", expectType, "elapsed", elapsed.String())
			return nil, fmt.Errorf("unexpected response type %q (expected %s)", display.Sanitize(base.Type), expectType)
		}
		clilog.Default().Debug("request ok", "request_id", requestID, "response_type", base.Type, "elapsed", elapsed.String())
		return resp.raw, nil
	case <-time.After(timeout):
		clilog.Default().Debug("request timeout", "request_id", requestID, "timeout", timeout.String())
		return nil, fmt.Errorf("request timed out (%s)", timeout)
	}
}

// SubscribeTasksStream sends subscribe_tasks and yields every task_snapshot push
// until ctx is cancelled or the connection closes. The caller must cancel ctx to
// stop the stream and unregister the handler.
func (c *Conn) SubscribeTasksStream(ctx context.Context) (<-chan *TaskSnapshotMsg, error) {
	rawCh := make(chan json.RawMessage, 4)

	c.typeHandlerMu.Lock()
	c.typeHandlers[TypeTaskSnapshot] = rawCh
	c.typeHandlerMu.Unlock()

	req := SubscribeTasksMsg{
		Type:      TypeSubscribeTasks,
		RequestID: generateID(),
	}
	c.mu.Lock()
	err := c.raw.WriteJSON(req)
	c.mu.Unlock()
	if err != nil {
		c.typeHandlerMu.Lock()
		delete(c.typeHandlers, TypeTaskSnapshot)
		c.typeHandlerMu.Unlock()
		return nil, fmt.Errorf("failed to send subscribe_tasks: %w", err)
	}

	out := make(chan *TaskSnapshotMsg, 4)
	go func() {
		defer close(out)
		defer func() {
			c.typeHandlerMu.Lock()
			delete(c.typeHandlers, TypeTaskSnapshot)
			c.typeHandlerMu.Unlock()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-rawCh:
				if !ok {
					return
				}
				var snap TaskSnapshotMsg
				if err := json.Unmarshal(raw, &snap); err != nil {
					clilog.Default().Debug("skipping invalid task_snapshot frame", "reason", err.Error(), "size", len(raw))
					continue
				}
				snapCopy := snap
				select {
				case out <- &snapCopy:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// SubscribeTasksOnce sends a subscribe_tasks message to Ghost Downloader and waits
// for the first task_snapshot push message, returning it within the given timeout.
//
// Real Ghost Downloader sends task_snapshot without a requestId, so this must not
// use SendGD (which correlates only on requestId). Use this for get, list, and progress reads.
// The caller is responsible for closing the connection when done.
func (c *Conn) SubscribeTasksOnce(timeout time.Duration) (*TaskSnapshotMsg, error) {
	ch := make(chan json.RawMessage, 1)

	c.typeHandlerMu.Lock()
	c.typeHandlers[TypeTaskSnapshot] = ch
	c.typeHandlerMu.Unlock()

	defer func() {
		c.typeHandlerMu.Lock()
		delete(c.typeHandlers, TypeTaskSnapshot)
		c.typeHandlerMu.Unlock()
	}()

	req := SubscribeTasksMsg{
		Type:      TypeSubscribeTasks,
		RequestID: generateID(),
	}
	c.mu.Lock()
	err := c.raw.WriteJSON(req)
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to send subscribe_tasks: %w", err)
	}

	select {
	case raw, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("connection closed while waiting for task_snapshot")
		}
		var snap TaskSnapshotMsg
		if err := json.Unmarshal(raw, &snap); err != nil {
			return nil, fmt.Errorf("failed to parse task_snapshot: %w", err)
		}
		return &snap, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timed out waiting for task_snapshot (%s)", timeout)
	}
}

var idCounter int

func generateID() string {
	idCounter++
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), idCounter)
}
