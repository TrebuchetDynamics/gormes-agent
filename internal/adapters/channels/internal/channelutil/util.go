// Package channelutil provides reusable utility functions shared across
// channel adapter implementations. Extraction target: toSet and NormalizedPolicy
// are each duplicated in 3+ channel subpackages.
package channelutil

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/trace"
)

// ToSet converts a string slice to a set map, trimming whitespace and
// skipping empty entries. Behavior is identical to the duplicated helper
// found in wecom, weixin, qqbot, feishu, and homeassistant.
func ToSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

// BoolSet converts a string slice to a boolean membership set, trimming
// whitespace and skipping empty entries. It is the bool-valued counterpart to
// ToSet for channel policies that use map[string]bool.
func BoolSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[value] = true
	}
	return out
}

// BoolSetsIntersect reports whether two boolean membership sets share at least
// one key.
func BoolSetsIntersect(a, b map[string]bool) bool {
	for value := range a {
		if b[value] {
			return true
		}
	}
	return false
}

// NormalizedPolicy lowercases and trims a policy string. Returns "open" when
// policy is empty. Behavior matches the duplicated local helper in wecom,
// qqbot, and feishu.
func NormalizedPolicy(policy string) string {
	policy = strings.TrimSpace(strings.ToLower(policy))
	if policy == "" {
		return "open"
	}
	return policy
}

// AllowedByPolicy evaluates the shared open/disabled/allowlist channel policy.
// Unknown policies stay permissive for compatibility with the local helpers it
// replaces. Allowlist checks trim the candidate value before lookup.
func AllowedByPolicy(policy string, allowed map[string]struct{}, value string) bool {
	switch NormalizedPolicy(policy) {
	case "disabled":
		return false
	case "allowlist":
		_, ok := allowed[strings.TrimSpace(value)]
		return ok
	default:
		return true
	}
}

// StripLeadingMentions removes leading @-mention tokens from text. Used by
// qqbot and feishu to normalize group mentions before parsing.
func StripLeadingMentions(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	for len(fields) > 0 {
		if !strings.HasPrefix(fields[0], "@") {
			break
		}
		fields = fields[1:]
	}
	return strings.TrimSpace(strings.Join(fields, " "))
}

// FirstNonEmpty returns the first non-empty (after trimming whitespace)
// string from values, with trailing/leading whitespace removed. Returns ""
// when all values are empty. Used by 8+ channel subpackages.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// CompactStrings trims values and returns non-empty entries in their original
// order. It intentionally preserves duplicates for configuration lists where
// later validation may care about repeated values.
func CompactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// SplitCommaList splits a comma-separated adapter configuration list, trims
// entries, drops empty values, and preserves duplicate entries.
func SplitCommaList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return CompactStrings(strings.Split(raw, ","))
}

// UniqueCommaList splits a comma-separated adapter configuration list, trims
// entries, drops empty values, and keeps the first occurrence of each value.
func UniqueCommaList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return UniqueStrings(strings.Split(raw, ","))
}

// UniqueStrings trims values and returns the first occurrence of each non-empty
// entry in original order.
func UniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

// UniqueSortedStrings trims, deduplicates, and sorts non-empty values.
func UniqueSortedStrings(values []string) []string {
	out := UniqueStrings(values)
	sort.Strings(out)
	return out
}

// UniqueLowerSortedStrings trims, lowercases, deduplicates, and sorts
// non-empty values. It is useful for case-insensitive channel/provider lists
// that need stable API output.
func UniqueLowerSortedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// ContainsString reports whether values contains want exactly.
func ContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ContainsEqualFold reports whether values contains want using strings.EqualFold.
func ContainsEqualFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

// ParseBoolDefault parses common channel configuration booleans. Empty or
// unrecognized values return def so env/config overlays can preserve their
// existing default behavior.
func ParseBoolDefault(raw string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return def
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}

var commonDocumentMediaTypes = map[string]string{
	".cfg":  "text/plain",
	".csv":  "text/csv",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".ini":  "text/plain",
	".json": "application/json",
	".log":  "text/plain",
	".md":   "text/markdown",
	".pdf":  "application/pdf",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".toml": "application/toml",
	".txt":  "text/plain",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xml":  "application/xml",
	".yaml": "application/yaml",
	".yml":  "application/yaml",
	".zip":  "application/zip",
}

