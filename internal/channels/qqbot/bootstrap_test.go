package qqbot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestQQBotBootstrapConfigAndDependencyStatus(t *testing.T) {
	t.Setenv("QQ_APP_ID", "env-app")
	t.Setenv("QQ_CLIENT_SECRET", "env-secret")

	cfg := ResolveBootstrapConfig(BootstrapOptions{
		AppID:           " explicit-app ",
		ClientSecret:    " explicit-secret ",
		MarkdownSupport: boolPtr(false),
		DMPolicy:        " allowlist ",
		AllowFrom:       "user-1, user-2",
		GroupPolicy:     "disabled",
		GroupAllowFrom:  []string{" group-1 ", "group-2"},
	})
	if cfg.AppID != "explicit-app" || cfg.ClientSecret != "explicit-secret" {
		t.Fatalf("explicit credentials did not win over env: %+v", cfg)
	}
	if cfg.MarkdownSupport {
		t.Fatalf("MarkdownSupport = true, want explicit false")
	}
	if cfg.DMPolicy != "allowlist" || strings.Join(cfg.AllowFrom, ",") != "user-1,user-2" {
		t.Fatalf("DM policy/list not parsed: %+v", cfg)
	}
	if cfg.GroupPolicy != "disabled" || strings.Join(cfg.GroupAllowFrom, ",") != "group-1,group-2" {
		t.Fatalf("group policy/list not parsed: %+v", cfg)
	}

	envCfg := ResolveBootstrapConfig(BootstrapOptions{})
	if envCfg.AppID != "env-app" || envCfg.ClientSecret != "env-secret" {
		t.Fatalf("env fallback not applied: %+v", envCfg)
	}
	if !envCfg.MarkdownSupport {
		t.Fatalf("MarkdownSupport default = false, want true")
	}

	deps := NewBootstrap(BootstrapOptions{
		AppID:        "app",
		ClientSecret: "super-secret-value",
		Dependencies: DependencyStatus{
			OK:        false,
			Code:      "qq_missing_dependency",
			Message:   "httpx missing for super-secret-value",
			Retryable: true,
		},
	})
	status := deps.Status()
	if status.OK || status.Code != "qq_missing_dependency" || !status.Retryable {
		t.Fatalf("dependency status = %+v", status)
	}
	if strings.Contains(status.Message, "super-secret-value") {
		t.Fatalf("dependency status leaked secret: %q", status.Message)
	}

	missing := NewBootstrap(BootstrapOptions{
		AppID:        "app-only",
		Dependencies: DependencyStatus{OK: true},
		Getenv:       func(string) string { return "" },
	})
	if status := missing.Status(); status.OK || status.Code != "qq_missing_credentials" {
		t.Fatalf("missing credential status = %+v", status)
	}

	locker := &fakeLocker{}
	failing := NewBootstrap(BootstrapOptions{
		AppID:        "app",
		ClientSecret: "secret",
		Dependencies: DependencyStatus{OK: true},
		Locker:       locker,
		TokenClient:  &fakeTokenClient{err: errors.New("token says secret=secret")},
		GatewayClient: &fakeGatewayClient{
			url: "wss://gateway.qq.example",
		},
		Websocket: &fakeWebsocket{},
	})
	result := failing.Connect(context.Background())
	if result.OK || result.Code != "qq_token_failed" {
		t.Fatalf("Connect() = %+v, want qq_token_failed", result)
	}
	if locker.acquired != 1 || locker.released != 1 {
		t.Fatalf("lock lifecycle = acquired %d released %d, want 1/1", locker.acquired, locker.released)
	}
	if strings.Contains(result.Message, "secret") {
		t.Fatalf("connect failure leaked secret: %q", result.Message)
	}
}

