package feishu

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const (
	FeishuEvidenceConfigMissing    = "feishu_config_missing"
	FeishuEvidenceSignatureInvalid = "feishu_signature_invalid"
	FeishuEvidenceLoopNotReady     = "feishu_loop_not_ready"
	FeishuEvidenceSendFailed       = "feishu_send_failed"
)

const (
	defaultDomain      = "feishu"
	defaultWebhookHost = "127.0.0.1"
	defaultWebhookPort = 8765
	defaultWebhookPath = "/feishu/webhook"
)

// BootstrapConfig is a Go-native, SDK-free subset of Hermes' Feishu adapter
// settings. It is intentionally enough to select lifecycle, verify webhook
// traffic, and construct fake adapter tests without opening sockets.
type BootstrapConfig struct {
	AppID             string
	AppSecret         string
	Domain            string
	ConnectionMode    string
	EncryptKey        string
	VerificationToken string
	WebhookHost       string
	WebhookPort       int
	WebhookPath       string
}

// BootstrapStatus reports whether the fakeable lifecycle can start.
type BootstrapStatus struct {
	Ready    bool
	Mode     string
	Evidence string
	Error    string
}

// ResolveBootstrapConfig mirrors the Hermes env/extra precedence for the Feishu
// bootstrap fields. extra values win, then env values, then safe defaults.
func ResolveBootstrapConfig(extra map[string]string, getenv func(string) string) BootstrapConfig {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	get := func(key, envKey, def string) string {
		if extra != nil {
			if v := strings.TrimSpace(extra[key]); v != "" {
				return v
			}
		}
		if v := strings.TrimSpace(getenv(envKey)); v != "" {
			return v
		}
		return def
	}
	port := defaultWebhookPort
	if raw := get("webhook_port", "FEISHU_WEBHOOK_PORT", ""); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			port = parsed
		}
	}
	path := get("webhook_path", "FEISHU_WEBHOOK_PATH", defaultWebhookPath)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return BootstrapConfig{
		AppID:             get("app_id", "FEISHU_APP_ID", ""),
		AppSecret:         get("app_secret", "FEISHU_APP_SECRET", ""),
		Domain:            strings.ToLower(get("domain", "FEISHU_DOMAIN", defaultDomain)),
		ConnectionMode:    strings.ToLower(get("connection_mode", "FEISHU_CONNECTION_MODE", ModeWebsocket)),
		EncryptKey:        get("encrypt_key", "FEISHU_ENCRYPT_KEY", ""),
		VerificationToken: get("verification_token", "FEISHU_VERIFICATION_TOKEN", ""),
		WebhookHost:       get("webhook_host", "FEISHU_WEBHOOK_HOST", defaultWebhookHost),
		WebhookPort:       port,
		WebhookPath:       path,
	}
}

func SelectBootstrapLifecycle(cfg BootstrapConfig) BootstrapStatus {
	mode := strings.ToLower(strings.TrimSpace(cfg.ConnectionMode))
	if mode == "" {
		mode = ModeWebsocket
	}
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return BootstrapStatus{Mode: mode, Evidence: FeishuEvidenceConfigMissing, Error: "Feishu app credentials are not configured"}
	}
	if mode != ModeWebsocket && mode != ModeWebhook {
		return BootstrapStatus{Mode: mode, Evidence: FeishuEvidenceConfigMissing, Error: "Feishu connection_mode must be websocket or webhook"}
	}
	return BootstrapStatus{Ready: true, Mode: mode}
}

// WebhookVerificationResult is a sanitized result for Feishu webhook security.
type WebhookVerificationResult struct {
	Status    int
	Challenge string
	EventType string
	Evidence  string
	Error     string
}

func VerifyWebhookRequest(body []byte, headers map[string]string, verificationToken, encryptKey string) WebhookVerificationResult {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return WebhookVerificationResult{Status: 400, Evidence: FeishuEvidenceSignatureInvalid, Error: "invalid webhook json"}
	}
	if payloadString(payload, "type") == "url_verification" {
		return WebhookVerificationResult{Status: 200, Challenge: payloadString(payload, "challenge")}
	}
	if verificationToken != "" {
		token := payloadString(payload, "token")
		if header, ok := payload["header"].(map[string]any); ok {
			if hv := payloadString(header, "token"); hv != "" {
				token = hv
			}
		}
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(verificationToken)) != 1 {
			return WebhookVerificationResult{Status: 401, Evidence: FeishuEvidenceSignatureInvalid, Error: "invalid verification token"}
		}
	}
	if encryptKey != "" && !WebhookSignatureValid(headers, body, encryptKey) {
		return WebhookVerificationResult{Status: 401, Evidence: FeishuEvidenceSignatureInvalid, Error: "invalid signature"}
	}
	eventType := ""
	if header, ok := payload["header"].(map[string]any); ok {
		eventType = payloadString(header, "event_type")
	}
	return WebhookVerificationResult{Status: 200, EventType: eventType}
}

