package providererrors

import (
	"encoding/json"
	"strings"
)

func SanitizeText(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") || strings.Contains(lower, "<svg") {
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			prefix := strings.TrimSpace(trimmed[:idx])
			if prefix != "" && !strings.ContainsAny(prefix, "<>\n\r") {
				return prefix + ": provider returned HTML error body"
			}
		}
		return "provider returned HTML error body"
	}
	if prefix, body, ok := splitProviderErrorBody(trimmed); ok && looksLikeJSON(body) {
		return sanitizeProviderJSONError(prefix, body)
	}
	if looksLikeJSON(trimmed) {
		return sanitizeProviderJSONError("", trimmed)
	}
	return trimmed
}

func splitProviderErrorBody(s string) (prefix, body string, ok bool) {
	prefix, body, ok = strings.Cut(s, ":")
	if !ok {
		return "", "", false
	}
	prefix = strings.TrimSpace(prefix)
	body = strings.TrimSpace(body)
	return prefix, body, prefix != "" && body != ""
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func sanitizeProviderJSONError(prefix, body string) string {
	message, code, ok := providerJSONErrorSummary(body)
	prefix = strings.TrimSpace(prefix)
	if isProviderAuthPrefix(prefix) {
		if isGenericProviderAuthMessage(message) || message == "" {
			return prefix + ": provider authentication failed"
		}
		return prefix + ": " + message
	}
	if prefix == "" {
		switch {
		case message != "":
			return message
		case code != "":
			return code
		default:
			return "provider returned JSON error body"
		}
	}
	if ok {
		switch {
		case message != "":
			return prefix + ": " + message
		case code != "":
			return prefix + ": " + code
		}
	}
	return prefix + ": provider returned JSON error body"
}

func providerJSONErrorSummary(body string) (message, code string, ok bool) {
	var decoded any
	if json.Unmarshal([]byte(strings.TrimSpace(body)), &decoded) != nil {
		return "", "", false
	}
	message, code = providerJSONErrorSummaryValue(decoded)
	return message, code, true
}

func providerJSONErrorSummaryValue(v any) (message, code string) {
	obj, ok := v.(map[string]any)
	if !ok {
		return "", ""
	}
	if errObj, ok := obj["error"].(map[string]any); ok {
		message = providerJSONStringField(errObj["message"])
		code = firstProviderJSONStringField(errObj, "code", "type")
		return message, code
	}
	if errText := providerJSONStringField(obj["error"]); errText != "" {
		code = errText
		message = firstProviderJSONStringField(obj, "error_description", "message", "detail")
		if message == "" {
			message = errText
		}
		return message, code
	}
	message = firstProviderJSONStringField(obj, "message", "detail", "error_description")
	code = firstProviderJSONStringField(obj, "code", "error_code", "type")
	return message, code
}

func firstProviderJSONStringField(obj map[string]any, names ...string) string {
	for _, name := range names {
		if s := providerJSONStringField(obj[name]); s != "" {
			return s
		}
	}
	return ""
}

func providerJSONStringField(v any) string {
	switch x := v.(type) {
	case string:
		return strings.Join(strings.Fields(x), " ")
	default:
		return ""
	}
}

func isProviderAuthPrefix(prefix string) bool {
	switch strings.ToLower(strings.TrimSpace(prefix)) {
	case "unauthorized", "forbidden":
		return true
	default:
		return false
	}
}

func isGenericProviderAuthMessage(message string) bool {
	switch strings.ToLower(strings.TrimSpace(message)) {
	case "unauthorized", "forbidden", "authentication failed", "auth failed":
		return true
	default:
		return false
	}
}