func TestQQBotConnectTokenGatewayAndRedirectGuard(t *testing.T) {
	token := fakeTokenClient{token: Token{AccessToken: "tok-1", ExpiresIn: time.Hour}}
	gatewayClient := fakeGatewayClient{url: "wss://gateway.qq.example/ws"}
	ws := &fakeWebsocket{}
	boot := NewBootstrap(BootstrapOptions{
		AppID:         "app",
		ClientSecret:  "secret",
		Dependencies:  DependencyStatus{OK: true},
		Locker:        &fakeLocker{},
		TokenClient:   &token,
		GatewayClient: &gatewayClient,
		Websocket:     ws,
	})

	result := boot.Connect(context.Background())
	if !result.OK {
		t.Fatalf("Connect() = %+v", result)
	}
	if token.calls != 1 || token.seenAppID != "app" || token.seenSecret != "secret" {
		t.Fatalf("token client calls = %+v", token)
	}
	if gatewayClient.calls != 1 || gatewayClient.seenToken != "tok-1" {
		t.Fatalf("gateway client calls = %+v", gatewayClient)
	}
	if ws.openedURL != "wss://gateway.qq.example/ws" {
		t.Fatalf("openedURL = %q", ws.openedURL)
	}

	if err := RejectUnsafeRedirect("https://bots.qq.com/app/getAppAccessToken", "http://127.0.0.1/token"); err == nil {
		t.Fatal("RejectUnsafeRedirect allowed loopback redirect")
	}
	if err := RejectUnsafeRedirect("https://bots.qq.com/app/getAppAccessToken", "https://api.sgroup.qq.com/gateway"); err != nil {
		t.Fatalf("RejectUnsafeRedirect blocked QQ HTTPS redirect: %v", err)
	}
}

func TestQQBotWebsocketDispatchAndHeartbeat(t *testing.T) {
	ws := &fakeWebsocket{}
	bot := New(Config{GroupPolicy: "open"}, newMockClient(), nil)
	boot := NewBootstrap(BootstrapOptions{
		AppID:        "app",
		ClientSecret: "secret",
		Dependencies: DependencyStatus{OK: true},
		Websocket:    ws,
		Bot:          bot,
	})

	if _, ok, err := boot.DispatchPayload(context.Background(), map[string]any{
		"op": float64(10),
		"d":  map[string]any{"heartbeat_interval": float64(50000)},
	}); err != nil || ok {
		t.Fatalf("hello dispatch = ok %v err %v", ok, err)
	}
	if got := boot.State().HeartbeatInterval; got != 40*time.Second {
		t.Fatalf("HeartbeatInterval = %v, want 40s", got)
	}
	if len(ws.sent) != 1 || intValue(ws.sent[0]["op"]) != 2 {
		t.Fatalf("identify frame = %#v", ws.sent)
	}

	if _, ok, err := boot.DispatchPayload(context.Background(), map[string]any{
		"op": float64(0),
		"t":  "READY",
		"s":  float64(100),
		"d":  map[string]any{"session_id": "sess-1"},
	}); err != nil || ok {
		t.Fatalf("READY dispatch = ok %v err %v", ok, err)
	}
	if state := boot.State(); state.SessionID != "sess-1" || state.LastSeq != 100 {
		t.Fatalf("state after READY = %+v", state)
	}

	if _, ok, err := boot.DispatchPayload(context.Background(), map[string]any{
		"op": float64(0),
		"t":  "RESUMED",
		"s":  float64(105),
		"d":  map[string]any{},
	}); err != nil || ok {
		t.Fatalf("RESUMED dispatch = ok %v err %v", ok, err)
	}
	if state := boot.State(); state.SessionID != "sess-1" || state.LastSeq != 105 {
		t.Fatalf("state after RESUMED = %+v", state)
	}

	ev, ok, err := boot.DispatchPayload(context.Background(), map[string]any{
		"op": float64(0),
		"t":  "GROUP_AT_MESSAGE_CREATE",
		"s":  float64(106),
		"d": map[string]any{
			"id":           "msg-1",
			"group_openid": "group-1",
			"content":      "@Gormes /help",
			"author":       map[string]any{"id": "user-1", "username": "Ada"},
		},
	})
	if err != nil || !ok {
		t.Fatalf("message dispatch = ok %v err %v", ok, err)
	}
	if ev.Platform != "qqbot" || ev.ChatID != "group-1" || ev.UserID != "user-1" || ev.Kind != gateway.EventStart {
		t.Fatalf("unexpected event: %+v", ev)
	}

	if err := boot.SendHeartbeat(context.Background()); err != nil {
		t.Fatalf("SendHeartbeat() error = %v", err)
	}
	last := ws.sent[len(ws.sent)-1]
	if intValue(last["op"]) != 1 || intValue(last["d"]) != 106 {
		t.Fatalf("heartbeat frame = %#v, want op=1 d=106", last)
	}

	evidence := QQCloseEvidence(NewQQCloseError(4008, "rate limit"))
	if evidence.Code != "qq_ws_closed" || !strings.Contains(evidence.Message, "4008") || !strings.Contains(evidence.Message, "rate limit") || !evidence.Retryable {
		t.Fatalf("close evidence = %+v", evidence)
	}
}

