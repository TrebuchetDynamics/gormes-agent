package feishu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestFeishuBootstrapConnectionModeSelectsWebhookOrWebsocket(t *testing.T) {
	env := map[string]string{
		"FEISHU_APP_ID":          "cli_app",
		"FEISHU_APP_SECRET":      "plain-existing-secret",
		"FEISHU_CONNECTION_MODE": "webhook",
		"FEISHU_DOMAIN":          "lark",
		"FEISHU_WEBHOOK_HOST":    "127.0.0.2",
		"FEISHU_WEBHOOK_PORT":    "9001",
		"FEISHU_WEBHOOK_PATH":    "/hook",
	}
	cfg := ResolveBootstrapConfig(nil, func(k string) string { return env[k] })
	if cfg.AppID != "cli_app" || cfg.AppSecret != "plain-existing-secret" {
		t.Fatalf("ResolveBootstrapConfig credentials = %#v", cfg)
	}
	if cfg.ConnectionMode != ModeWebhook || cfg.Domain != "lark" {
		t.Fatalf("ResolveBootstrapConfig mode/domain = %#v", cfg)
	}
	if cfg.WebhookHost != "127.0.0.2" || cfg.WebhookPort != 9001 || cfg.WebhookPath != "/hook" {
		t.Fatalf("ResolveBootstrapConfig webhook = %#v", cfg)
	}

	extra := map[string]string{"connection_mode": "websocket", "app_id": "extra_app", "app_secret": "extra_secret"}
	cfg = ResolveBootstrapConfig(extra, func(k string) string { return env[k] })
	status := SelectBootstrapLifecycle(cfg)
	if !status.Ready || status.Mode != ModeWebsocket || status.Evidence != "" {
		t.Fatalf("SelectBootstrapLifecycle websocket = %+v", status)
	}

	cfg.ConnectionMode = "smtp"
	status = SelectBootstrapLifecycle(cfg)
	if status.Ready || status.Evidence != FeishuEvidenceConfigMissing {
		t.Fatalf("SelectBootstrapLifecycle invalid mode = %+v", status)
	}

	cfg.ConnectionMode = ModeWebhook
	cfg.AppSecret = ""
	status = SelectBootstrapLifecycle(cfg)
	if status.Ready || status.Evidence != FeishuEvidenceConfigMissing {
		t.Fatalf("SelectBootstrapLifecycle missing secret = %+v", status)
	}
}

func TestFeishuBootstrapWebhookURLVerificationAndSignature(t *testing.T) {
	challenge := VerifyWebhookRequest([]byte(`{"type":"url_verification","challenge":"abc123"}`), nil, "verify-token", "encrypt-key")
	if challenge.Status != 200 || challenge.Challenge != "abc123" || challenge.Evidence != "" {
		t.Fatalf("url verification = %+v", challenge)
	}

	body := []byte(`{"header":{"token":"verify-token","event_type":"im.message.receive_v1"},"event":{}}`)
	headers := map[string]string{
		"x-lark-request-timestamp": "1700000000",
		"x-lark-request-nonce":     "nonce-1",
	}
	headers["x-lark-signature"] = feishuSignature(headers["x-lark-request-timestamp"], headers["x-lark-request-nonce"], "encrypt-key", body)
	ok := VerifyWebhookRequest(body, headers, "verify-token", "encrypt-key")
	if ok.Status != 200 || ok.Evidence != "" || ok.EventType != "im.message.receive_v1" {
		t.Fatalf("valid webhook = %+v", ok)
	}

	badToken := VerifyWebhookRequest([]byte(`{"header":{"token":"plain-existing-token"}}`), headers, "verify-token", "encrypt-key")
	if badToken.Status != 401 || badToken.Evidence != FeishuEvidenceSignatureInvalid {
		t.Fatalf("bad token = %+v", badToken)
	}
	if strings.Contains(badToken.Error, "plain-existing-token") || strings.Contains(badToken.Error, "verify-token") {
		t.Fatalf("bad token error leaked credential: %+v", badToken)
	}

	headers["x-lark-signature"] = "bad-signature"
	badSig := VerifyWebhookRequest(body, headers, "verify-token", "encrypt-key")
	if badSig.Status != 401 || badSig.Evidence != FeishuEvidenceSignatureInvalid {
		t.Fatalf("bad signature = %+v", badSig)
	}
}

