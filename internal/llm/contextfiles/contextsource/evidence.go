package contextsource

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	DefaultMaxChars = 20000
	headRatio       = 0.65
	tailRatio       = 0.35
)

// Evidence describes one context source considered for prompt input.
type Evidence struct {
	Source         string
	Path           string
	Loaded         bool
	Skipped        bool
	Missing        bool
	Blocked        bool
	Truncated      bool
	Findings       []string
	OriginalLength int
	RenderedLength int
	Error          string
}

type threatPattern struct {
	re *regexp.Regexp
	id string
}

var threatPatterns = []threatPattern{
	{regexp.MustCompile(`(?i)ignore\s+(previous|all|above|prior)\s+instructions`), "prompt_injection"},
	{regexp.MustCompile(`(?i)do\s+not\s+tell\s+the\s+user`), "deception_hide"},
	{regexp.MustCompile(`(?i)system\s+prompt\s+override`), "sys_prompt_override"},
	{regexp.MustCompile(`(?i)disregard\s+(your|all|any)\s+(instructions|rules|guidelines)`), "disregard_rules"},
	{regexp.MustCompile(`(?i)act\s+as\s+(if|though)\s+you\s+(have\s+no|don't\s+have)\s+(restrictions|limits|rules)`), "bypass_restrictions"},
	{regexp.MustCompile(`(?is)<!--[^>]*(?:ignore|override|system|secret|hidden)[^>]*-->`), "html_comment_injection"},
	{regexp.MustCompile(`(?is)<\s*div\s+style\s*=\s*["'][\s\S]*?display\s*:\s*none`), "hidden_div"},
	{regexp.MustCompile(`(?i)translate\s+.*\s+into\s+.*\s+and\s+(execute|run|eval)`), "translate_execute"},
	{regexp.MustCompile(`(?i)curl\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`), "exfil_curl"},
	{regexp.MustCompile(`(?i)cat\s+[^\n]*(\.env|credentials|\.netrc|\.pgpass)`), "read_secrets"},
}

var invisibleChars = []rune{'\u200b', '\u200c', '\u200d', '\u2060', '\ufeff', '\u202a', '\u202b', '\u202c', '\u202d', '\u202e'}

// ReadFile reads a non-empty context file and records deterministic evidence.
func ReadFile(path string, ev *Evidence) (string, bool) {
	info := FileInfo(path)
	if info == nil {
		ev.Missing = true
		return "", false
	}
	if info.IsDir() {
		ev.Missing = true
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		ev.Error = err.Error()
		return "", false
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		ev.Missing = true
		return "", false
	}
	ev.Loaded = true
	ev.OriginalLength = utf8.RuneCountInString(content)
	return content, true
}

// ScanContent blocks known prompt-injection and invisible-character patterns.
func ScanContent(content, filename string, ev Evidence) (string, Evidence) {
	findings := []string{}
	for _, char := range invisibleChars {
		if strings.ContainsRune(content, char) {
			findings = append(findings, fmt.Sprintf("invisible unicode U+%04X", char))
		}
	}
	for _, pattern := range threatPatterns {
		if pattern.re.MatchString(content) {
			findings = append(findings, pattern.id)
		}
	}
	if len(findings) == 0 {
		return content, ev
	}
	ev.Blocked = true
	ev.Findings = append(ev.Findings, findings...)
	return fmt.Sprintf("[BLOCKED: %s contained potential prompt injection (%s). Content not loaded.]", filename, strings.Join(findings, ", ")), ev
}

// TruncateContent keeps deterministic head/tail excerpts for large context files.
func TruncateContent(content, filename string, maxChars int, ev Evidence) (string, Evidence) {
	if maxChars <= 0 {
		maxChars = DefaultMaxChars
	}
	if utf8.RuneCountInString(content) <= maxChars {
		ev.RenderedLength = utf8.RuneCountInString(content)
		return content, ev
	}
	runes := []rune(content)
	headChars := int(float64(maxChars) * headRatio)
	tailChars := int(float64(maxChars) * tailRatio)
	if headChars+tailChars > len(runes) {
		headChars = maxChars / 2
		tailChars = maxChars - headChars
	}
	marker := fmt.Sprintf("\n\n[...truncated %s: kept %d+%d of %d chars. Use file tools to read the full file.]\n\n", filename, headChars, tailChars, len(runes))
	truncated := string(runes[:headChars]) + marker + string(runes[len(runes)-tailChars:])
	ev.Truncated = true
	ev.RenderedLength = utf8.RuneCountInString(truncated)
	if ev.OriginalLength == 0 {
		ev.OriginalLength = len(runes)
	}
	return truncated, ev
}

func FileInfo(path string) os.FileInfo {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return info
}
