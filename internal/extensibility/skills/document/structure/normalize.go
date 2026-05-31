package structure

import "strings"

// NormalizeBytes applies the byte-to-document normalization shared by parsing
// and frontmatter validation before they inspect SKILL.md delimiters.
func NormalizeBytes(raw []byte) string {
	content := string(raw)
	content = strings.TrimPrefix(content, "\uFEFF")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return content
}

// SplitLines keeps delimiter scanning consistent across document consumers.
func SplitLines(content string) []string {
	return strings.Split(content, "\n")
}
