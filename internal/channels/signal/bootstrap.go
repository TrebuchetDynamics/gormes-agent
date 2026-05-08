package signal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// sseScanner is the minimal bufio.Scanner subset that both the real scanner
// and fake test scanners implement.
type sseScanner interface {
	Scan() bool
	Text() string
	Err() error
}

// ---------------------------------------------------------------------------
// Bootstrap configuration
// ---------------------------------------------------------------------------

// BootstrapConfig holds the minimal config needed to start a signal-cli HTTP
// bridge. In production these values come from SIGNAL_HTTP_URL and SIGNAL_ACCOUNT
// environment variables. Tests inject values directly.
type BootstrapConfig struct {
	HTTPURL string
	Account string
}

// ResolveBootstrapConfig reads the canonical env vars. When env is nil the
// caller intends test/fake mode; the fields stay empty and must be set later.
func ResolveBootstrapConfig(getenv func(string) string) BootstrapConfig {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	return BootstrapConfig{
		HTTPURL: strings.TrimSpace(getenv("SIGNAL_HTTP_URL")),
		Account: strings.TrimSpace(getenv("SIGNAL_ACCOUNT")),
	}
}

// ---------------------------------------------------------------------------
// Fakeable transport interfaces
// ---------------------------------------------------------------------------

// httpRoundTripper is the minimal http.RoundTripper subset needed by the
// bootstrap so tests can inject fake HTTP responses without live signal-cli.
type httpRoundTripper interface {
	RoundTrip(req *http.Request) (*http.Response, error)
}

// clock abstracts time so tests can advance it deterministically.
type clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	NewTicker(d time.Duration) ticker
}

// ticker matches time.Ticker's public API.
type ticker interface {
	C() <-chan time.Time
	Stop()
}

// realClock delegates to the time package.
type realClock struct{}

func (realClock) Now() time.Time                   { return time.Now() }
func (realClock) Since(t time.Time) time.Duration  { return time.Since(t) }
func (realClock) NewTicker(d time.Duration) ticker { return &realTicker{time.NewTicker(d)} }

type realTicker struct{ *time.Ticker }

func (t *realTicker) C() <-chan time.Time { return t.Ticker.C }

// platformLock prevents duplicate signal-cli connections for the same account.
type platformLock interface {
	Acquire(ctx context.Context, kind, key, label string) (release func(), acquired bool)
}

// ---------------------------------------------------------------------------
// SSE types
// ---------------------------------------------------------------------------

// sseEvent is a single parsed SSE data envelope.
type sseEvent struct {
	Data string
}

// sseReader wraps the signal-cli SSE response body and yields parsed events.
type sseReader struct {
	scanner sseScanner
	closer  io.Closer
}

func newSSEReader(body io.ReadCloser) *sseReader {
	return &sseReader{
		scanner: bufio.NewScanner(body),
		closer:  body,
	}
}

// Next returns the next data envelope from the SSE stream. Returns io.EOF when
// the stream ends or is closed. SSE comments (lines starting with ":") are
// consumed but surfaced as events with empty Data so the caller can update
// liveness.
func (r *sseReader) Next() (sseEvent, error) {
	for r.scanner.Scan() {
		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}
		// SSE keepalive comments prove the connection is alive.
		if strings.HasPrefix(line, ":") {
			return sseEvent{}, nil
		}
		if strings.HasPrefix(line, "data:") {
			return sseEvent{Data: strings.TrimSpace(line[5:])}, nil
		}
	}
	if err := r.scanner.Err(); err != nil {
		return sseEvent{}, err
	}
	return sseEvent{}, io.EOF
}

// Close releases the underlying body.
func (r *sseReader) Close() error {
	return r.closer.Close()
}

// ---------------------------------------------------------------------------
// Bootstrap — implements signal.Client backed by signal-cli HTTP JSON-RPC + SSE
// ---------------------------------------------------------------------------

const (
	sseRetryDelayInitial = 2.0
	sseRetryDelayMax     = 60.0
	healthCheckInterval  = 30 * time.Second
	healthStaleThreshold = 120 * time.Second
	typingInterval       = 8 * time.Second
	defaultHTTPTimeout   = 30 * time.Second
	eventsBufferSize     = 64
)