func TestQQBotTextBodyMarkdownPlainTruncateAndReply(t *testing.T) {
	markdown := NewBootstrap(BootstrapOptions{
		AppID:           "app",
		ClientSecret:    "secret",
		MarkdownSupport: boolPtr(true),
	})
	body := markdown.BuildTextBody("**bold** text", "msg-1")
	if body.MsgType != QQMsgTypeMarkdown || body.Markdown == nil || body.Markdown["content"] != "**bold** text" {
		t.Fatalf("markdown body = %+v", body)
	}
	if body.MessageReference != nil {
		t.Fatalf("markdown body should not include message_reference: %+v", body)
	}

	plain := NewBootstrap(BootstrapOptions{
		AppID:           "app",
		ClientSecret:    "secret",
		MarkdownSupport: boolPtr(false),
	})
	longText := strings.Repeat("x", QQMaxMessageLength+100)
	body = plain.BuildTextBody(longText, "msg-2")
	if body.MsgType != QQMsgTypeText || len(body.Content) != QQMaxMessageLength {
		t.Fatalf("plain body type/length = %+v len=%d", body, len(body.Content))
	}
	if body.MessageReference == nil || body.MessageReference["message_id"] != "msg-2" {
		t.Fatalf("plain reply reference = %+v", body.MessageReference)
	}
}

func TestQQBotSendReconnectRetryable(t *testing.T) {
	bot := New(Config{GroupPolicy: "open"}, newMockClient(), nil)
	if _, ok := bot.toInboundEvent(InboundMessage{
		ChatType:  ChatTypeGroup,
		ChatID:    "group-1",
		UserID:    "user-1",
		MessageID: "msg-1",
		Text:      "@Gormes hello",
		Mentioned: true,
	}); !ok {
		t.Fatal("failed to seed bot reply metadata")
	}

	rest := &fakeRESTClient{}
	boot := NewBootstrap(BootstrapOptions{
		AppID:            "app",
		ClientSecret:     "secret",
		Bot:              bot,
		RESTClient:       rest,
		MarkdownSupport:  boolPtr(false),
		ReconnectWait:    200 * time.Millisecond,
		ReconnectPoll:    10 * time.Millisecond,
		WaitForReconnect: func(context.Context, time.Duration, time.Duration) bool { return false },
	})
	result := boot.Send(context.Background(), "group-1", "reply")
	if result.Success || !result.Retryable || result.Code != "qq_send_retryable" {
		t.Fatalf("disconnected Send() = %+v", result)
	}
	if len(rest.calls) != 0 {
		t.Fatalf("REST called while disconnected: %+v", rest.calls)
	}

	boot = NewBootstrap(BootstrapOptions{
		AppID:            "app",
		ClientSecret:     "secret",
		Bot:              bot,
		RESTClient:       rest,
		MarkdownSupport:  boolPtr(false),
		WaitForReconnect: func(context.Context, time.Duration, time.Duration) bool { return true },
	})
	result = boot.Send(context.Background(), "group-1", "reply")
	if !result.Success || result.MessageID != "sent-1" {
		t.Fatalf("connected Send() = %+v", result)
	}
	if len(rest.calls) != 1 || rest.calls[0].ChatType != ChatTypeGroup || rest.calls[0].Body.MessageReference["message_id"] != "msg-1" {
		t.Fatalf("REST calls = %+v", rest.calls)
	}
}

