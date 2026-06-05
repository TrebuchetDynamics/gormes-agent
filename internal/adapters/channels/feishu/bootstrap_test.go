package feishu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestFeishuUpdatePromptCardBuildsYesNoActions(t *testing.T) {
	card := BuildUpdatePromptCard("Restore stashed changes after update?", "y", 17)
	payload, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal card: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal card: %v", err)
	}

	header := got["header"].(map[string]any)
	if header["template"] != "orange" {
		t.Fatalf("header template = %q, want orange", header["template"])
	}
	elements := got["elements"].([]any)
	content := elements[0].(map[string]any)["content"].(string)
	if !strings.Contains(content, "Restore stashed changes after update?") || !strings.Contains(content, "Default: `y`") {
		t.Fatalf("card content = %q", content)
	}
	actions := elements[1].(map[string]any)["actions"].([]any)
	if len(actions) != 2 {
		t.Fatalf("actions = %d, want 2", len(actions))
	}
	for i, want := range []string{"y", "n"} {
		value := actions[i].(map[string]any)["value"].(map[string]any)
		if value["hermes_update_prompt_action"] != want {
			t.Fatalf("action %d answer = %q, want %q", i, value["hermes_update_prompt_action"], want)
		}
		if value["update_prompt_id"].(float64) != 17 {
			t.Fatalf("action %d prompt id = %#v, want 17", i, value["update_prompt_id"])
		}
	}
}

func TestFeishuUpdatePromptResolveWritesOnce(t *testing.T) {
	store := NewUpdatePromptStore()
	store.Store(3, UpdatePromptRecord{
		SessionKey: "agent:main:feishu:group:oc_12345",
		MessageID:  "msg-up-1",
		ChatID:     "oc_12345",
	})
	var writes []string
	resolved, ok, err := store.Resolve(UpdatePromptAction{
		PromptID:   3,
		Answer:     "y",
		ActorName:  "Alice",
		Authorized: true,
	}, func(answer string) error {
		writes = append(writes, answer)
		return nil
	})
	if err != nil || !ok {
		t.Fatalf("Resolve = %+v, %v, %v; want ok nil", resolved, ok, err)
	}
	if !reflect.DeepEqual(writes, []string{"y"}) {
		t.Fatalf("writes = %#v, want [y]", writes)
	}
	if resolved.Record.SessionKey != "agent:main:feishu:group:oc_12345" || resolved.Record.MessageID != "msg-up-1" {
		t.Fatalf("resolved record = %+v", resolved.Record)
	}
	cardPayload, err := json.Marshal(resolved.Card)
	if err != nil {
		t.Fatalf("marshal resolved card: %v", err)
	}
	var card map[string]any
	if err := json.Unmarshal(cardPayload, &card); err != nil {
		t.Fatalf("unmarshal resolved card: %v", err)
	}
	if got := card["header"].(map[string]any)["template"]; got != "green" {
		t.Fatalf("resolved template = %q, want green", got)
	}
	if content := card["elements"].([]any)[0].(map[string]any)["content"].(string); !strings.Contains(content, "Alice") {
		t.Fatalf("resolved card content = %q, want actor", content)
	}

	_, ok, err = store.Resolve(UpdatePromptAction{PromptID: 3, Answer: "n", ActorName: "Bob", Authorized: true}, func(answer string) error {
		writes = append(writes, answer)
		return nil
	})
	if err != nil || ok {
		t.Fatalf("second Resolve ok=%v err=%v, want already resolved", ok, err)
	}
	if !reflect.DeepEqual(writes, []string{"y"}) {
		t.Fatalf("writes after second resolve = %#v, want [y]", writes)
	}

	redCard := BuildResolvedUpdatePromptCard("n", "Bob")
	redPayload, err := json.Marshal(redCard)
	if err != nil {
		t.Fatalf("marshal red card: %v", err)
	}
	var red map[string]any
	if err := json.Unmarshal(redPayload, &red); err != nil {
		t.Fatalf("unmarshal red card: %v", err)
	}
	if got := red["header"].(map[string]any)["template"]; got != "red" {
		t.Fatalf("no card template = %q, want red", got)
	}
}

func TestFeishuUpdatePromptRejectsInvalidActions(t *testing.T) {
	store := NewUpdatePromptStore()
	store.Store(5, UpdatePromptRecord{SessionKey: "s", MessageID: "m", ChatID: "c"})
	var writes []string
	for _, action := range []UpdatePromptAction{
		{PromptID: 0, Answer: "y", ActorName: "Alice", Authorized: true},
		{PromptID: 99, Answer: "y", ActorName: "Alice", Authorized: true},
		{PromptID: 5, Answer: "maybe", ActorName: "Alice", Authorized: true},
		{PromptID: 5, Answer: "y", ActorName: "Mallory", Authorized: false},
	} {
		resolved, ok, err := store.Resolve(action, func(answer string) error {
			writes = append(writes, answer)
			return nil
		})
		if err != nil || ok || resolved.Card != nil {
			t.Fatalf("Resolve(%+v) = %+v, %v, %v; want rejected", action, resolved, ok, err)
		}
	}
	if len(writes) != 0 {
		t.Fatalf("writes = %#v, want none", writes)
	}
	if _, ok, err := store.Resolve(UpdatePromptAction{PromptID: 5, Answer: "n", ActorID: "ou_1", Authorized: true}, func(answer string) error {
		writes = append(writes, answer)
		return nil
	}); err != nil || !ok {
		t.Fatalf("valid action after rejected actions ok=%v err=%v, want resolved", ok, err)
	}
	if !reflect.DeepEqual(writes, []string{"n"}) {
		t.Fatalf("writes after valid action = %#v, want [n]", writes)
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