var commonMIMEExtensionFallbacks = map[string]string{
	"application/json":   ".json",
	"application/pdf":    ".pdf",
	"application/toml":   ".toml",
	"application/xml":    ".xml",
	"application/yaml":   ".yaml",
	"application/zip":    ".zip",
	"text/csv":           ".csv",
	"text/markdown":      ".md",
	"text/plain":         ".txt",
	"x-application/yaml": ".yaml",
}

// DocumentMediaTypeForExtension returns the shared channel document media type
// for a normalized extension set used by Telegram and Discord attachments.
func DocumentMediaTypeForExtension(ext string) (string, bool) {
	mediaType, ok := commonDocumentMediaTypes[strings.ToLower(strings.TrimSpace(ext))]
	return mediaType, ok
}

// DocumentExtensions returns the sorted shared channel document extensions.
func DocumentExtensions() []string {
	extensions := make([]string, 0, len(commonDocumentMediaTypes))
	for ext := range commonDocumentMediaTypes {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return extensions
}

// MIMEExtensionFallback returns the shared extension fallback for mediaType.
func MIMEExtensionFallback(mediaType string) string {
	return commonMIMEExtensionFallbacks[CleanMediaType(mediaType)]
}

// CleanMediaType strips MIME parameters and normalizes a media type.
func CleanMediaType(mediaType string) string {
	if mediaType = strings.TrimSpace(mediaType); mediaType == "" {
		return ""
	}
	if semi := strings.Index(mediaType, ";"); semi >= 0 {
		mediaType = mediaType[:semi]
	}
	return strings.ToLower(strings.TrimSpace(mediaType))
}

// ImageExtensionForMediaType returns the preferred image file extension for a
// normalized media type, defaulting to .jpg for unknown image payloads.
func ImageExtensionForMediaType(mediaType string) string {
	switch CleanMediaType(mediaType) {
	case "image/gif":
		return ".gif"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

// ImageMediaTypeForExtension returns the preferred media type for supported
// channel image extensions, defaulting to image/jpeg.
func ImageMediaTypeForExtension(ext string) string {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".gif":
		return "image/gif"
	case ".jpeg", ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// ImageExtensionSupported reports whether ext is a supported channel image
// extension.
func ImageExtensionSupported(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".gif", ".jpeg", ".jpg", ".png", ".webp":
		return true
	default:
		return false
	}
}

// SafeFileName returns a bounded basename safe for local cache paths and
// evidence text.
func SafeFileName(fileName string) string {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	var out strings.Builder
	for _, r := range fileName {
		switch {
		case r == 0 || r < 32 || r == 127 || r == '/' || r == '\\':
			out.WriteByte('_')
		default:
			out.WriteRune(r)
		}
	}
	cleaned := strings.Trim(out.String(), " .")
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return ""
	}
	if len(cleaned) <= 160 {
		return cleaned
	}
	ext := filepath.Ext(cleaned)
	stem := strings.TrimSuffix(cleaned, ext)
	if len(ext) > 32 {
		ext = ""
	}
	if len(stem) > 128 {
		stem = stem[:128]
	}
	return stem + ext
}

// SafeTokenDefault returns a bounded token safe for local cache directory and
// generated file names, using fallback when all characters are removed.
func SafeTokenDefault(s, fallback string) string {
	s = strings.TrimSpace(s)
	var out strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			out.WriteRune(r)
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}
	cleaned := strings.Trim(out.String(), "._-")
	if cleaned == "" {
		return fallback
	}
	if len(cleaned) > 64 {
		return cleaned[:64]
	}
	return cleaned
}

// FormatToolTrace joins SoulEntry texts and formats them as a tool trace block.
// Behavior is identical to the duplicated function found in discord/legacy/render.go
// and slack/render.go. Extracted to share across channel renderers.
func FormatToolTrace(events []kernel.SoulEntry) string {
	texts := make([]string, 0, len(events))
	for _, event := range events {
		texts = append(texts, event.Text)
	}
	return trace.FormatBlock(texts)
}

// TruncateRunes truncates a string to at most max runes and appends "..."
// when truncation occurs. Returns empty when max is <= 0. Behavior is
// identical to the duplicated function found in slack/approval_buttons.go
// and telegram/callbacks/approval.go.
func TruncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