func TestFeishuBootstrapEventHandlerRegistration(t *testing.T) {
	builder := &fakeEventHandlerBuilder{}
	RegisterDefaultEventHandlers(builder, FeishuEventHandlers{})
	want := []string{
		"message_read", "message_receive", "reaction_created", "reaction_deleted", "card_action",
		"bot_added", "bot_deleted", "p2p_entered", "message_recalled", "customized:drive.notice.comment_add_v1",
	}
	if !reflect.DeepEqual(builder.calls, want) {
		t.Fatalf("handler registrations = %#v, want %#v", builder.calls, want)
	}
	if !builder.built {
		t.Fatal("RegisterDefaultEventHandlers did not build handler")
	}
}

func TestFeishuBootstrapLoopNotReadyQueuesAndDrainsOnce(t *testing.T) {
	buffer := NewLoopBuffer()
	var drained []string
	buffer.Submit(InboundMessage{MessageID: "m1"}, func(msg InboundMessage) { drained = append(drained, msg.MessageID) })
	buffer.Submit(InboundMessage{MessageID: "m2"}, func(msg InboundMessage) { drained = append(drained, msg.MessageID) })
	if got, want := buffer.Pending(), 2; got != want {
		t.Fatalf("Pending before ready = %d, want %d", got, want)
	}
	if buffer.LastEvidence() != FeishuEvidenceLoopNotReady {
		t.Fatalf("LastEvidence before ready = %q", buffer.LastEvidence())
	}
	buffer.MarkReady(func(msg InboundMessage) { drained = append(drained, msg.MessageID) })
	buffer.MarkReady(func(msg InboundMessage) { drained = append(drained, "again:"+msg.MessageID) })
	buffer.Submit(InboundMessage{MessageID: "m3"}, func(msg InboundMessage) { drained = append(drained, msg.MessageID) })
	want := []string{"m1", "m2", "m3"}
	if !reflect.DeepEqual(drained, want) {
		t.Fatalf("drained = %#v, want %#v", drained, want)
	}
	if got := buffer.Pending(); got != 0 {
		t.Fatalf("Pending after ready = %d, want 0", got)
	}
}

func TestFeishuBootstrapRichTextSendFailureEvidence(t *testing.T) {
	mc := newMockClient()
	mc.richErr = errors.New("upstream rejected plain-existing-secret token body")
	result := SendRichTextWithEvidence(context.Background(), mc, "chat-1", "**hello**", SendOptions{ReplyToMessageID: "msg-1"})
	if result.Evidence != FeishuEvidenceSendFailed || result.MessageID != "" {
		t.Fatalf("SendRichTextWithEvidence = %+v", result)
	}
	if strings.Contains(result.Error, "plain-existing-secret") || strings.Contains(result.Error, "token body") {
		t.Fatalf("send failure leaked upstream secret/body: %+v", result)
	}
	if len(mc.richSent) != 1 || mc.richSent[0].Options.ReplyToMessageID != "msg-1" {
		t.Fatalf("rich send call = %#v", mc.richSent)
	}
}

func feishuSignature(timestamp, nonce, encryptKey string, body []byte) string {
	sum := sha256.Sum256([]byte(timestamp + nonce + encryptKey + string(body)))
	return hex.EncodeToString(sum[:])
}

type fakeEventHandlerBuilder struct {
	calls []string
	built bool
}

func (f *fakeEventHandlerBuilder) RegisterMessageRead(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "message_read")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterMessageReceive(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "message_receive")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterReactionCreated(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "reaction_created")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterReactionDeleted(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "reaction_deleted")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterCardAction(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "card_action")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterBotAdded(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "bot_added")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterBotDeleted(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "bot_deleted")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterP2PChatEntered(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "p2p_entered")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterMessageRecalled(any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "message_recalled")
	return f
}
func (f *fakeEventHandlerBuilder) RegisterCustomized(event string, _ any) FeishuEventHandlerBuilder {
	f.calls = append(f.calls, "customized:"+event)
	return f
}
func (f *fakeEventHandlerBuilder) Build() any {
	f.built = true
	return struct{}{}
}
