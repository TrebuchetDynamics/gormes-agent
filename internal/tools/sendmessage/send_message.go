package sendmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gwtarget "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/routing"
)

// Target is a model-visible channel destination returned by action=list.
type Target struct {
	Platform string `json:"platform"`
	ChatID   string `json:"chat_id,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (t Target) String() string {
	parsed := gwtarget.Target{Platform: t.Platform, ChatID: t.ChatID, ThreadID: t.ThreadID}
	return parsed.String()
}

// Directory provides a sanitized, model-safe channel directory for action=list
// and friendly-name resolution. Implementations must not expose credentials.
type Directory interface {
	ListTargets(context.Context) ([]Target, error)
	ResolveTarget(ctx context.Context, platform, ref string) (target string, ok bool, err error)
}

// SendRequest is the validated send payload passed to the runtime sender.
type SendRequest struct {
	Target  gwtarget.Target
	Raw     string
	Message string
}

// Sender sends a message after schema, required-field, target, and friendly-name
// validation have already succeeded.
type Sender interface {
	SendMessage(context.Context, SendRequest) error
}

// Options configures the hermetic send_message tool seams.
type Options struct {
	Directory Directory
	Sender    Sender
}

type sendFuncSender struct {
	fn func(target, message string) error
}

func (s sendFuncSender) SendMessage(_ context.Context, req SendRequest) error {
	return s.fn(req.Target.String(), req.Message)
}

type SendMessageTool struct {
	directory Directory
	sender    Sender
}

func NewSendMessageTool(fn func(target, message string) error) *SendMessageTool {
	var sender Sender
	if fn != nil {
		sender = sendFuncSender{fn: fn}
	}
	return NewSendMessageToolWithOptions(Options{Sender: sender})
}

func NewSendMessageToolWithOptions(opts Options) *SendMessageTool {
	return &SendMessageTool{directory: opts.Directory, sender: opts.Sender}
}

func (*SendMessageTool) Name() string { return "send_message" }
func (*SendMessageTool) Description() string {
	return "Send a message to a connected messaging platform, or list available targets. When sending to a specific channel or person, call send_message(action='list') first to see available targets."
}
func (*SendMessageTool) Timeout() time.Duration { return 10 * time.Second }

func (*SendMessageTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["send","list"],"description":"Action to perform. 'send' (default) sends a message. 'list' returns available channels/contacts across connected platforms."},
			"target":{"type":"string","description":"Delivery target. Format: 'platform' (uses home channel), 'platform:#channel-name', 'platform:chat_id', or 'platform:chat_id:thread_id'. Examples: 'telegram', 'telegram:-1001234567890:17585', 'discord:999888777:555444333', 'discord:#bot-home', 'slack:#engineering'."},
			"message":{"type":"string","description":"The message text to send. Media delivery remains runtime-adapter specific; include any attachment intent in the message for adapters that support it."}
		},
		"required":[]
	}`)
}

func (t *SendMessageTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var in struct {
		Action  string `json:"action"`
		Target  string `json:"target"`
		Message string `json:"message"`
	}
	if len(strings.TrimSpace(string(args))) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
	}
	action := strings.ToLower(strings.TrimSpace(in.Action))
	if action == "" {
		action = "send"
	}
	switch action {
	case "list":
		return t.handleList(ctx)
	case "send":
		return t.handleSend(ctx, strings.TrimSpace(in.Target), in.Message)
	default:
		return json.Marshal(toolError(fmt.Sprintf("Unknown action: %s", action)))
	}
}

func (t *SendMessageTool) handleList(ctx context.Context) (json.RawMessage, error) {
	if t.directory == nil {
		return json.Marshal(map[string]any{
			"success":  false,
			"evidence": "send_message_directory_unavailable",
			"error":    "Channel directory is unavailable; configure a gateway/channel directory before listing send_message targets.",
			"degraded": true,
		})
	}
	targets, err := t.directory.ListTargets(ctx)
	if err != nil {
		return json.Marshal(map[string]any{
			"success":  false,
			"evidence": "send_message_directory_unavailable",
			"error":    "Failed to load channel directory: " + sanitizeError(err),
			"degraded": true,
		})
	}
	return json.Marshal(map[string]any{"success": true, "targets": targets})
}

func (t *SendMessageTool) handleSend(ctx context.Context, rawTarget, message string) (json.RawMessage, error) {
	if rawTarget == "" || strings.TrimSpace(message) == "" {
		return json.Marshal(toolError("Both 'target' and 'message' are required when action='send'"))
	}
	resolvedTarget := rawTarget
	platform, ref, hasRef := splitTarget(rawTarget)
	if hasRef && isFriendlyRef(ref) {
		if t.directory == nil {
			return json.Marshal(unresolvedTarget(platform, ref, "Channel directory is unavailable."))
		}
		resolved, ok, err := t.directory.ResolveTarget(ctx, platform, ref)
		if err != nil {
			return json.Marshal(unresolvedTarget(platform, ref, "Channel directory lookup failed: "+sanitizeError(err)))
		}
		if !ok || strings.TrimSpace(resolved) == "" {
			return json.Marshal(unresolvedTarget(platform, ref, "Could not resolve target."))
		}
		if !strings.Contains(resolved, ":") {
			resolvedTarget = platform + ":" + strings.TrimSpace(resolved)
		} else {
			resolvedTarget = strings.TrimSpace(resolved)
		}
	}
	parsed, err := gwtarget.ParseTarget(resolvedTarget, nil)
	if err != nil {
		return json.Marshal(map[string]any{
			"success":  false,
			"evidence": "send_message_invalid_target",
			"target":   rawTarget,
			"error":    err.Error(),
		})
	}
	if t.sender == nil {
		return json.Marshal(map[string]any{
			"success":  false,
			"evidence": "send_message_backend_unavailable",
			"target":   parsed.String(),
			"error":    "send_message backend is unavailable; configure a gateway sender before sending messages.",
			"degraded": true,
		})
	}
	if err := t.sender.SendMessage(ctx, SendRequest{Target: parsed, Raw: rawTarget, Message: message}); err != nil {
		return json.Marshal(map[string]any{
			"success": false,
			"target":  parsed.String(),
			"error":   sanitizeError(err),
		})
	}
	return json.Marshal(map[string]any{
		"success": true,
		"target":  parsed.String(),
		"message": "Message sent successfully.",
	})
}

func splitTarget(raw string) (platform, ref string, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if len(parts) != 2 {
		return strings.ToLower(strings.TrimSpace(raw)), "", false
	}
	return strings.ToLower(strings.TrimSpace(parts[0])), strings.TrimSpace(parts[1]), true
}

func isFriendlyRef(ref string) bool {
	return strings.HasPrefix(strings.TrimSpace(ref), "#")
}

func unresolvedTarget(platform, ref, detail string) map[string]any {
	return map[string]any{
		"success":  false,
		"evidence": "send_message_target_unresolved",
		"target":   strings.TrimSpace(platform + ":" + ref),
		"error":    fmt.Sprintf("Could not resolve '%s' on %s. Use send_message(action='list') to see available targets.", ref, platform),
		"detail":   detail,
	}
}

func toolError(message string) map[string]any {
	return map[string]any{"error": message}
}

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