// Bootstrap implements Client by wrapping the signal-cli HTTP daemon.
type Bootstrap struct {
	cfg        BootstrapConfig
	httpClient *http.Client
	clk        clock
	lock       platformLock

	// events is the buffered channel exposed via Events().
	events chan InboundMessage

	// lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// SSE health tracking
	lastSSEActivity time.Time
	activityMu      sync.RWMutex

	closeOnce sync.Once
	closeErr  error

	// For tests: inject a custom SSE dial function. When nil the default
	// HTTP streaming GET is used.
	sseDial func(ctx context.Context, url string) (*sseReader, error)
}

// BootstrapOption configures optional Bootstrap fields.
type BootstrapOption func(*Bootstrap)

// WithBootstrapClock replaces the real clock (test use only).
func WithBootstrapClock(c clock) BootstrapOption {
	return func(b *Bootstrap) { b.clk = c }
}

// WithBootstrapHTTPClient replaces the transport-level HTTP client (test use).
func WithBootstrapHTTPClient(rt httpRoundTripper) BootstrapOption {
	return func(b *Bootstrap) {
		b.httpClient = &http.Client{
			Transport: rt,
			Timeout:   defaultHTTPTimeout,
		}
	}
}

// WithBootstrapLock replaces the platform lock (test use only).
func WithBootstrapLock(l platformLock) BootstrapOption {
	return func(b *Bootstrap) { b.lock = l }
}

// WithBootstrapSSEDial replaces the SSE dial function (test use only).
func WithBootstrapSSEDial(dial func(ctx context.Context, url string) (*sseReader, error)) BootstrapOption {
	return func(b *Bootstrap) { b.sseDial = dial }
}

// NewBootstrap creates a Bootstrap with the given config. Options allow
// injecting fake clock, HTTP client, lock, and SSE dial for tests.
func NewBootstrap(cfg BootstrapConfig, opts ...BootstrapOption) *Bootstrap {
	b := &Bootstrap{
		cfg:  cfg,
		clk:  realClock{},
		lock: &noopLock{},
		done: make(chan struct{}, 1),
		ctx:  context.Background(),
	}
	for _, opt := range opts {
		opt(b)
	}
	if b.httpClient == nil {
		b.httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	b.events = make(chan InboundMessage, eventsBufferSize)
	return b
}

// redactedAccount returns a safe-for-logs version of the account.
func (b *Bootstrap) redactedAccount() string {
	a := b.cfg.Account
	if len(a) <= 4 {
		return "***"
	}
	return a[len(a)-4:]
}

// ---------------------------------------------------------------------------
// Lifecycle — Connect / Close
// ---------------------------------------------------------------------------

// Connect performs health check, acquires the platform lock, and starts the
// SSE listener and health monitor.
func (b *Bootstrap) Connect(ctx context.Context) error {
	if b.cfg.HTTPURL == "" || b.cfg.Account == "" {
		return fmt.Errorf("signal: SIGNAL_HTTP_URL and SIGNAL_ACCOUNT are required")
	}

	// Acquire account-scoped lock to prevent duplicate listeners.
	release, ok := b.lock.Acquire(ctx, "signal-phone", b.cfg.Account, "Signal account")
	if !ok {
		return fmt.Errorf("signal: could not acquire platform lock for account %s", b.redactedAccount())
	}

	// Health check — verify daemon is reachable.
	if err := b.healthCheck(ctx); err != nil {
		release()
		return fmt.Errorf("signal: health check failed: %w", err)
	}

	b.ctx, b.cancel = context.WithCancel(ctx)
	b.lastSSEActivity = b.clk.Now()

	// Start SSE listener.
	go b.sseListener(b.ctx, release)

	// Start health monitor.
	go b.healthMonitor(b.ctx)

	slog.Info("signal: connected", "url", b.cfg.HTTPURL, "account", b.redactedAccount())
	return nil
}

// Close stops the SSE listener and health monitor, releases the lock, and
// closes the events channel.
func (b *Bootstrap) Close() error {
	b.closeOnce.Do(func() {
		if b.cancel != nil {
			b.cancel()
		}
		<-b.done
		close(b.events)
	})
	return b.closeErr
}

// Events returns the inbound message channel.
func (b *Bootstrap) Events() <-chan InboundMessage {
	return b.events
}

// ---------------------------------------------------------------------------
// Health check
// ---------------------------------------------------------------------------

func (b *Bootstrap) healthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.cfg.HTTPURL+"/api/v1/check", nil)
	if err != nil {
		return err
	}
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach signal-cli at %s: %w", b.cfg.HTTPURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// SSE listener
// ---------------------------------------------------------------------------

