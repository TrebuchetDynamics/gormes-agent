package mattermost

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/threadtext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	MattermostEvidenceConfigMissing        = "mattermost_config_missing"
	MattermostEvidenceAuthFailed           = "mattermost_auth_failed"
	MattermostEvidenceWSUnavailable        = "mattermost_ws_unavailable"
	MattermostEvidenceUploadFailed         = "mattermost_upload_failed"
	MattermostEvidenceTransportUnavailable = "mattermost_transport_unavailable"
)

const (
	defaultReconnectBase = 2 * time.Second
	defaultReconnectMax  = 60 * time.Second
)

type Config struct {
	BaseURL              string
	Token                string
	HomeChannel          string
	HomeChannelName      string
	ReplyMode            string
	AllowedChannels      []string
	FreeResponseChannels []string
	RequireMention       bool
}

type Bootstrap struct {
	cfg       Config
	factory   TransportFactory
	transport Transport
	seam      *Seam
	reconnect ReconnectPolicy
}

type TransportFactory func(Config) (Transport, error)

type Transport interface {
	Get(context.Context, string, map[string]string) (mattermostRESTResponse, error)
	Post(context.Context, string, map[string]any, map[string]string) (mattermostRESTResponse, error)
	Put(context.Context, string, map[string]any, map[string]string) (mattermostRESTResponse, error)
	UploadFile(context.Context, MattermostUpload, map[string]string) (mattermostRESTResponse, error)
}

type BootstrapResult struct {
	Ready       bool
	Evidence    string
	Error       string
	BotUserID   string
	BotUsername string
	BaseURL     string
}

type RESTEvidence struct {
	Evidence string
	Status   int
	Error    string
}

type SendResult struct {
	Success   bool
	MessageID string
	Evidence  string
	Error     string
}

type MattermostUpload struct {
	ChannelID   string
	Data        []byte
	Filename    string
	ContentType string
}

type ReconnectPolicy struct {
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxAttempts int
}

type ReconnectOutcome struct {
	Retry    bool
	Attempt  int
	Delay    time.Duration
	Evidence string
	Error    string
}

type mattermostRESTResponse struct {
	Status int
	JSON   map[string]any
	Body   string
}

func NewBootstrap(cfg Config, factory TransportFactory) *Bootstrap {
	return &Bootstrap{
		cfg:     normalizeConfig(cfg),
		factory: factory,
		reconnect: ReconnectPolicy{
			BaseDelay:   defaultReconnectBase,
			MaxDelay:    defaultReconnectMax,
			MaxAttempts: 5,
		},
	}
}

func ResolveBootstrapConfig(extra map[string]string, getenv func(string) string) Config {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	get := func(key, envKey string) string {
		if extra != nil {
			if v := strings.TrimSpace(extra[key]); v != "" {
				return v
			}
		}
		return strings.TrimSpace(getenv(envKey))
	}
	return normalizeConfig(Config{
		BaseURL:              get("url", "MATTERMOST_URL"),
		Token:                get("token", "MATTERMOST_TOKEN"),
		HomeChannel:          get("home_channel", "MATTERMOST_HOME_CHANNEL"),
		HomeChannelName:      get("home_channel_name", "MATTERMOST_HOME_CHANNEL_NAME"),
		ReplyMode:            get("reply_mode", "MATTERMOST_REPLY_MODE"),
		AllowedChannels:      splitList(get("allowed_channels", "MATTERMOST_ALLOWED_CHANNELS")),
		FreeResponseChannels: splitList(get("free_response_channels", "MATTERMOST_FREE_RESPONSE_CHANNELS")),
		RequireMention:       parseBoolDefault(get("require_mention", "MATTERMOST_REQUIRE_MENTION"), true),
	})
}

func (b *Bootstrap) Start(ctx context.Context) BootstrapResult {
	if b.cfg.BaseURL == "" || b.cfg.Token == "" {
		return BootstrapResult{Evidence: MattermostEvidenceConfigMissing, Error: "Mattermost URL and token are required"}
	}
	if b.factory == nil {
		return BootstrapResult{Evidence: MattermostEvidenceTransportUnavailable, Error: "Mattermost transport factory is not configured"}
	}
	transport, err := b.factory(b.cfg)
	if err != nil {
		return BootstrapResult{Evidence: MattermostEvidenceTransportUnavailable, Error: b.sanitizeError(err)}
	}
	b.transport = transport

	me, evidence := b.APIGet(ctx, "users/me")
	if evidence.Evidence != "" {
		return BootstrapResult{Evidence: MattermostEvidenceAuthFailed, Error: evidence.Error, BaseURL: b.cfg.BaseURL}
	}
	botID := stringValue(me, "id")
	if botID == "" {
		return BootstrapResult{Evidence: MattermostEvidenceAuthFailed, Error: "Mattermost users/me response missing bot id", BaseURL: b.cfg.BaseURL}
	}
	botUsername := stringValue(me, "username")
	b.seam = NewSeam(threadReplyMode(b.cfg.ReplyMode), MentionGatingInputs{
		Kind:            requireMentionKind(b.cfg.RequireMention),
		AllowedChannels: boolSet(b.cfg.AllowedChannels),
		FreeChannelIDs:  boolSet(b.cfg.FreeResponseChannels),
	}, botID, nil)
	return BootstrapResult{
		Ready:       true,
		BotUserID:   botID,
		BotUsername: botUsername,
		BaseURL:     b.cfg.BaseURL,
	}
}

