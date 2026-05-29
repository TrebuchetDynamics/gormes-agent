package qqbot

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	QQMaxMessageLength = 4000
	QQMsgTypeText      = 0
	QQMsgTypeMarkdown  = 2
)

// BootstrapOptions collects fakeable QQ Bot transport dependencies. Production
// wiring can provide real clients later without changing the tested contract.
type BootstrapOptions struct {
	AppID           string
	ClientSecret    string
	MarkdownSupport *bool
	DMPolicy        string
	AllowFrom       any
	GroupPolicy     string
	GroupAllowFrom  any
	Getenv          func(string) string

	Dependencies  DependencyStatus
	Locker        QQAppLocker
	TokenClient   TokenClient
	GatewayClient GatewayClient
	Websocket     WebsocketClient
	RESTClient    RESTClient
	Bot           *Bot

	ReconnectWait    time.Duration
	ReconnectPoll    time.Duration
	WaitForReconnect func(context.Context, time.Duration, time.Duration) bool

	SafeURL         func(string) bool
	VoiceTranscribe func(context.Context, VoiceAttachment) VoiceResult
}

type BootstrapConfig struct {
	AppID           string
	ClientSecret    string
	MarkdownSupport bool
	DMPolicy        string
	AllowFrom       []string
	GroupPolicy     string
	GroupAllowFrom  []string
}

type DependencyStatus struct {
	OK        bool
	Code      string
	Message   string
	Retryable bool
}

type BootstrapStatus struct {
	OK        bool
	Code      string
	Message   string
	Retryable bool
}

type BootstrapState struct {
	Connected         bool
	SessionID         string
	LastSeq           int
	HeartbeatInterval time.Duration
}

type Token struct {
	AccessToken string
	ExpiresIn   time.Duration
}

type TokenClient interface {
	FetchToken(context.Context, string, string) (Token, error)
}

type GatewayClient interface {
	FetchGatewayURL(context.Context, string) (string, error)
}

type WebsocketClient interface {
	Open(context.Context, string) error
	Send(context.Context, map[string]any) error
}

type RESTClient interface {
	SendText(context.Context, string, string, QQTextBody) (string, error)
}

type QQAppLocker interface {
	AcquireQQApp(context.Context, string) (func(), bool)
}

type Bootstrap struct {
	cfg  BootstrapConfig
	deps DependencyStatus

	locker        QQAppLocker
	releaseLock   func()
	tokenClient   TokenClient
	gatewayClient GatewayClient
	ws            WebsocketClient
	rest          RESTClient
	bot           *Bot

	reconnectWait    time.Duration
	reconnectPoll    time.Duration
	waitForReconnect func(context.Context, time.Duration, time.Duration) bool
	safeURL          func(string) bool
	voiceTranscribe  func(context.Context, VoiceAttachment) VoiceResult

	mu                sync.Mutex
	connected         bool
	sessionID         string
	lastSeq           int
	heartbeatInterval time.Duration
}

type QQTextBody struct {
	Content          string
	Markdown         map[string]string
	MsgType          int
	MsgSeq           int
	MessageReference map[string]string
}

type QQSendResult struct {
	Success   bool
	MessageID string
	Code      string
	Error     string
	Retryable bool
}

type VoiceAttachment struct {
	URL          string
	VoiceWAVURL  string
	ContentType  string
	Filename     string
	ASRReferText string
}

type VoiceResult struct {
	OK         bool
	Transcript string
	Code       string
	Message    string
	Retryable  bool
}

type QQCloseError struct {
	Code   int
	Reason string
}

func (e QQCloseError) Error() string {
	return fmt.Sprintf("QQ Bot websocket closed (code=%d reason=%s)", e.Code, e.Reason)
}

func NewQQCloseError(code int, reason string) error {
	return QQCloseError{Code: code, Reason: reason}
}

func ResolveBootstrapConfig(opts BootstrapOptions) BootstrapConfig {
	getenv := opts.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	markdown := true
	if opts.MarkdownSupport != nil {
		markdown = *opts.MarkdownSupport
	}
	return BootstrapConfig{
		AppID:           firstNonEmpty(opts.AppID, getenv("QQ_APP_ID")),
		ClientSecret:    firstNonEmpty(opts.ClientSecret, getenv("QQ_CLIENT_SECRET")),
		MarkdownSupport: markdown,
		DMPolicy:        normalizedPolicy(opts.DMPolicy),
		AllowFrom:       CoerceList(opts.AllowFrom),
		GroupPolicy:     normalizedPolicy(opts.GroupPolicy),
		GroupAllowFrom:  CoerceList(opts.GroupAllowFrom),
	}
}

