package llm

import (
	"fmt"
	"unicode/utf8"
)

func compressionContentLength(content any) int {
	switch v := content.(type) {
	case nil:
		return 0
	case string:
		return utf8.RuneCountInString(v)
	case map[string]any:
		return compressionContentBlockLength(v)
	case []any:
		total := 0
		for _, item := range v {
			total += compressionContentListItemLength(item)
		}
		return total
	default:
		return utf8.RuneCountInString(fmt.Sprint(v))
	}
}

func compressionContentListItemLength(item any) int {
	switch v := item.(type) {
	case nil:
		return 0
	case string:
		return utf8.RuneCountInString(v)
	case map[string]any:
		return compressionContentBlockLength(v)
	default:
		return utf8.RuneCountInString(fmt.Sprint(v))
	}
}

func compressionContentBlockLength(block map[string]any) int {
	text, ok := block["text"].(string)
	if !ok {
		return 0
	}
	return utf8.RuneCountInString(text)
}

// compressionContentBudgetLength returns the effective char-length of a
// message's content for tail-cut budget accounting. Plain strings count as
// rune length; multimodal lists charge a flat imageCharEquivalent per image
// part (image_url, input_image, image) and the rune length of any text
// fields. Raw base64 payloads inside image_url dicts are intentionally
// ignored — only the presence of an image part matters.
func compressionContentBudgetLength(content any) int {
	switch v := content.(type) {
	case nil:
		return 0
	case string:
		return utf8.RuneCountInString(v)
	case map[string]any:
		return compressionContentBudgetBlockLength(v)
	case []any:
		total := 0
		for _, item := range v {
			total += compressionContentBudgetItemLength(item)
		}
		return total
	default:
		return utf8.RuneCountInString(fmt.Sprint(v))
	}
}

func compressionContentBudgetItemLength(item any) int {
	switch v := item.(type) {
	case nil:
		return 0
	case string:
		return utf8.RuneCountInString(v)
	case map[string]any:
		return compressionContentBudgetBlockLength(v)
	default:
		return utf8.RuneCountInString(fmt.Sprint(v))
	}
}

func compressionContentBudgetBlockLength(block map[string]any) int {
	switch block["type"] {
	case "image_url", "input_image", "image":
		return imageCharEquivalent
	}
	if text, ok := block["text"].(string); ok {
		return utf8.RuneCountInString(text)
	}
	return 0
}