func (b *Bootstrap) APIGet(ctx context.Context, path string) (map[string]any, RESTEvidence) {
	if b.transport == nil {
		return nil, RESTEvidence{Evidence: MattermostEvidenceTransportUnavailable, Error: "Mattermost transport is not configured"}
	}
	resp, err := b.transport.Get(ctx, cleanAPIPath(path), b.authHeaders())
	return b.handleREST("GET", path, resp, err)
}

func (b *Bootstrap) APIPost(ctx context.Context, path string, payload map[string]any) (map[string]any, RESTEvidence) {
	if b.transport == nil {
		return nil, RESTEvidence{Evidence: MattermostEvidenceTransportUnavailable, Error: "Mattermost transport is not configured"}
	}
	resp, err := b.transport.Post(ctx, cleanAPIPath(path), payload, b.authHeaders())
	return b.handleREST("POST", path, resp, err)
}

func (b *Bootstrap) APIPut(ctx context.Context, path string, payload map[string]any) (map[string]any, RESTEvidence) {
	if b.transport == nil {
		return nil, RESTEvidence{Evidence: MattermostEvidenceTransportUnavailable, Error: "Mattermost transport is not configured"}
	}
	resp, err := b.transport.Put(ctx, cleanAPIPath(path), payload, b.authHeaders())
	return b.handleREST("PUT", path, resp, err)
}

func (b *Bootstrap) handleREST(method, path string, resp mattermostRESTResponse, err error) (map[string]any, RESTEvidence) {
	if err != nil {
		return nil, RESTEvidence{Evidence: MattermostEvidenceTransportUnavailable, Error: b.sanitizeError(err)}
	}
	if resp.Status >= 400 {
		return nil, RESTEvidence{
			Evidence: MattermostEvidenceTransportUnavailable,
			Status:   resp.Status,
			Error:    fmt.Sprintf("Mattermost %s %s failed with status %d: %s", method, cleanAPIPath(path), resp.Status, b.sanitizeText(resp.Body)),
		}
	}
	return resp.JSON, RESTEvidence{}
}

func (b *Bootstrap) Send(ctx context.Context, channelID, content, replyTo string) SendResult {
	if strings.TrimSpace(content) == "" {
		return SendResult{Success: true}
	}
	payload := map[string]any{
		"channel_id": strings.TrimSpace(channelID),
		"message":    formatMessage(content),
	}
	if strings.EqualFold(b.cfg.ReplyMode, "thread") && strings.TrimSpace(replyTo) != "" {
		payload["root_id"] = strings.TrimSpace(replyTo)
	}
	data, evidence := b.APIPost(ctx, "posts", payload)
	if evidence.Evidence != "" {
		return SendResult{Evidence: evidence.Evidence, Error: evidence.Error}
	}
	msgID := stringValue(data, "id")
	if msgID == "" {
		return SendResult{Evidence: MattermostEvidenceTransportUnavailable, Error: "Mattermost post response missing id"}
	}
	return SendResult{Success: true, MessageID: msgID}
}

func (b *Bootstrap) Edit(ctx context.Context, messageID, content string) SendResult {
	data, evidence := b.APIPut(ctx, "posts/"+strings.TrimSpace(messageID)+"/patch", map[string]any{"message": formatMessage(content)})
	if evidence.Evidence != "" {
		return SendResult{Evidence: evidence.Evidence, Error: evidence.Error}
	}
	msgID := stringValue(data, "id")
	if msgID == "" {
		return SendResult{Evidence: MattermostEvidenceTransportUnavailable, Error: "Mattermost edit response missing id"}
	}
	return SendResult{Success: true, MessageID: msgID}
}