func WebhookSignatureValid(headers map[string]string, body []byte, encryptKey string) bool {
	timestamp := headerValue(headers, "x-lark-request-timestamp")
	nonce := headerValue(headers, "x-lark-request-nonce")
	sig := headerValue(headers, "x-lark-signature")
	if timestamp == "" || nonce == "" || sig == "" {
		return false
	}
	sum := sha256.Sum256([]byte(timestamp + nonce + encryptKey + string(body)))
	computed := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(sig)) == 1
}

func headerValue(headers map[string]string, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func payloadString(payload map[string]any, key string) string {
	v, ok := payload[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

// FeishuEventHandlerBuilder is a small adapter over Hermes' EventDispatcher
// registration chain. Tests can fake it without importing the Feishu SDK.
type FeishuEventHandlerBuilder interface {
	RegisterMessageRead(any) FeishuEventHandlerBuilder
	RegisterMessageReceive(any) FeishuEventHandlerBuilder
	RegisterReactionCreated(any) FeishuEventHandlerBuilder
	RegisterReactionDeleted(any) FeishuEventHandlerBuilder
	RegisterCardAction(any) FeishuEventHandlerBuilder
	RegisterBotAdded(any) FeishuEventHandlerBuilder
	RegisterBotDeleted(any) FeishuEventHandlerBuilder
	RegisterP2PChatEntered(any) FeishuEventHandlerBuilder
	RegisterMessageRecalled(any) FeishuEventHandlerBuilder
	RegisterCustomized(string, any) FeishuEventHandlerBuilder
	Build() any
}

type FeishuEventHandlers struct {
	MessageRead     any
	MessageReceive  any
	ReactionCreated any
	ReactionDeleted any
	CardAction      any
	BotAdded        any
	BotDeleted      any
	P2PChatEntered  any
	MessageRecalled any
	DriveComment    any
}

func RegisterDefaultEventHandlers(builder FeishuEventHandlerBuilder, handlers FeishuEventHandlers) any {
	if builder == nil {
		return nil
	}
	return builder.
		RegisterMessageRead(handlers.MessageRead).
		RegisterMessageReceive(handlers.MessageReceive).
		RegisterReactionCreated(handlers.ReactionCreated).
		RegisterReactionDeleted(handlers.ReactionDeleted).
		RegisterCardAction(handlers.CardAction).
		RegisterBotAdded(handlers.BotAdded).
		RegisterBotDeleted(handlers.BotDeleted).
		RegisterP2PChatEntered(handlers.P2PChatEntered).
		RegisterMessageRecalled(handlers.MessageRecalled).
		RegisterCustomized("drive.notice.comment_add_v1", handlers.DriveComment).
		Build()
}

type LoopBuffer struct {
	mu           sync.Mutex
	ready        bool
	pending      []InboundMessage
	lastEvidence string
}

func NewLoopBuffer() *LoopBuffer { return &LoopBuffer{} }

func (b *LoopBuffer) Submit(msg InboundMessage, dispatch func(InboundMessage)) {
	b.mu.Lock()
	if !b.ready {
		b.pending = append(b.pending, msg)
		b.lastEvidence = FeishuEvidenceLoopNotReady
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()
	if dispatch != nil {
		dispatch(msg)
	}
}

func (b *LoopBuffer) MarkReady(dispatch func(InboundMessage)) {
	b.mu.Lock()
	if b.ready {
		b.mu.Unlock()
		return
	}
	b.ready = true
	pending := append([]InboundMessage(nil), b.pending...)
	b.pending = nil
	b.lastEvidence = ""
	b.mu.Unlock()
	if dispatch != nil {
		for _, msg := range pending {
			dispatch(msg)
		}
	}
}

func (b *LoopBuffer) Pending() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

func (b *LoopBuffer) LastEvidence() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastEvidence
}

type DeliveryResult struct {
	MessageID string
	Evidence  string
	Error     string
}

type UpdatePromptRecord struct {
	SessionKey string
	MessageID  string
	ChatID     string
}

type UpdatePromptAction struct {
	PromptID   int
	Answer     string
	ActorID    string
	ActorName  string
	Authorized bool
}

type UpdatePromptResolution struct {
	Record UpdatePromptRecord
	Card   map[string]any
}

type UpdatePromptStore struct {
	mu      sync.Mutex
	records map[int]UpdatePromptRecord
}

func NewUpdatePromptStore() *UpdatePromptStore {
	return &UpdatePromptStore{records: map[int]UpdatePromptRecord{}}
}

func (s *UpdatePromptStore) Store(promptID int, record UpdatePromptRecord) {
	if s == nil || promptID <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records == nil {
		s.records = map[int]UpdatePromptRecord{}
	}
	s.records[promptID] = record
}

func (s *UpdatePromptStore) Resolve(action UpdatePromptAction, writeAnswer func(string) error) (UpdatePromptResolution, bool, error) {
	answer, ok := normalizeUpdatePromptAnswer(action.Answer)
	if s == nil || action.PromptID <= 0 || !ok || !action.Authorized {
		return UpdatePromptResolution{}, false, nil
	}
	if writeAnswer == nil {
		return UpdatePromptResolution{}, false, fmt.Errorf("feishu update prompt writer is nil")
	}

	s.mu.Lock()
	record, ok := s.records[action.PromptID]
	if ok {
		delete(s.records, action.PromptID)
	}
	s.mu.Unlock()
	if !ok {
		return UpdatePromptResolution{}, false, nil
	}

	if err := writeAnswer(answer); err != nil {
		return UpdatePromptResolution{Record: record}, true, err
	}
	return UpdatePromptResolution{
		Record: record,
		Card:   BuildResolvedUpdatePromptCard(answer, updatePromptActorName(action)),
	}, true, nil
}

func BuildUpdatePromptCard(prompt, defaultAnswer string, promptID int) map[string]any {
	content := strings.TrimSpace(prompt)
	if content == "" {
		content = "Continue update?"
	}
	if defaultAnswer = strings.TrimSpace(defaultAnswer); defaultAnswer != "" {
		content += "\n\nDefault: `" + defaultAnswer + "`"
	}
	return map[string]any{
		"config": map[string]any{"wide_screen_mode": true},
		"header": map[string]any{
			"title":    map[string]any{"content": "Update Needs Your Input", "tag": "plain_text"},
			"template": "orange",
		},
		"elements": []map[string]any{
			{"tag": "markdown", "content": content},
			{
				"tag": "action",
				"actions": []map[string]any{
					updatePromptButton("Yes", "y", "primary", promptID),
					updatePromptButton("No", "n", "danger", promptID),
				},
			},
		},
	}
}

func BuildResolvedUpdatePromptCard(answer, actorName string) map[string]any {
	normalized, ok := normalizeUpdatePromptAnswer(answer)
	if !ok {
		normalized = "n"
	}
	yes := normalized == "y"
	label := "No"
	template := "red"
	if yes {
		label = "Yes"
		template = "green"
	}
	actor := strings.TrimSpace(actorName)
	if actor == "" {
		actor = "User"
	}
	return map[string]any{
		"config": map[string]any{"wide_screen_mode": true},
		"header": map[string]any{
			"title":    map[string]any{"content": "Update prompt answered: " + label, "tag": "plain_text"},
			"template": template,
		},
		"elements": []map[string]any{
			{"tag": "markdown", "content": "Answered by **" + actor + "**"},
		},
	}
}

func updatePromptButton(label, answer, buttonType string, promptID int) map[string]any {
	return map[string]any{
		"tag":  "button",
		"text": map[string]any{"tag": "plain_text", "content": label},
		"type": buttonType,
		"value": map[string]any{
			"hermes_update_prompt_action": answer,
			"update_prompt_id":            promptID,
		},
	}
}

func normalizeUpdatePromptAnswer(answer string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y":
		return "y", true
	case "n":
		return "n", true
	default:
		return "", false
	}
}

func updatePromptActorName(action UpdatePromptAction) string {
	if name := strings.TrimSpace(action.ActorName); name != "" {
		return name
	}
	if id := strings.TrimSpace(action.ActorID); id != "" {
		return id
	}
	return "User"
}

func SendRichTextWithEvidence(ctx context.Context, client Client, chatID, text string, opts SendOptions) DeliveryResult {
	if client == nil {
		return DeliveryResult{Evidence: FeishuEvidenceSendFailed, Error: "Feishu send client is unavailable"}
	}
	msgID, err := client.SendRichText(ctx, chatID, text, opts)
	if err != nil {
		return DeliveryResult{Evidence: FeishuEvidenceSendFailed, Error: "Feishu rich-text send failed"}
	}
	return DeliveryResult{MessageID: msgID}
}