func (b *Bootstrap) sseListener(ctx context.Context, release func()) {
	defer release()
	defer close(b.done)

	accountEncoded := url.QueryEscape(b.cfg.Account)
	sseURL := fmt.Sprintf("%s/api/v1/events?account=%s", b.cfg.HTTPURL, accountEncoded)
	backoff := time.Duration(sseRetryDelayInitial * float64(time.Second))

	for ctx.Err() == nil {
		reader, err := b.dialSSE(ctx, sseURL)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Warn("signal: SSE connect error, retrying", "err", err, "backoff", backoff)
			b.sleepWithJitter(ctx, backoff)
			backoff = minDuration(backoff*2, time.Duration(sseRetryDelayMax*float64(time.Second)))
			continue
		}

		backoff = time.Duration(sseRetryDelayInitial * float64(time.Second))
		slog.Debug("signal: SSE connected")

		b.processSSEStream(ctx, reader)
		reader.Close()

		if ctx.Err() != nil {
			return
		}
		slog.Warn("signal: SSE disconnected, reconnecting", "backoff", backoff)
		b.sleepWithJitter(ctx, backoff)
		backoff = minDuration(backoff*2, time.Duration(sseRetryDelayMax*float64(time.Second)))
	}
}

func (b *Bootstrap) dialSSE(ctx context.Context, url string) (*sseReader, error) {
	if b.sseDial != nil {
		return b.sseDial(ctx, url)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("SSE stream returned status %d", resp.StatusCode)
	}
	return newSSEReader(resp.Body), nil
}

func (b *Bootstrap) processSSEStream(ctx context.Context, reader *sseReader) {
	for ctx.Err() == nil {
		ev, err := reader.Next()
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				slog.Debug("signal: SSE read error", "err", err)
			}
			return
		}
		// SSE comments indicate liveness.
		if ev.Data == "" {
			b.updateActivity()
			continue
		}
		b.updateActivity()
		b.handleEventEnvelope(ev.Data)
	}
}

func (b *Bootstrap) handleEventEnvelope(dataStr string) {
	var envelope json.RawMessage
	if err := json.Unmarshal([]byte(dataStr), &envelope); err != nil {
		return
	}

	msg := b.parseDataMessage(envelope)
	if msg == nil {
		return
	}

	select {
	case b.events <- *msg:
	case <-b.ctx.Done():
	}
}

