package credentials

import (
	"fmt"
	"strings"
	"sync"
)

const CredentialSanitizerEvidenceNonASCIIStripped = "credential_non_ascii_stripped"

type CredentialSanitizerWarning struct {
	Code            string   `json:"code"`
	Key             string   `json:"key"`
	CodePoints      []string `json:"code_points"`
	WarningOnce     bool     `json:"warning_once"`
	RedactedPreview string   `json:"redacted_preview"`
	Message         string   `json:"message"`
	Redacted        bool     `json:"redacted"`
}

type CredentialSanitizer struct {
	mu       sync.Mutex
	warned   map[string]struct{}
	recorder func(CredentialSanitizerWarning)
}

func NewCredentialSanitizer(recorder func(CredentialSanitizerWarning)) *CredentialSanitizer {
	return &CredentialSanitizer{warned: make(map[string]struct{}), recorder: recorder}
}

func (s *CredentialSanitizer) Sanitize(key, value string) string {
	key = strings.TrimSpace(key)
	if key == "" || value == "" || !isCredentialLikeKey(key) {
		return value
	}
	cleaned, codePoints, stripped := stripNonASCII(value)
	if stripped == 0 {
		return value
	}
	if s == nil {
		return cleaned
	}
	s.recordOnce(key, value, codePoints, stripped)
	return cleaned
}

func (s *CredentialSanitizer) ResetWarnings() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.warned = make(map[string]struct{})
}

func (s *CredentialSanitizer) recordOnce(key, value string, codePoints []string, stripped int) {
	warnKey := strings.ToUpper(key)
	s.mu.Lock()
	if s.warned == nil {
		s.warned = make(map[string]struct{})
	}
	if _, ok := s.warned[warnKey]; ok {
		s.mu.Unlock()
		return
	}
	s.warned[warnKey] = struct{}{}
	recorder := s.recorder
	s.mu.Unlock()
	if recorder == nil {
		return
	}
	recorder(CredentialSanitizerWarning{
		Code:            CredentialSanitizerEvidenceNonASCIIStripped,
		Key:             key,
		CodePoints:      append([]string(nil), codePoints...),
		WarningOnce:     true,
		RedactedPreview: redactedCredentialPreview(value, stripped),
		Message:         fmt.Sprintf("%s contained non-ASCII credential code points (%s); stripped automatically. If authentication fails, re-copy the key from the provider dashboard.", key, strings.Join(codePoints, ", ")),
		Redacted:        true,
	})
}

func isCredentialLikeKey(key string) bool {
	upper := strings.ToUpper(strings.TrimSpace(key))
	switch upper {
	case "API_KEY", "KEY", "SECRET", "TOKEN", "AUTH_TOKEN", "BOT_TOKEN":
		return true
	}
	return strings.HasSuffix(upper, "_API_KEY") ||
		strings.HasSuffix(upper, "_KEY") ||
		strings.HasSuffix(upper, "_SECRET") ||
		strings.HasSuffix(upper, "_TOKEN")
}

func stripNonASCII(value string) (string, []string, int) {
	var b strings.Builder
	b.Grow(len(value))
	codePoints := make([]string, 0)
	seen := make(map[string]struct{})
	stripped := 0
	for _, r := range value {
		if r > 127 {
			cp := fmt.Sprintf("U+%04X", r)
			if _, ok := seen[cp]; !ok {
				seen[cp] = struct{}{}
				codePoints = append(codePoints, cp)
			}
			stripped++
			continue
		}
		b.WriteRune(r)
	}
	return b.String(), codePoints, stripped
}

func redactedCredentialPreview(value string, stripped int) string {
	runes := 0
	for range value {
		runes++
	}
	return fmt.Sprintf("[redacted length=%d stripped=%d]", runes, stripped)
}

var defaultCredentialSanitizer = NewCredentialSanitizer(nil)

func SanitizeCredentialValue(key, value string) string {
	return defaultCredentialSanitizer.Sanitize(key, value)
}

func sanitizeCredentialValue(key, value string) string {
	return SanitizeCredentialValue(key, value)
}

func ResetDefaultCredentialSanitizerWarnings() {
	defaultCredentialSanitizer.ResetWarnings()
}
