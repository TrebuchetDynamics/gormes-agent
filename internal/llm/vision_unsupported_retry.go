package llm

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

const (
	visionUnsupportedEvidencePlanned      = "vision_unsupported_retry_planned"
	visionUnsupportedEvidenceLimitReached = "vision_unsupported_retry_limit_reached"
	visionUnsupportedEvidenceNoImages     = "vision_unsupported_retry_no_images"
)

// VisionUnsupportedRetryRequest packages the inputs needed to decide whether a
// provider image-rejection error should be retried with text-only messages.
type VisionUnsupportedRetryRequest struct {
	Err      error
	Messages []Message
	Attempts int
}

// VisionUnsupportedRetryPlan is the bounded retry verdict for providers that
// reject native image content parts on a 4xx response.
type VisionUnsupportedRetryPlan struct {
	Retry         bool
	NewMessages   []Message
	ImagesRemoved bool
	EvidenceCode  string
}

// PlanVisionUnsupportedRetry mirrors Hermes' image-rejection recovery: for a
// 4xx provider error whose body says image/vision content is unsupported, strip
// image content parts from the in-flight request and permit one retry.
func PlanVisionUnsupportedRetry(req VisionUnsupportedRetryRequest) VisionUnsupportedRetryPlan {
	if req.Attempts > 0 {
		return VisionUnsupportedRetryPlan{EvidenceCode: visionUnsupportedEvidenceLimitReached}
	}
	if !isVisionUnsupportedProviderError(req.Err) {
		return VisionUnsupportedRetryPlan{}
	}
	messages, removed := stripVisionUnsupportedImagesFromMessages(req.Messages)
	if !removed {
		return VisionUnsupportedRetryPlan{EvidenceCode: visionUnsupportedEvidenceNoImages}
	}
	return VisionUnsupportedRetryPlan{
		Retry:         true,
		NewMessages:   messages,
		ImagesRemoved: true,
		EvidenceCode:  visionUnsupportedEvidencePlanned,
	}
}

var visionUnsupportedPhrases = []string{
	"only 'text' content type is supported",
	"only text content type is supported",
	"image_url is not supported",
	"image content is not supported",
	"multimodal is not supported",
	"multimodal content is not supported",
	"multimodal input is not supported",
	"vision is not supported",
	"vision input is not supported",
	"does not support images",
	"does not support image input",
	"does not support multimodal",
	"does not support vision",
	"model does not support image",
	"unsupported content type: image_url",
	"unknown variant image_url",
	"unknown variant `image_url`",
}

func isVisionUnsupportedProviderError(err error) bool {
	if err == nil {
		return false
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	if httpErr.Status < http.StatusBadRequest || httpErr.Status >= http.StatusInternalServerError {
		return false
	}
	message, combined, code := providerHTTPErrorText(httpErr)
	combined = strings.ToLower(strings.Join([]string{combined, message, code, httpErr.Body, httpErr.Error()}, " "))
	for _, phrase := range visionUnsupportedPhrases {
		if strings.Contains(combined, phrase) {
			return true
		}
	}
	return false
}

func stripVisionUnsupportedImagesFromMessages(messages []Message) ([]Message, bool) {
	out := make([]Message, 0, len(messages))
	removedAny := false
	for _, msg := range messages {
		next := cloneVisionRetryMessage(msg)
		if len(next.ContentParts) > 0 {
			parts := make([]MessageContentPart, 0, len(next.ContentParts))
			for _, part := range next.ContentParts {
				if isVisionImagePart(part) {
					removedAny = true
					continue
				}
				parts = append(parts, part)
			}
			next.ContentParts = parts
			if len(next.ContentParts) == 0 {
				next.ContentParts = nil
				if strings.EqualFold(strings.TrimSpace(next.Role), "tool") && strings.TrimSpace(next.Content) == "" {
					next.Content = "[image content removed - server does not support images]"
				} else if strings.TrimSpace(next.Content) == "" {
					continue
				}
			}
		}
		out = append(out, next)
	}
	return out, removedAny
}

func cloneVisionRetryMessage(msg Message) Message {
	next := msg
	if len(msg.ContentParts) > 0 {
		next.ContentParts = append([]MessageContentPart(nil), msg.ContentParts...)
	}
	if len(msg.ToolCalls) > 0 {
		next.ToolCalls = append([]ToolCall(nil), msg.ToolCalls...)
	}
	return next
}

func isVisionImagePart(part MessageContentPart) bool {
	switch strings.ToLower(strings.TrimSpace(part.Type)) {
	case "image_url", "input_image", "image":
		return true
	default:
		return false
	}
}

func (c *httpClient) prepareVisionUnsupportedRequest(req ChatRequest) ChatRequest {
	if strings.TrimSpace(req.SessionID) == "" || !c.sessionVisionUnsupported(req.SessionID) {
		return req
	}
	messages, removed := stripVisionUnsupportedImagesFromMessages(req.Messages)
	if !removed {
		return req
	}
	next := req
	next.Messages = messages
	slog.Info("vision_unsupported_session_text_only",
		"session_id", req.SessionID,
		"model", req.Model,
		"evidence", visionUnsupportedEvidencePlanned,
	)
	return next
}

func (c *httpClient) planVisionUnsupportedRetry(req ChatRequest, err error) (ChatRequest, bool) {
	plan := PlanVisionUnsupportedRetry(VisionUnsupportedRetryRequest{
		Err:      err,
		Messages: req.Messages,
		Attempts: 0,
	})
	if !plan.Retry {
		return ChatRequest{}, false
	}
	if strings.TrimSpace(req.SessionID) != "" {
		c.markVisionUnsupportedSession(req.SessionID)
	}
	next := req
	next.Messages = plan.NewMessages
	slog.Info("vision_unsupported_retry",
		"session_id", req.SessionID,
		"model", req.Model,
		"evidence", plan.EvidenceCode,
		"images_removed", plan.ImagesRemoved,
	)
	return next, true
}

func (c *httpClient) sessionVisionUnsupported(sessionID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.visionUnsupportedSessions[strings.TrimSpace(sessionID)]
}

func (c *httpClient) markVisionUnsupportedSession(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.visionUnsupportedSessions == nil {
		c.visionUnsupportedSessions = map[string]bool{}
	}
	c.visionUnsupportedSessions[sessionID] = true
}