func CoerceList(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return splitList(v)
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func NewBootstrap(opts BootstrapOptions) *Bootstrap {
	cfg := ResolveBootstrapConfig(opts)
	deps := opts.Dependencies
	if deps.Code == "" && deps.Message == "" {
		deps.OK = true
	}
	wait := opts.ReconnectWait
	if wait <= 0 {
		wait = 15 * time.Second
	}
	poll := opts.ReconnectPoll
	if poll <= 0 {
		poll = 250 * time.Millisecond
	}
	safe := opts.SafeURL
	if safe == nil {
		safe = DefaultSafeURL
	}
	return &Bootstrap{
		cfg:               cfg,
		deps:              deps,
		locker:            opts.Locker,
		tokenClient:       opts.TokenClient,
		gatewayClient:     opts.GatewayClient,
		ws:                opts.Websocket,
		rest:              opts.RESTClient,
		bot:               opts.Bot,
		reconnectWait:     wait,
		reconnectPoll:     poll,
		waitForReconnect:  opts.WaitForReconnect,
		safeURL:           safe,
		voiceTranscribe:   opts.VoiceTranscribe,
		heartbeatInterval: 30 * time.Second,
	}
}

func (b *Bootstrap) Status() BootstrapStatus {
	if !b.deps.OK {
		return BootstrapStatus{
			OK:        false,
			Code:      nonEmpty(b.deps.Code, "qq_missing_dependency"),
			Message:   b.redact(nonEmpty(b.deps.Message, "QQ Bot runtime dependency is unavailable")),
			Retryable: b.deps.Retryable,
		}
	}
	if b.cfg.AppID == "" || b.cfg.ClientSecret == "" {
		return BootstrapStatus{
			OK:        false,
			Code:      "qq_missing_credentials",
			Message:   "QQ Bot credentials are required: QQ_APP_ID and QQ_CLIENT_SECRET",
			Retryable: true,
		}
	}
	return BootstrapStatus{OK: true}
}

func (b *Bootstrap) Connect(ctx context.Context) BootstrapStatus {
	if status := b.Status(); !status.OK {
		return status
	}
	var release func()
	if b.locker != nil {
		var ok bool
		release, ok = b.locker.AcquireQQApp(ctx, b.cfg.AppID)
		if !ok {
			return BootstrapStatus{Code: "qq_lock_unavailable", Message: "QQ Bot app ID is already connected", Retryable: true}
		}
		b.releaseLock = release
	}
	fail := func(status BootstrapStatus) BootstrapStatus {
		if release != nil {
			release()
			b.releaseLock = nil
		}
		return status
	}
	if b.tokenClient == nil {
		return fail(BootstrapStatus{Code: "qq_token_failed", Message: "QQ Bot token client is not configured", Retryable: true})
	}
	token, err := b.tokenClient.FetchToken(ctx, b.cfg.AppID, b.cfg.ClientSecret)
	if err != nil || strings.TrimSpace(token.AccessToken) == "" {
		return fail(BootstrapStatus{Code: "qq_token_failed", Message: "QQ Bot token request failed", Retryable: true})
	}
	if b.gatewayClient == nil {
		return fail(BootstrapStatus{Code: "qq_gateway_unavailable", Message: "QQ Bot gateway client is not configured", Retryable: true})
	}
	gatewayURL, err := b.gatewayClient.FetchGatewayURL(ctx, token.AccessToken)
	if err != nil || strings.TrimSpace(gatewayURL) == "" {
		return fail(BootstrapStatus{Code: "qq_gateway_unavailable", Message: "QQ Bot gateway URL request failed", Retryable: true})
	}
	if b.ws == nil {
		return fail(BootstrapStatus{Code: "qq_connect_error", Message: "QQ Bot websocket client is not configured", Retryable: true})
	}
	if err := b.ws.Open(ctx, gatewayURL); err != nil {
		return fail(BootstrapStatus{Code: "qq_connect_error", Message: "QQ Bot websocket connection failed", Retryable: true})
	}
	b.mu.Lock()
	b.connected = true
	b.mu.Unlock()
	return BootstrapStatus{OK: true}
}

func (b *Bootstrap) State() BootstrapState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return BootstrapState{
		Connected:         b.connected,
		SessionID:         b.sessionID,
		LastSeq:           b.lastSeq,
		HeartbeatInterval: b.heartbeatInterval,
	}
}

