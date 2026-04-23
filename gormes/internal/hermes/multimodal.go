package hermes

import (
	"encoding/base64"
	"fmt"
	"strings"

	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type ContentPartType string

const (
	ContentPartText  ContentPartType = "text"
	ContentPartImage ContentPartType = "image"
)

// ContentPart extends Message with canonical multimodal blocks while keeping
// Message.Content as the legacy plain-text shorthand.
type ContentPart struct {
	Type     ContentPartType `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL string          `json:"image_url,omitempty"`
	Detail   string          `json:"detail,omitempty"`
}

type normalizedImagePart struct {
	url        string
	mediaType  string
	base64Data string
	detail     string
}

type openAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type openAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

func messageParts(msg Message) []ContentPart {
	parts := make([]ContentPart, 0, 1+len(msg.Parts))
	if msg.Content != "" {
		parts = append(parts, ContentPart{Type: ContentPartText, Text: msg.Content})
	}
	parts = append(parts, msg.Parts...)
	return parts
}

func openAIChatContent(msg Message) (any, error) {
	if len(msg.Parts) == 0 {
		return msg.Content, nil
	}
	parts, err := openAIChatParts(msg)
	if err != nil {
		return nil, err
	}
	return parts, nil
}

func openAIChatParts(msg Message) ([]openAIContentPart, error) {
	parts := messageParts(msg)
	out := make([]openAIContentPart, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case ContentPartText:
			out = append(out, openAIContentPart{Type: "text", Text: part.Text})
		case ContentPartImage:
			image, err := normalizeImagePart(part)
			if err != nil {
				return nil, err
			}
			out = append(out, openAIContentPart{
				Type: "image_url",
				ImageURL: &openAIImageURL{
					URL:    image.url,
					Detail: imageDetail(image.detail),
				},
			})
		default:
			return nil, fmt.Errorf("unsupported content part type %q", part.Type)
		}
	}
	return out, nil
}

func codexMessageContent(msg Message) (any, error) {
	if len(msg.Parts) == 0 {
		return msg.Content, nil
	}
	parts := messageParts(msg)
	out := make([]codexContentPart, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case ContentPartText:
			out = append(out, codexContentPart{Type: "input_text", Text: part.Text})
		case ContentPartImage:
			image, err := normalizeImagePart(part)
			if err != nil {
				return nil, err
			}
			out = append(out, codexContentPart{
				Type:     "input_image",
				ImageURL: image.url,
				Detail:   imageDetail(image.detail),
			})
		default:
			return nil, fmt.Errorf("unsupported content part type %q", part.Type)
		}
	}
	return out, nil
}

func anthropicMessageBlocks(msg Message) ([]anthropicContentBlock, error) {
	parts := messageParts(msg)
	out := make([]anthropicContentBlock, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case ContentPartText:
			out = append(out, anthropicContentBlock{Type: "text", Text: part.Text})
		case ContentPartImage:
			image, err := normalizeImagePart(part)
			if err != nil {
				return nil, err
			}
			source := &anthropicContentSource{}
			switch {
			case image.base64Data != "":
				source.Type = "base64"
				source.MediaType = image.mediaType
				source.Data = image.base64Data
			case image.url != "":
				source.Type = "url"
				source.URL = image.url
			default:
				return nil, fmt.Errorf("anthropic image content missing source")
			}
			out = append(out, anthropicContentBlock{Type: "image", Source: source})
		default:
			return nil, fmt.Errorf("unsupported content part type %q", part.Type)
		}
	}
	return out, nil
}

func geminiMessageParts(msg Message) ([]geminiPart, error) {
	parts := messageParts(msg)
	out := make([]geminiPart, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case ContentPartText:
			out = append(out, geminiPart{Text: part.Text})
		case ContentPartImage:
			image, err := normalizeImagePart(part)
			if err != nil {
				return nil, err
			}
			if image.base64Data == "" {
				return nil, fmt.Errorf("gemini image content must be a data URL")
			}
			out = append(out, geminiPart{
				InlineData: &geminiInlineData{
					MimeType: image.mediaType,
					Data:     image.base64Data,
				},
			})
		default:
			return nil, fmt.Errorf("unsupported content part type %q", part.Type)
		}
	}
	return out, nil
}

func bedrockMessageBlocks(msg Message) ([]bedrocktypes.ContentBlock, error) {
	parts := messageParts(msg)
	out := make([]bedrocktypes.ContentBlock, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case ContentPartText:
			out = append(out, &bedrocktypes.ContentBlockMemberText{Value: part.Text})
		case ContentPartImage:
			imageBytes, mediaType, err := imageBytes(part)
			if err != nil {
				return nil, err
			}
			format, err := imageFormatFromMediaType(mediaType)
			if err != nil {
				return nil, err
			}
			out = append(out, &bedrocktypes.ContentBlockMemberImage{
				Value: bedrocktypes.ImageBlock{
					Format: bedrocktypes.ImageFormat(format),
					Source: &bedrocktypes.ImageSourceMemberBytes{Value: imageBytes},
				},
			})
		default:
			return nil, fmt.Errorf("unsupported content part type %q", part.Type)
		}
	}
	return out, nil
}

func normalizeImagePart(part ContentPart) (normalizedImagePart, error) {
	if part.Type != ContentPartImage {
		return normalizedImagePart{}, fmt.Errorf("unsupported content part type %q", part.Type)
	}
	url := strings.TrimSpace(part.ImageURL)
	if url == "" {
		return normalizedImagePart{}, fmt.Errorf("image content missing image_url")
	}
	image := normalizedImagePart{url: url, detail: strings.TrimSpace(part.Detail)}
	if !strings.HasPrefix(url, "data:") {
		return image, nil
	}
	mediaType, data, err := parseDataURL(url)
	if err != nil {
		return normalizedImagePart{}, err
	}
	image.mediaType = mediaType
	image.base64Data = data
	return image, nil
}

func parseDataURL(raw string) (string, string, error) {
	if !strings.HasPrefix(raw, "data:") {
		return "", "", fmt.Errorf("image data URL must start with data:")
	}
	body := strings.TrimPrefix(raw, "data:")
	header, data, ok := strings.Cut(body, ",")
	if !ok {
		return "", "", fmt.Errorf("image data URL missing comma separator")
	}
	if !strings.HasSuffix(header, ";base64") {
		return "", "", fmt.Errorf("image data URL must be base64 encoded")
	}
	mediaType := strings.TrimSuffix(header, ";base64")
	if mediaType == "" {
		return "", "", fmt.Errorf("image data URL missing media type")
	}
	return mediaType, data, nil
}

func imageBytes(part ContentPart) ([]byte, string, error) {
	image, err := normalizeImagePart(part)
	if err != nil {
		return nil, "", err
	}
	if image.base64Data == "" {
		return nil, "", fmt.Errorf("image content must be a data URL")
	}
	decoded, err := base64.StdEncoding.DecodeString(image.base64Data)
	if err != nil {
		return nil, "", fmt.Errorf("decode image data URL: %w", err)
	}
	return decoded, image.mediaType, nil
}

func imageFormatFromMediaType(mediaType string) (string, error) {
	switch mediaType {
	case "image/png":
		return "png", nil
	case "image/jpeg":
		return "jpeg", nil
	case "image/gif":
		return "gif", nil
	case "image/webp":
		return "webp", nil
	default:
		return "", fmt.Errorf("unsupported image media type %q", mediaType)
	}
}

func imageDetail(detail string) string {
	if strings.TrimSpace(detail) == "" {
		return "auto"
	}
	return detail
}