func TestQQBotVoiceAttachmentSSRFGuard(t *testing.T) {
	downloader := &fakeVoiceTranscriber{}
	boot := NewBootstrap(BootstrapOptions{
		AppID:           "app",
		ClientSecret:    "secret",
		SafeURL:         func(string) bool { return false },
		VoiceTranscribe: downloader.TranscribeVoice,
	})
	result := boot.TranscribeVoice(context.Background(), VoiceAttachment{
		URL:         "http://127.0.0.1/voice.silk",
		ContentType: "audio/silk",
		Filename:    "voice.silk",
	})
	if result.OK || result.Code != "qq_ssrf_blocked" {
		t.Fatalf("unsafe voice result = %+v", result)
	}
	if downloader.calls != 0 {
		t.Fatalf("unsafe voice URL reached transcriber")
	}

	direct := boot.TranscribeVoice(context.Background(), VoiceAttachment{
		URL:          "http://127.0.0.1/voice.silk",
		ASRReferText: "built in transcript",
	})
	if !direct.OK || direct.Transcript != "built in transcript" {
		t.Fatalf("ASR refer result = %+v", direct)
	}

	boot = NewBootstrap(BootstrapOptions{
		AppID:           "app",
		ClientSecret:    "secret",
		SafeURL:         func(url string) bool { return strings.HasPrefix(url, "https://cdn.qq.example/") },
		VoiceTranscribe: downloader.TranscribeVoice,
	})
	safe := boot.TranscribeVoice(context.Background(), VoiceAttachment{
		URL:         "https://cdn.qq.example/voice.silk",
		ContentType: "audio/silk",
		Filename:    "voice.silk",
	})
	if !safe.OK || safe.Transcript != "fake transcript" || downloader.calls != 1 {
		t.Fatalf("safe voice result = %+v calls=%d", safe, downloader.calls)
	}
}

func boolPtr(v bool) *bool { return &v }

type fakeLocker struct {
	acquired int
	released int
}

func (f *fakeLocker) AcquireQQApp(_ context.Context, appID string) (func(), bool) {
	if strings.TrimSpace(appID) == "" {
		return nil, false
	}
	f.acquired++
	return func() { f.released++ }, true
}

type fakeTokenClient struct {
	token      Token
	err        error
	calls      int
	seenAppID  string
	seenSecret string
}

func (f *fakeTokenClient) FetchToken(_ context.Context, appID, clientSecret string) (Token, error) {
	f.calls++
	f.seenAppID = appID
	f.seenSecret = clientSecret
	if f.err != nil {
		return Token{}, f.err
	}
	return f.token, nil
}

type fakeGatewayClient struct {
	url       string
	err       error
	calls     int
	seenToken string
}

func (f *fakeGatewayClient) FetchGatewayURL(_ context.Context, token string) (string, error) {
	f.calls++
	f.seenToken = token
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

type fakeWebsocket struct {
	openedURL string
	sent      []map[string]any
}

func (f *fakeWebsocket) Open(_ context.Context, url string) error {
	f.openedURL = url
	return nil
}

func (f *fakeWebsocket) Send(_ context.Context, frame map[string]any) error {
	f.sent = append(f.sent, frame)
	return nil
}

type fakeRESTClient struct {
	calls []qqRESTCall
}

func (f *fakeRESTClient) SendText(_ context.Context, chatType, chatID string, body QQTextBody) (string, error) {
	f.calls = append(f.calls, qqRESTCall{ChatType: chatType, ChatID: chatID, Body: body})
	return "sent-1", nil
}

type qqRESTCall struct {
	ChatType string
	ChatID   string
	Body     QQTextBody
}

type fakeVoiceTranscriber struct {
	calls int
}

func (f *fakeVoiceTranscriber) TranscribeVoice(_ context.Context, attachment VoiceAttachment) VoiceResult {
	f.calls++
	if attachment.URL == "" {
		return VoiceResult{Code: "qq_voice_stt_unavailable", Message: "missing voice url"}
	}
	return VoiceResult{OK: true, Transcript: "fake transcript"}
}

func intValue(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}