func (b *Bootstrap) DispatchPayload(ctx context.Context, payload map[string]any) (gateway.InboundEvent, bool, error) {
	op := intValueFromAny(payload["op"])
	eventType, _ := payload["t"].(string)
	seq, hasSeq := optionalInt(payload["s"])
	data, _ := payload["d"].(map[string]any)

	b.mu.Lock()
	if hasSeq && seq > b.lastSeq {
		b.lastSeq = seq
	}
	switch op {
	case 10:
		intervalMS := intValueFromAny(data["heartbeat_interval"])
		if intervalMS <= 0 {
			intervalMS = 30000
		}
		b.heartbeatInterval = time.Duration(float64(intervalMS)*0.8) * time.Millisecond
		resume := b.sessionID != "" && b.lastSeq > 0
		lastSeq := b.lastSeq
		sessionID := b.sessionID
		b.mu.Unlock()
		if b.ws != nil {
			frame := map[string]any{"op": 2, "d": map[string]any{"token": b.cfg.AppID}}
			if resume {
				frame = map[string]any{"op": 6, "d": map[string]any{"session_id": sessionID, "seq": lastSeq}}
			}
			if err := b.ws.Send(ctx, frame); err != nil {
				return gateway.InboundEvent{}, false, err
			}
		}
		return gateway.InboundEvent{}, false, nil
	case 0:
		switch eventType {
		case "READY":
			if sid, _ := data["session_id"].(string); strings.TrimSpace(sid) != "" {
				b.sessionID = strings.TrimSpace(sid)
			}
			b.mu.Unlock()
			return gateway.InboundEvent{}, false, nil
		case "RESUMED":
			b.mu.Unlock()
			return gateway.InboundEvent{}, false, nil
		}
	default:
		b.mu.Unlock()
		return gateway.InboundEvent{}, false, nil
	}
	b.mu.Unlock()

	msg, ok := inboundFromDispatch(eventType, data)
	if !ok || b.bot == nil {
		return gateway.InboundEvent{}, false, nil
	}
	ev, ok := b.bot.toInboundEvent(msg)
	return ev, ok, nil
}

func (b *Bootstrap) SendHeartbeat(ctx context.Context) error {
	if b.ws == nil {
		return nil
	}
	state := b.State()
	return b.ws.Send(ctx, map[string]any{"op": 1, "d": state.LastSeq})
}

func (b *Bootstrap) BuildTextBody(content, replyTo string) QQTextBody {
	content = truncate(content, QQMaxMessageLength)
	body := QQTextBody{
		MsgSeq: stableMsgSeq(replyTo, content),
	}
	if b.cfg.MarkdownSupport {
		body.MsgType = QQMsgTypeMarkdown
		body.Markdown = map[string]string{"content": content}
		return body
	}
	body.MsgType = QQMsgTypeText
	body.Content = content
	if strings.TrimSpace(replyTo) != "" {
		body.MessageReference = map[string]string{"message_id": strings.TrimSpace(replyTo)}
	}
	return body
}

func (b *Bootstrap) Send(ctx context.Context, chatID, content string) QQSendResult {
	if strings.TrimSpace(content) == "" {
		return QQSendResult{Success: true}
	}
	if !b.State().Connected {
		wait := b.waitForReconnect
		if wait == nil {
			wait = defaultWaitForReconnect
		}
		if !wait(ctx, b.reconnectWait, b.reconnectPoll) {
			return QQSendResult{Code: "qq_send_retryable", Error: "Not connected", Retryable: true}
		}
		b.mu.Lock()
		b.connected = true
		b.mu.Unlock()
	}
	if b.bot == nil {
		return QQSendResult{Code: "qq_send_failed", Error: "QQ Bot seam is not configured", Retryable: true}
	}
	opts, chatType, err := b.bot.nextSendOptions(chatID)
	if err != nil {
		return QQSendResult{Code: "qq_send_failed", Error: err.Error(), Retryable: false}
	}
	if b.rest == nil {
		return QQSendResult{Code: "qq_send_failed", Error: "QQ Bot REST client is not configured", Retryable: true}
	}
	body := b.BuildTextBody(content, opts.ReplyToMessageID)
	msgID, err := b.rest.SendText(ctx, chatType, chatID, body)
	if err != nil {
		return QQSendResult{Code: "qq_send_retryable", Error: "QQ Bot send failed", Retryable: true}
	}
	return QQSendResult{Success: true, MessageID: msgID}
}

