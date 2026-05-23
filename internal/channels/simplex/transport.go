package simplex

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketTransport is the live adapter for a local simplex-chat daemon.
// Tests exercise Channel through the Transport seam and do not dial it.
type WebSocketTransport struct {
	url    string
	dialer *websocket.Dialer

	mu   sync.Mutex
	conn *websocket.Conn
}

func NewWebSocketTransport(wsURL string) *WebSocketTransport {
	return &WebSocketTransport{url: strings.TrimSpace(wsURL), dialer: websocket.DefaultDialer}
}

func (t *WebSocketTransport) Connect(ctx context.Context) error {
	if strings.TrimSpace(t.url) == "" {
		return errors.New("simplex_missing_ws_url")
	}
	conn, _, err := t.dialer.DialContext(ctx, t.url, nil)
	if err != nil {
		return err
	}
	t.mu.Lock()
	old := t.conn
	t.conn = conn
	t.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

func (t *WebSocketTransport) Receive(context.Context) ([]byte, error) {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()
	if conn == nil {
		return nil, errors.New("simplex_transport_not_connected")
	}
	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if messageType != websocket.TextMessage {
		return nil, nil
	}
	return payload, nil
}

func (t *WebSocketTransport) Send(ctx context.Context, payload []byte) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()
	if conn == nil {
		if err := t.Connect(ctx); err != nil {
			return err
		}
		t.mu.Lock()
		conn = t.conn
		t.mu.Unlock()
	}
	if conn == nil {
		return errors.New("simplex_transport_not_connected")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func (t *WebSocketTransport) Health(ctx context.Context) error {
	if strings.TrimSpace(t.url) == "" {
		return errors.New("simplex_missing_ws_url")
	}
	conn, _, err := t.dialer.DialContext(ctx, t.url, nil)
	if err != nil {
		return err
	}
	return conn.Close()
}

func (t *WebSocketTransport) Close(context.Context) error {
	t.mu.Lock()
	conn := t.conn
	t.conn = nil
	t.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close()
}

var (
	userinfoRE = regexp.MustCompile(`(?i)(wss?|https?)://[^/@\s]+@`)
	tokenRE    = regexp.MustCompile(`(?i)(token|secret|key)=([^&\s]+)`)
)

func redactEvidence(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parsed.User = nil
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String()
	}
	value = userinfoRE.ReplaceAllString(value, `${1}://[redacted]@`)
	value = tokenRE.ReplaceAllString(value, `${1}=[REDACTED]`)
	return value
}
