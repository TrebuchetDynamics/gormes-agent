package streamdiag

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func RetryClassification(err error) llm.ProviderErrorClassification {
	classification := llm.ClassifyProviderError(err)
	if classification.Class == llm.ClassRetryable {
		return classification
	}
	return llm.ProviderErrorClassification{
		Kind:      llm.ProviderErrorRetryable,
		Class:     llm.ClassRetryable,
		Retryable: true,
	}
}

func ErrorType(err error) string {
	if err == nil {
		return "unknown"
	}
	var httpErr *llm.HTTPError
	if errors.As(err, &httpErr) {
		return "HTTPError"
	}
	return ConcreteErrorType(err)
}

func ConcreteErrorType(err error) string {
	if err == nil {
		return "unknown"
	}
	name := strings.TrimPrefix(fmt.Sprintf("%T", err), "*")
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	if name == "" {
		return "unknown"
	}
	return name
}

func ErrorChain(err error) string {
	if err == nil {
		return "unknown"
	}
	parts := make([]string, 0, 4)
	for current := err; current != nil && len(parts) < 4; current = errors.Unwrap(current) {
		parts = append(parts, fmt.Sprintf("%s(%s)", ConcreteErrorType(current), CompactText(current.Error(), 200)))
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, " <- ")
}

func FormatHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(headers))
	for rawName, rawValue := range headers {
		name := strings.ToLower(strings.TrimSpace(rawName))
		value := CompactText(rawValue, 120)
		if name == "" || value == "" {
			continue
		}
		pairs = append(pairs, name+"="+value)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, " ")
}

func CompactText(text string, maxLen int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if maxLen > 3 && len(text) > maxLen {
		return text[:maxLen-3] + "..."
	}
	return text
}

func ErrorText(err error) string {
	if err == nil {
		return "unknown stream drop"
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return ErrorType(err)
	}
	return text
}
