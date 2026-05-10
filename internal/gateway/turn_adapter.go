package gateway

import (
	"context"
	"log/slog"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// TurnRequest is the channel-neutral turn dispatch envelope every channel
// produces and the native runtime adapter consumes. Channel adapters translate
// SDK-specific traffic into a TurnRequest; the adapter here never inspects the
// originating platform name to make a runtime decision, so provider/runtime
// fixes preserve Hermes channel parity instead of hard-coding Telegram (or any
// other) channel behavior.
type TurnRequest struct {
	// Channel is the channel adapter that owns reply delivery for this turn.
	// Channel-specific identity safety, require-mention, delivery, and thread
	// rules remain inside the adapter — TurnAdapter only sends through Send.
	Channel Channel

	// Source carries the gateway-facing origin of the turn (chat/thread,
	// sender identity).
	Source SessionSource

	// SessionKey is the gateway's session map key for this turn (typically
	// "<platform>:<chat_id>").
	SessionKey string

	// ResolvedSessionID is the kernel session id resolved from SessionKey.
	ResolvedSessionID string

	// SubmitText is the rendered text submitted through the kernel runtime,
	// already merged with attachments and reply context by the channel adapter
	// or the gateway helpers.
	SubmitText string

	// SessionContext is the deterministic per-turn prompt block injected ahead
	// of the user message.
	SessionContext string

	// Attachments references the channel-normalized inbound media for the
	// turn. The shared adapter forwards them to OnTurnStart so observers can
	// record them; it does not download or rewrite them.
	Attachments []Attachment

	// CommandKind classifies the inbound command/admission for runtime
	// decisions (e.g. EventSubmit).
	CommandKind EventKind

	// ReplyChatID is the channel-side chat id the runtime should reply to.
	ReplyChatID string

	// ReplyMsgID is the optional originating platform message id used for
	// hook routing or reply threads.
	ReplyMsgID string
}

// TurnAdapter dispatches a TurnRequest through the native runtime. It does
// not embed channel-specific behavior; channel adapters are responsible for
// translating their SDK traffic into TurnRequest and for honoring channel-side
// identity, require-mention, delivery, and thread rules. On runtime failure
// the adapter renders a sanitized safe-error reply through the request's
// Channel; the raw error is never exposed externally.
type TurnAdapter struct {
	// Submitter is the native runtime entry point. Required.
	Submitter kernelSubmitter

	// OnTurnStart is invoked with the channel-neutral request before the
	// runtime submit. Hooks/state observers use this to record the
	// channel-neutral fields without inspecting the channel name.
	OnTurnStart func(req TurnRequest)

	// OnTurnFailure is invoked when the native runtime submit fails. It
	// receives the original (raw) error so the caller can clear active turn
	// state and emit private telemetry; the adapter still renders a sanitized
	// safe-error reply through req.Channel itself.
	OnTurnFailure func(req TurnRequest, err error)
}

// Dispatch submits the turn through the native runtime. On runtime failure,
// it calls OnTurnFailure and sends a sanitized safe-error reply through
// req.Channel; the raw error is never exposed externally.
func (a *TurnAdapter) Dispatch(ctx context.Context, req TurnRequest) error {
	if a.OnTurnStart != nil {
		a.OnTurnStart(req)
	}
	if a.Submitter == nil {
		return nil
	}
	err := a.Submitter.Submit(kernel.PlatformEvent{
		Kind:           kernel.PlatformEventSubmit,
		Text:           req.SubmitText,
		ContentParts:   imageContentPartsFromAttachments(req.SubmitText, req.Attachments),
		SessionID:      req.ResolvedSessionID,
		SessionContext: req.SessionContext,
	})
	if err == nil {
		return nil
	}
	if a.OnTurnFailure != nil {
		a.OnTurnFailure(req, err)
	}
	if req.Channel != nil {
		_, _ = req.Channel.Send(ctx, req.ReplyChatID, SafeExternalChannelError(err))
	}
	return err
}

// SafeExternalChannelError maps a raw provider/runtime error into a sanitized
// reply text safe for delivery on external channels. The helper never
// includes the raw error string so secrets or internal classifications cannot
// leak into Telegram, Slack, Discord, WhatsApp, BlueBubbles, or future
// channels. Callers that need diagnostic detail should record the underlying
// error through telemetry, not by sending it to the channel.
func SafeExternalChannelError(_ error) string {
	return externalChannelSafeBusyReply
}

// externalChannelSafeBusyReply is the canonical sanitized busy reply already
// emitted by the gateway when the kernel submit path fails. Reusing this
// vocabulary keeps Hermes/Honcho channel parity stable.
const externalChannelSafeBusyReply = "Busy — try again in a second."
const visionPreAnalysisUnavailableMarker = "vision_pre_analysis_unavailable"

type imageInputModeOptions struct {
	Mode            hermes.ImageInputMode
	AuxiliaryVision hermes.AuxiliaryVisionConfig
	Provider        string
	Model           string
}

func imagePayloadFromAttachments(userText, submitText string, attachments []Attachment, opts imageInputModeOptions) (string, []hermes.MessageContentPart) {
	if !hasPhotoAttachments(attachments) {
		return submitText, nil
	}
	mode := decideImageInputModeForTurn(opts)
	if mode == hermes.ImageInputModeNative {
		return submitText, imageContentPartsFromAttachments(userText, attachments)
	}
	slog.Info("image_input_mode_text_degraded",
		"evidence", visionPreAnalysisUnavailableMarker,
		"provider", strings.TrimSpace(opts.Provider),
		"model", strings.TrimSpace(opts.Model))
	return degradedVisionSubmitText(submitText), nil
}

func decideImageInputModeForTurn(opts imageInputModeOptions) hermes.ImageInputMode {
	metadata := hermes.LookupModelMetadata(hermes.ModelRegistryQuery{
		Provider: opts.Provider,
		Model:    opts.Model,
	})
	return hermes.DecideImageInputMode(hermes.ImageRoutingConfig{
		Mode:                  opts.Mode,
		AuxiliaryVision:       opts.AuxiliaryVision,
		ModelVisionCapability: metadata.Capabilities.Vision,
	})
}

func degradedVisionSubmitText(submitText string) string {
	text := strings.TrimSpace(submitText)
	if strings.Contains(text, visionPreAnalysisUnavailableMarker) {
		return text
	}
	marker := "[" + visionPreAnalysisUnavailableMarker + ": no vision_analyze backend configured]"
	if text == "" {
		return marker
	}
	return marker + "\n\n" + text
}

func hasPhotoAttachments(attachments []Attachment) bool {
	for _, att := range attachments {
		if strings.EqualFold(strings.TrimSpace(att.Kind), "photo") {
			return true
		}
	}
	return false
}

// imageContentPartsFromAttachments materializes channel-cached photo bytes into
// Hermes native multimodal content parts. The first content part is the user
// caption, or the default image prompt when the caption is empty, with one
// local path hint per attached image so tools can address the same cache file.
// Unreadable files are skipped so the kernel never receives a broken image_url.
// Non-photo attachments are skipped — voice has its own transcriber resolver
// and documents lack a defined multimodal contract today.
func imageContentPartsFromAttachments(userText string, attachments []Attachment) []hermes.MessageContentPart {
	if len(attachments) == 0 {
		return nil
	}
	imagePaths := make([]string, 0, len(attachments))
	emptyPaths := 0
	for _, att := range attachments {
		if !strings.EqualFold(strings.TrimSpace(att.Kind), "photo") {
			continue
		}
		path := strings.TrimSpace(att.URL)
		if path == "" {
			emptyPaths++
			continue
		}
		imagePaths = append(imagePaths, path)
	}
	if len(imagePaths) == 0 && emptyPaths == 0 {
		return nil
	}
	parts, skipped := hermes.BuildNativeImageContentParts(userText, imagePaths)
	if len(parts) > 0 || len(skipped) > 0 || emptyPaths > 0 {
		slog.Info("imageContentPartsFromAttachments",
			"image_parts", countImageURLContentParts(parts),
			"skipped", len(skipped)+emptyPaths,
			"attachment_count", len(attachments))
	}
	return parts
}

func countImageURLContentParts(parts []hermes.MessageContentPart) int {
	count := 0
	for _, part := range parts {
		if part.Type == "image_url" {
			count++
		}
	}
	return count
}