// parseDataMessage extracts an InboundMessage from a signal-cli envelope.
// It mirrors the Hermes adapter's _handle_envelope logic.
func (b *Bootstrap) parseDataMessage(envelope json.RawMessage) *InboundMessage {
	var env struct {
		Envelope struct {
			Source       string `json:"source"`
			SourceName   string `json:"sourceName"`
			SourceUUID   string `json:"sourceUuid"`
			DataMessage  *struct {
				Message   string `json:"message"`
				GroupInfo *struct {
					GroupID   string `json:"groupId"`
					GroupName string `json:"groupName"`
				} `json:"groupInfo"`
			} `json:"dataMessage"`
			SyncMessage *struct {
				SentMessage *struct {
					Destination string `json:"destination"`
					Message     string `json:"message"`
					GroupInfo   *struct {
						GroupID   string `json:"groupId"`
						GroupName string `json:"groupName"`
					} `json:"groupInfo"`
				} `json:"sentMessage"`
			} `json:"syncMessage"`
		} `json:"envelope"`
	}

	if err := json.Unmarshal(envelope, &env); err != nil {
		return nil
	}
	e := env.Envelope

	// Filter syncMessage (read receipts, typing) — only promote Note to Self.
	var dm *struct {
		Message   string `json:"message"`
		GroupInfo *struct {
			GroupID   string `json:"groupId"`
			GroupName string `json:"groupName"`
		} `json:"groupInfo"`
	}
	var sender, senderName, senderUUID string

	if e.SyncMessage != nil && e.SyncMessage.SentMessage != nil {
		sm := e.SyncMessage.SentMessage
		if sm.Destination == b.cfg.Account {
			dm = &struct {
				Message   string `json:"message"`
				GroupInfo *struct {
					GroupID   string `json:"groupId"`
					GroupName string `json:"groupName"`
				} `json:"groupInfo"`
			}{
				Message:   sm.Message,
				GroupInfo: sm.GroupInfo,
			}
			sender = b.cfg.Account
			senderName = "Me"
		} else {
			return nil
		}
	} else {
		sender = e.Source
		senderName = e.SourceName
		senderUUID = e.SourceUUID
		if e.DataMessage != nil {
			dm = e.DataMessage
		}
	}

	if dm == nil || sender == "" {
		return nil
	}

	// Self-message filter (skip echo of our own sends, but allow Note to Self).
	if sender == b.cfg.Account && e.SyncMessage == nil {
		return nil
	}

	text := strings.TrimSpace(dm.Message)
	if text == "" {
		return nil
	}

	var chatType ChatType
	var groupID, groupName string
	if dm.GroupInfo != nil && dm.GroupInfo.GroupID != "" {
		chatType = ChatTypeGroup
		groupID = dm.GroupInfo.GroupID
		groupName = dm.GroupInfo.GroupName
	} else {
		chatType = ChatTypeDirect
	}

	msgID := fmt.Sprintf("%d", b.clk.Now().UnixNano())

	return &InboundMessage{
		ChatType:   chatType,
		SenderID:   sender,
		SenderUUID: senderUUID,
		SenderName: senderName,
		GroupID:    groupID,
		GroupName:  groupName,
		MessageID:  msgID,
		Text:       text,
	}
}

// ---------------------------------------------------------------------------
// Health monitor
// ---------------------------------------------------------------------------

func (b *Bootstrap) healthMonitor(ctx context.Context) {
	ticker := b.clk.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
		}

		if b.sinceLastActivity() > healthStaleThreshold {
			slog.Warn("signal: SSE idle, checking daemon health")
			if err := b.healthCheck(ctx); err != nil {
				slog.Warn("signal: health check failed, forcing reconnect", "err", err)
				// Force reconnect by cancelling the SSE context.
				if b.cancel != nil {
					b.cancel()
				}
				return
			}
			// Daemon is healthy but SSE is idle — update activity.
			b.updateActivity()
		}
	}
}

// ---------------------------------------------------------------------------
// JSON-RPC
// ---------------------------------------------------------------------------

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      string          `json:"id"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	ID      string          `json:"id"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (b *Bootstrap) rpc(ctx context.Context, method string, params any) (json.RawMessage, error) {
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("signal: marshal rpc params: %w", err)
	}

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramBytes,
		ID:      fmt.Sprintf("%s_%d", method, b.clk.Now().UnixMilli()),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.HTTPURL+"/api/v1/rpc", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("signal: rpc %s failed: %w", method, err)
	}
	defer resp.Body.Close()

	var rpcResp jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("signal: decode rpc response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("signal: rpc %s error: %s (code %d)", method, rpcResp.Error.Message, rpcResp.Error.Code)
	}

	return rpcResp.Result, nil
}

// ---------------------------------------------------------------------------
// Send implementations
// ---------------------------------------------------------------------------

// sendParams is the JSON-RPC send parameters.
type sendParams struct {
	Account    string  `json:"account"`
	Message    string  `json:"message"`
	Recipient  *string `json:"recipient,omitempty"`
	GroupID    *string `json:"groupId,omitempty"`
	TextStyles *string `json:"textStyles,omitempty"`
}

// stopTypingParams is the JSON-RPC sendTyping stop parameters.
type stopTypingParams struct {
	Account      string  `json:"account"`
	Recipient    *string `json:"recipient,omitempty"`
	GroupID      *string `json:"groupId,omitempty"`
	Typing       bool    `json:"typing"`
	TimestampEnd int64   `json:"timestamp"`
}