func (b *Bootstrap) TranscribeVoice(ctx context.Context, attachment VoiceAttachment) VoiceResult {
	if text := strings.TrimSpace(attachment.ASRReferText); text != "" {
		return VoiceResult{OK: true, Transcript: text}
	}
	downloadURL := strings.TrimSpace(attachment.URL)
	if wav := strings.TrimSpace(attachment.VoiceWAVURL); wav != "" {
		if strings.HasPrefix(wav, "//") {
			wav = "https:" + wav
		}
		downloadURL = wav
	}
	if !b.safeURL(downloadURL) {
		return VoiceResult{Code: "qq_ssrf_blocked", Message: "QQ Bot voice URL blocked by SSRF guard", Retryable: false}
	}
	if b.voiceTranscribe == nil {
		return VoiceResult{Code: "qq_voice_stt_unavailable", Message: "QQ Bot voice STT is not configured", Retryable: true}
	}
	attachment.URL = downloadURL
	return b.voiceTranscribe(ctx, attachment)
}

func QQCloseEvidence(err error) BootstrapStatus {
	var closeErr QQCloseError
	if !AsQQCloseError(err, &closeErr) {
		return BootstrapStatus{Code: "qq_ws_closed", Message: "QQ Bot websocket closed", Retryable: true}
	}
	retryable := true
	if closeErr.Code == 4914 || closeErr.Code == 4915 {
		retryable = false
	}
	return BootstrapStatus{
		Code:      "qq_ws_closed",
		Message:   fmt.Sprintf("QQ Bot websocket closed: code=%d reason=%s", closeErr.Code, closeErr.Reason),
		Retryable: retryable,
	}
}

func AsQQCloseError(err error, target *QQCloseError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(QQCloseError); ok {
		*target = e
		return true
	}
	if e, ok := err.(*QQCloseError); ok {
		*target = *e
		return true
	}
	return false
}

func RejectUnsafeRedirect(_, target string) error {
	u, err := url.Parse(strings.TrimSpace(target))
	if err != nil {
		return err
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("unsafe QQ redirect scheme %q", u.Scheme)
	}
	if !DefaultSafeURL(target) {
		return fmt.Errorf("unsafe QQ redirect target %q", target)
	}
	return nil
}

func DefaultSafeURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast())
	}
	return true
}

func (b *Bootstrap) redact(message string) string {
	secret := strings.TrimSpace(b.cfg.ClientSecret)
	if secret != "" {
		message = strings.ReplaceAll(message, secret, "[redacted]")
	}
	return message
}

func inboundFromDispatch(eventType string, data map[string]any) (InboundMessage, bool) {
	content := stringFromAny(data["content"])
	msgID := stringFromAny(data["id"])
	author, _ := data["author"].(map[string]any)
	userID := firstNonEmpty(stringFromAny(author["id"]), stringFromAny(data["author_id"]), stringFromAny(data["openid"]))
	userName := firstNonEmpty(stringFromAny(author["username"]), stringFromAny(author["name"]))
	switch eventType {
	case "GROUP_AT_MESSAGE_CREATE":
		return InboundMessage{
			ChatType:  ChatTypeGroup,
			ChatID:    firstNonEmpty(stringFromAny(data["group_openid"]), stringFromAny(data["group_id"])),
			UserID:    userID,
			UserName:  userName,
			MessageID: msgID,
			Text:      content,
			Mentioned: true,
		}, true
	case "C2C_MESSAGE_CREATE", "DIRECT_MESSAGE_CREATE":
		return InboundMessage{
			ChatType:  ChatTypeDirect,
			ChatID:    firstNonEmpty(stringFromAny(data["openid"]), userID),
			UserID:    userID,
			UserName:  userName,
			MessageID: msgID,
			Text:      content,
		}, true
	default:
		return InboundMessage{}, false
	}
}

func defaultWaitForReconnect(ctx context.Context, wait, poll time.Duration) bool {
	_ = poll
	deadline := time.NewTimer(wait)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-deadline.C:
			return false
		}
	}
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func stableMsgSeq(values ...string) int {
	var n uint32 = 2166136261
	for _, value := range values {
		for _, b := range []byte(value) {
			n ^= uint32(b)
			n *= 16777619
		}
	}
	return int(n % 65536)
}

func optionalInt(value any) (int, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func intValueFromAny(value any) int {
	i, _ := optionalInt(value)
	return i
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