func (b *Bootstrap) UploadAndPost(ctx context.Context, upload MattermostUpload, caption, replyTo string) SendResult {
	if b.transport == nil {
		return SendResult{Evidence: MattermostEvidenceTransportUnavailable, Error: "Mattermost transport is not configured"}
	}
	upload.ChannelID = strings.TrimSpace(upload.ChannelID)
	upload.Filename = strings.TrimSpace(upload.Filename)
	upload.ContentType = firstNonEmpty(strings.TrimSpace(upload.ContentType), "application/octet-stream")
	resp, err := b.transport.UploadFile(ctx, upload, uploadHeaders(b.cfg.Token))
	if err != nil {
		return SendResult{Evidence: MattermostEvidenceUploadFailed, Error: b.sanitizeError(err)}
	}
	if resp.Status >= 400 {
		return SendResult{Evidence: MattermostEvidenceUploadFailed, Error: b.sanitizeText(resp.Body)}
	}
	fileIDs := stringSlice(resp.JSON["file_ids"])
	if len(fileIDs) == 0 {
		if infos, ok := resp.JSON["file_infos"].([]any); ok {
			for _, info := range infos {
				if m, ok := info.(map[string]any); ok {
					if id := stringValue(m, "id"); id != "" {
						fileIDs = append(fileIDs, id)
					}
				}
			}
		}
	}
	if len(fileIDs) == 0 {
		return SendResult{Evidence: MattermostEvidenceUploadFailed, Error: "Mattermost upload response missing file id"}
	}
	payload := map[string]any{
		"channel_id": upload.ChannelID,
		"message":    strings.TrimSpace(caption),
		"file_ids":   fileIDs,
	}
	if strings.EqualFold(b.cfg.ReplyMode, "thread") && strings.TrimSpace(replyTo) != "" {
		payload["root_id"] = strings.TrimSpace(replyTo)
	}
	data, evidence := b.APIPost(ctx, "posts", payload)
	if evidence.Evidence != "" {
		return SendResult{Evidence: evidence.Evidence, Error: evidence.Error}
	}
	return SendResult{Success: true, MessageID: stringValue(data, "id")}
}

func (b *Bootstrap) HandleWebsocketEvent(rawJSON string) (gateway.InboundEvent, bool) {
	if b.seam == nil {
		return gateway.InboundEvent{}, false
	}
	ev, ok := b.seam.ParsePostedEvent(rawJSON)
	if !ok {
		return gateway.InboundEvent{}, false
	}
	return ev, true
}

func (b *Bootstrap) StepReconnect(err error) ReconnectOutcome {
	if err == nil {
		return ReconnectOutcome{}
	}
	delay := b.reconnect.BaseDelay
	if delay <= 0 {
		delay = defaultReconnectBase
	}
	maxDelay := b.reconnect.MaxDelay
	if maxDelay <= 0 {
		maxDelay = defaultReconnectMax
	}
	attempt := 1
	next := delay
	if next > maxDelay {
		next = maxDelay
	}
	return ReconnectOutcome{Retry: true, Attempt: attempt, Delay: next, Evidence: MattermostEvidenceWSUnavailable, Error: b.sanitizeError(err)}
}

func (b *Bootstrap) authHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + b.cfg.Token,
		"Content-Type":  "application/json",
	}
}

func uploadHeaders(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + strings.TrimSpace(token)}
}

func (b *Bootstrap) sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return b.sanitizeText(err.Error())
}

func (b *Bootstrap) sanitizeText(value string) string {
	value = sanitizeMattermostText(value, b.cfg.Token)
	if len(value) > 200 {
		return value[:200] + "..."
	}
	return value
}

func normalizeConfig(cfg Config) Config {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Token = strings.TrimSpace(cfg.Token)
	cfg.HomeChannel = strings.TrimSpace(cfg.HomeChannel)
	cfg.HomeChannelName = strings.TrimSpace(cfg.HomeChannelName)
	cfg.ReplyMode = strings.ToLower(strings.TrimSpace(cfg.ReplyMode))
	if cfg.ReplyMode == "" {
		cfg.ReplyMode = "off"
	}
	cfg.AllowedChannels = compactStrings(cfg.AllowedChannels)
	cfg.FreeResponseChannels = compactStrings(cfg.FreeResponseChannels)
	return cfg
}

func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	return compactStrings(parts)
}

func compactStrings(values []string) []string { return channelutil.CompactStrings(values) }

func parseBoolDefault(raw string, def bool) bool { return channelutil.ParseBoolDefault(raw, def) }

func boolSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func requireMentionKind(require bool) GatingKind {
	if require {
		return KindGated
	}
	return KindFree
}

func threadReplyMode(mode string) threadtext.ReplyMode {
	if strings.EqualFold(mode, string(threadtext.ReplyModeThread)) {
		return threadtext.ReplyModeThread
	}
	return threadtext.ReplyModeFlat
}

func cleanAPIPath(path string) string {
	return strings.TrimPrefix(strings.TrimSpace(path), "/")
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func stringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return compactStrings(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func firstNonEmpty(values ...string) string { return channelutil.FirstNonEmpty(values...) }

func formatMessage(content string) string {
	return strings.TrimSpace(content)
}

func sanitizeMattermostText(value, token string) string {
	if token = strings.TrimSpace(token); token != "" {
		value = strings.ReplaceAll(value, token, "[redacted]")
	}
	return strings.TrimSpace(value)
}
