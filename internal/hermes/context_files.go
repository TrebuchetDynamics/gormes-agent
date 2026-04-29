package hermes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	contextFilesDefaultMaxChars = 20000
	contextFilesHeadRatio       = 0.65
	contextFilesTailRatio       = 0.35
)

// ContextFilesOptions controls pure-Go Hermes-compatible context file discovery.
// Tests and callers should pass explicit temp/profile directories; the helper
// does not read live provider credentials, channels, or Python Hermes runtime
// services.
type ContextFilesOptions struct {
	CWD        string
	ProfileDir string
	SkipSoul   bool
	MaxChars   int
}

// ContextFilesReport records deterministic evidence for loaded, missing,
// blocked, skipped, or truncated context sources.
type ContextFilesReport struct {
	Soul    ContextFileEvidence
	Project ContextFileEvidence
}

// ContextFileEvidence describes one context source considered for prompt input.
type ContextFileEvidence struct {
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

var contextThreatPatterns = []struct {
	re *regexp.Regexp
	id string
}{
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

var contextInvisibleChars = []rune{'\u200b', '\u200c', '\u200d', '\u2060', '\ufeff', '\u202a', '\u202b', '\u202c', '\u202d', '\u202e'}

// BuildContextFilesPrompt discovers and renders Hermes-compatible project
// context files. Project context precedence is first-match-wins; SOUL.md from
// the profile remains independent unless SkipSoul is set.
func BuildContextFilesPrompt(opts ContextFilesOptions) (string, ContextFilesReport) {
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = contextFilesDefaultMaxChars
	}
	cwd := opts.CWD
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	cwd, _ = filepath.Abs(cwd)

	var report ContextFilesReport
	sections := make([]string, 0, 2)
	if project, ev := loadProjectContext(cwd, maxChars); project != "" {
		sections = append(sections, project)
		report.Project = ev
	} else {
		report.Project = ev
	}

	if opts.SkipSoul {
		report.Soul = ContextFileEvidence{Source: "SOUL.md", Skipped: true}
	} else if soul, ev := loadSoulContext(opts.ProfileDir, maxChars); soul != "" {
		sections = append(sections, soul)
		report.Soul = ev
	} else {
		report.Soul = ev
	}

	if len(sections) == 0 {
		return "", report
	}
	return "# Project Context\n\nThe following project context files have been loaded and should be followed:\n\n" + strings.Join(sections, "\n"), report
}

func loadSoulContext(profileDir string, maxChars int) (string, ContextFileEvidence) {
	ev := ContextFileEvidence{Source: "SOUL.md"}
	if profileDir == "" {
		ev.Missing = true
		return "", ev
	}
	path := filepath.Join(profileDir, "SOUL.md")
	content, ok := readContextFile(path, &ev)
	if !ok {
		return "", ev
	}
	content, ev = scanContextContent(content, "SOUL.md", ev)
	content, ev = truncateContextContent(content, "SOUL.md", maxChars, ev)
	return content, ev
}

func loadProjectContext(cwd string, maxChars int) (string, ContextFileEvidence) {
	if content, ev := loadHermesMD(cwd, maxChars); content != "" || ev.Loaded || ev.Blocked || ev.Error != "" {
		return content, ev
	}
	if content, ev := loadNamedCWDContext(cwd, []string{"AGENTS.md", "agents.md"}, "AGENTS.md", maxChars); content != "" || ev.Loaded || ev.Blocked || ev.Error != "" {
		return content, ev
	}
	if content, ev := loadNamedCWDContext(cwd, []string{"CLAUDE.md", "claude.md"}, "CLAUDE.md", maxChars); content != "" || ev.Loaded || ev.Blocked || ev.Error != "" {
		return content, ev
	}
	if content, ev := loadCursorContext(cwd, maxChars); content != "" || ev.Loaded || ev.Blocked || ev.Error != "" {
		return content, ev
	}
	return "", ContextFileEvidence{Missing: true}
}

func loadHermesMD(cwd string, maxChars int) (string, ContextFileEvidence) {
	path := findHermesMD(cwd)
	if path == "" {
		return "", ContextFileEvidence{Missing: true}
	}
	rel := filepath.Base(path)
	if r, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(r, "..") {
		rel = filepath.ToSlash(r)
	}
	ev := ContextFileEvidence{Source: rel, Path: path}
	content, ok := readContextFile(path, &ev)
	if !ok {
		return "", ev
	}
	content = stripYAMLFrontmatter(content)
	content, ev = scanContextContent(content, rel, ev)
	rendered := fmt.Sprintf("## %s\n\n%s", rel, content)
	rendered, ev = truncateContextContent(rendered, ".hermes.md", maxChars, ev)
	return rendered, ev
}

func loadNamedCWDContext(cwd string, names []string, truncateName string, maxChars int) (string, ContextFileEvidence) {
	for _, name := range names {
		path := filepath.Join(cwd, name)
		if fileInfo(path) == nil {
			continue
		}
		ev := ContextFileEvidence{Source: name, Path: path}
		content, ok := readContextFile(path, &ev)
		if !ok {
			return "", ev
		}
		content, ev = scanContextContent(content, name, ev)
		rendered := fmt.Sprintf("## %s\n\n%s", name, content)
		rendered, ev = truncateContextContent(rendered, truncateName, maxChars, ev)
		return rendered, ev
	}
	return "", ContextFileEvidence{Missing: true}
}

func loadCursorContext(cwd string, maxChars int) (string, ContextFileEvidence) {
	parts := []string{}
	ev := ContextFileEvidence{Source: ".cursorrules"}
	cursorPath := filepath.Join(cwd, ".cursorrules")
	if fileInfo(cursorPath) != nil {
		ev.Path = cursorPath
		content, ok := readContextFile(cursorPath, &ev)
		if !ok {
			return "", ev
		}
		content, ev = scanContextContent(content, ".cursorrules", ev)
		parts = append(parts, fmt.Sprintf("## .cursorrules\n\n%s\n", content))
	}
	rulesDir := filepath.Join(cwd, ".cursor", "rules")
	matches, _ := filepath.Glob(filepath.Join(rulesDir, "*.mdc"))
	sort.Strings(matches)
	for _, path := range matches {
		name := ".cursor/rules/" + filepath.Base(path)
		if ev.Path == "" {
			ev.Path = path
		}
		content, ok := readContextFile(path, &ev)
		if !ok {
			return "", ev
		}
		content, ev = scanContextContent(content, name, ev)
		parts = append(parts, fmt.Sprintf("## %s\n\n%s\n", name, content))
	}
	if len(parts) == 0 {
		return "", ContextFileEvidence{Missing: true}
	}
	ev.Loaded = true
	rendered := strings.TrimSpace(strings.Join(parts, "\n"))
	rendered, ev = truncateContextContent(rendered, ".cursorrules", maxChars, ev)
	return rendered, ev
}

func readContextFile(path string, ev *ContextFileEvidence) (string, bool) {
	info := fileInfo(path)
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

func scanContextContent(content, filename string, ev ContextFileEvidence) (string, ContextFileEvidence) {
	findings := []string{}
	for _, char := range contextInvisibleChars {
		if strings.ContainsRune(content, char) {
			findings = append(findings, fmt.Sprintf("invisible unicode U+%04X", char))
		}
	}
	for _, pattern := range contextThreatPatterns {
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

func truncateContextContent(content, filename string, maxChars int, ev ContextFileEvidence) (string, ContextFileEvidence) {
	if maxChars <= 0 {
		maxChars = contextFilesDefaultMaxChars
	}
	if utf8.RuneCountInString(content) <= maxChars {
		ev.RenderedLength = utf8.RuneCountInString(content)
		return content, ev
	}
	runes := []rune(content)
	headChars := int(float64(maxChars) * contextFilesHeadRatio)
	tailChars := int(float64(maxChars) * contextFilesTailRatio)
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

func findHermesMD(cwd string) string {
	stopAt := findGitRoot(cwd)
	current := cwd
	for {
		for _, name := range []string{".hermes.md", "HERMES.md"} {
			candidate := filepath.Join(current, name)
			if info := fileInfo(candidate); info != nil && !info.IsDir() {
				return candidate
			}
		}
		if stopAt != "" && samePath(current, stopAt) {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func findGitRoot(start string) string {
	current := start
	for {
		if info := fileInfo(filepath.Join(current, ".git")); info != nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func stripYAMLFrontmatter(content string) string {
	if strings.HasPrefix(content, "---") {
		end := strings.Index(content[3:], "\n---")
		if end != -1 {
			body := strings.TrimLeft(content[3+end+4:], "\n")
			if body != "" {
				return body
			}
		}
	}
	return content
}

func fileInfo(path string) os.FileInfo {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return info
}

func samePath(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	return a == b
}
