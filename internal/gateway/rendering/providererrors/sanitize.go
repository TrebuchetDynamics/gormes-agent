package providererrors

import (
	"encoding/json"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

func SanitizeText(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") || strings.Contains(lower, "<svg") {
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			prefix := providerErrorFieldText(trimmed[:idx])
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
	return providerErrorFieldText(trimmed)
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
		return providerErrorFieldText(x)
	default:
		return ""
	}
}

func providerErrorFieldText(value string) string {
	value = collapseRedactedProviderAssignments(redaction.RedactSecrets(value))
	return strings.Join(strings.Fields(value), " ")
}

func collapseRedactedProviderAssignments(value string) string {
	replacer := strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
	)
	return replacer.Replace(value)
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
