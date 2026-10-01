package wstest

import (
	"net/http/httptest"
	"strings"
)

// URL converts an httptest server URL (http://...) to a WebSocket URL (ws://...).
func URL(s *httptest.Server) string {
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

// TLSURL converts an httptest TLS server URL (https://...) to a secure WebSocket URL (wss://...).
func TLSURL(s *httptest.Server) string {
	return "wss" + strings.TrimPrefix(s.URL, "https")
}
