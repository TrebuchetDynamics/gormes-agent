package telegram

import "strings"

func telegramMergeCaption(existing, next string) string {
	caption := strings.TrimSpace(next)
	if caption == "" {
		return existing
	}
	if strings.TrimSpace(existing) == "" {
		return caption
	}
	for _, part := range strings.Split(existing, "\n\n") {
		if strings.TrimSpace(part) == caption {
			return existing
		}
	}
	return existing + "\n\n" + caption
}
