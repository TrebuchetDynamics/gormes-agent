package toolpayload

import (
	"encoding/json"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func AppendSubdirectoryHintToContentParts(parts []llm.MessageContentPart, hint string) []llm.MessageContentPart {
	out := cloneContentParts(parts)
	for i := range out {
		if strings.EqualFold(out[i].Type, "text") {
			out[i].Text += hint
			return out
		}
	}
	return append([]llm.MessageContentPart{{Type: "text", Text: hint}}, out...)
}

func MultimodalResult(payload json.RawMessage) (string, []llm.MessageContentPart, bool) {
	var envelope struct {
		Multimodal  bool              `json:"_multimodal"`
		TextSummary string            `json:"text_summary"`
		Content     []json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || !envelope.Multimodal || len(envelope.Content) == 0 {
		return "", nil, false
	}
	parts := make([]llm.MessageContentPart, 0, len(envelope.Content))
	for _, raw := range envelope.Content {
		part, ok := ContentPart(raw)
		if ok {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "", nil, false
	}
	summary := strings.TrimSpace(envelope.TextSummary)
	if summary == "" {
		for _, part := range parts {
			if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
				summary = strings.TrimSpace(part.Text)
				break
			}
		}
	}
	if summary == "" {
		summary = "Multimodal tool result attached."
	}
	return summary, parts, true
}

func ContentPart(raw json.RawMessage) (llm.MessageContentPart, bool) {
	var node map[string]any
	if err := json.Unmarshal(raw, &node); err != nil {
		return llm.MessageContentPart{}, false
	}
	partType := strings.ToLower(strings.TrimSpace(asString(node["type"])))
	switch partType {
	case "text", "input_text", "output_text":
		text := asString(node["text"])
		if strings.TrimSpace(text) == "" {
			return llm.MessageContentPart{}, false
		}
		return llm.MessageContentPart{Type: "text", Text: text}, true
	case "image_url", "input_image", "image":
		url, detail := imageURLPart(node)
		if strings.TrimSpace(url) == "" {
			return llm.MessageContentPart{}, false
		}
		return llm.MessageContentPart{Type: "image_url", ImageURL: url, Detail: detail}, true
	default:
		return llm.MessageContentPart{}, false
	}
}

func imageURLPart(node map[string]any) (string, string) {
	detail := strings.TrimSpace(asString(node["detail"]))
	switch image := node["image_url"].(type) {
	case string:
		return image, detail
	case map[string]any:
		if detail == "" {
			detail = strings.TrimSpace(asString(image["detail"]))
		}
		return asString(image["url"]), detail
	default:
		return asString(node["url"]), detail
	}
}

func asString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func cloneContentParts(parts []llm.MessageContentPart) []llm.MessageContentPart {
	if len(parts) == 0 {
		return nil
	}
	out := make([]llm.MessageContentPart, len(parts))
	copy(out, parts)
	return out
}