type getAttachmentParams struct {
	Account string `json:"account"`
	ID      string `json:"id"`
}

// SendDirect sends a direct message via JSON-RPC.
func (b *Bootstrap) SendDirect(ctx context.Context, recipientID, text string, opts SendOptions) (string, error) {
	return b.send(ctx, recipientID, nil, text, opts)
}

// SendGroup sends a group message via JSON-RPC.
func (b *Bootstrap) SendGroup(ctx context.Context, groupID, text string, opts SendOptions) (string, error) {
	return b.send(ctx, "", &groupID, text, opts)
}

func (b *Bootstrap) send(ctx context.Context, recipientID string, groupID *string, text string, opts SendOptions) (string, error) {
	// Stop typing before sending.
	b.stopTyping(ctx, recipientID, groupID)

	plainText, bodyRanges := MarkdownToSignal(text)
	params := sendParams{
		Account:   b.cfg.Account,
		Message:   plainText,
		TextStyles: buildTextStyles(bodyRanges),
	}

	if groupID != nil && *groupID != "" {
		params.GroupID = groupID
	} else {
		params.Recipient = &recipientID
	}

	result, err := b.rpc(ctx, "send", params)
	if err != nil {
		return "", fmt.Errorf("signal: send failed: %w", err)
	}

	// Extract timestamp as message ID from result.
	var sendResult struct {
		Timestamp int64 `json:"timestamp"`
	}
	var msgID string
	if json.Unmarshal(result, &sendResult) == nil && sendResult.Timestamp > 0 {
		msgID = fmt.Sprintf("%d", sendResult.Timestamp)
	} else {
		msgID = fmt.Sprintf("%d", time.Now().UnixMilli())
	}

	return msgID, nil
}

func (b *Bootstrap) stopTyping(ctx context.Context, recipientID string, groupID *string) {
	params := stopTypingParams{
		Account:      b.cfg.Account,
		Typing:       false,
		TimestampEnd: b.clk.Now().UnixMilli(),
	}
	if groupID != nil && *groupID != "" {
		params.GroupID = groupID
	} else {
		params.Recipient = &recipientID
	}
	_, _ = b.rpc(ctx, "sendTyping", params)
}

// FetchAttachment retrieves and decodes a Signal attachment.
func (b *Bootstrap) FetchAttachment(ctx context.Context, attachmentID string) ([]byte, error) {
	params := getAttachmentParams{
		Account: b.cfg.Account,
		ID:      attachmentID,
	}
	result, err := b.rpc(ctx, "getAttachment", params)
	if err != nil {
		return nil, fmt.Errorf("signal: fetch attachment failed: %w", err)
	}

	var attResult struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &attResult); err != nil {
		return nil, fmt.Errorf("signal: decode attachment response: %w", err)
	}
	if attResult.Data == "" {
		return nil, fmt.Errorf("signal: attachment response missing data key")
	}
	return []byte(attResult.Data), nil
}

// ---------------------------------------------------------------------------
// Helper types and functions
// ---------------------------------------------------------------------------

// noopLock implements platformLock with no actual locking.
type noopLock struct{}

func (noopLock) Acquire(_ context.Context, _, _, _ string) (func(), bool) {
	return func() {}, true
}

func buildTextStyles(ranges []SignalBodyRange) *string {
	if len(ranges) == 0 {
		return nil
	}
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		parts = append(parts, fmt.Sprintf("%d:%d:%s", r.Start, r.Length, r.Style))
	}
	s := strings.Join(parts, ",")
	return &s
}

func (b *Bootstrap) updateActivity() {
	b.activityMu.Lock()
	defer b.activityMu.Unlock()
	b.lastSSEActivity = b.clk.Now()
}

func (b *Bootstrap) sinceLastActivity() time.Duration {
	b.activityMu.RLock()
	defer b.activityMu.RUnlock()
	return b.clk.Since(b.lastSSEActivity)
}

func (b *Bootstrap) sleepWithJitter(ctx context.Context, d time.Duration) {
	jitter := time.Duration(float64(d) * 0.2 * rand.Float64())
	wait := d + jitter
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
