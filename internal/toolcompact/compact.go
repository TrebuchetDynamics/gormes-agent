package toolcompact

import (
	"fmt"
	"regexp"
	"strings"
)

type Mode string

const (
	ModeOff  Mode = "off"
	ModeAuto Mode = "auto"
)

const (
	ReducerGoTest           = "go_test"
	ReducerBuildDiagnostics = "build_diagnostics"
	ReducerGitStatus        = "git_status"
	ReducerHeadTail         = "head_tail"
)

const (
	EvidenceRawRequested       = "raw_requested"
	EvidenceUnderThreshold     = "under_threshold"
	EvidenceFailingTestsKept   = "failing_tests_kept"
	EvidencePassingPkgsCounted = "passing_packages_counted"
	EvidenceDiagnosticsKept    = "diagnostics_kept"
	EvidenceGitStatusCounted   = "git_status_counted"
	EvidenceTailKept           = "tail_kept"
	EvidenceHeadKept           = "head_kept"
)

type Request struct {
	ToolName     string
	Command      string
	Stream       string
	Text         string
	ExitCode     int
	RawRequested bool
}

type Config struct {
	Mode           Mode
	ThresholdBytes int
	HeadLines      int
	TailLines      int
}

type Result struct {
	Text           string
	Applied        bool
	Reducer        string
	OriginalBytes  int
	CompactedBytes int
	Evidence       []string
}

func Compact(req Request, cfg Config) Result {
	cfg = cfg.withDefaults()
	result := Result{
		Text:           req.Text,
		OriginalBytes:  len(req.Text),
		CompactedBytes: len(req.Text),
	}
	if req.RawRequested {
		result.Evidence = append(result.Evidence, EvidenceRawRequested)
		return result
	}
	if cfg.Mode != ModeAuto {
		return result
	}
	if len(req.Text) < cfg.ThresholdBytes {
		result.Evidence = append(result.Evidence, EvidenceUnderThreshold)
		return result
	}

	if looksLikeGoTest(req.Command, req.Text) {
		result.Text, result.Evidence = compactGoTest(req.Text)
		result.Reducer = ReducerGoTest
	} else if looksLikeBuildDiagnostics(req.Command, req.Text) {
		result.Text, result.Evidence = compactBuildDiagnostics(req.Text)
		result.Reducer = ReducerBuildDiagnostics
	} else if looksLikeGitStatus(req.Command, req.Text) {
		result.Text, result.Evidence = compactGitStatus(req.Text)
		result.Reducer = ReducerGitStatus
	} else {
		result.Text, result.Evidence = compactHeadTail(req.Text, cfg)
		result.Reducer = ReducerHeadTail
	}
	result.Applied = result.Text != req.Text
	result.CompactedBytes = len(result.Text)
	if !result.Applied {
		result.Reducer = ""
	}
	return result
}

func (cfg Config) withDefaults() Config {
	if cfg.ThresholdBytes <= 0 {
		cfg.ThresholdBytes = 8 * 1024
	}
	if cfg.HeadLines <= 0 {
		cfg.HeadLines = 20
	}
	if cfg.TailLines <= 0 {
		cfg.TailLines = 20
	}
	return cfg
}

var (
	goTestOKPackage     = regexp.MustCompile(`(?m)^(ok|\?)\s+\S+`)
	goTestFailPackage   = regexp.MustCompile(`(?m)^FAIL\s+\S+`)
	goTestFailTest      = regexp.MustCompile(`(?m)^\s*--- FAIL:\s+`)
	goDiagnosticLine    = regexp.MustCompile(`\b[-_A-Za-z0-9./]+\.go:\d+\b`)
	buildDiagnosticLine = regexp.MustCompile(`\b[-_A-Za-z0-9./]+\.(go|ts|tsx|js|jsx|c|cc|cpp|h|hpp|rs|java|kt|py|sh):\d+(:\d+)?:`)
)

func looksLikeGoTest(command, text string) bool {
	lowerCommand := strings.ToLower(command)
	commandSuggests := strings.Contains(lowerCommand, "go test")
	outputConfirms := goTestOKPackage.MatchString(text) ||
		goTestFailPackage.MatchString(text) ||
		goTestFailTest.MatchString(text) ||
		strings.Contains(text, "\nFAIL\n")
	return outputConfirms && (commandSuggests || goTestFailTest.MatchString(text))
}

func compactGoTest(text string) (string, []string) {
	lines := splitLines(text)
	passingPackages := 0
	var kept []string
	seen := map[string]bool{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case goTestOKPackage.MatchString(line):
			passingPackages++
		case goTestFailTest.MatchString(line),
			goTestFailPackage.MatchString(line),
			trimmed == "FAIL",
			goDiagnosticLine.MatchString(line),
			strings.Contains(strings.ToLower(line), "panic:"):
			if !seen[line] {
				kept = append(kept, line)
				seen[line] = true
			}
		}
	}
	if len(kept) == 0 {
		if passingPackages > 0 {
			var b strings.Builder
			fmt.Fprintf(&b, "[tool output compacted: reducer=%s]\n", ReducerGoTest)
			fmt.Fprintf(&b, "passing packages: %d\n", passingPackages)
			b.WriteString("go test completed without failure diagnostics in compacted output.")
			return b.String(), []string{EvidencePassingPkgsCounted}
		}
		return compactHeadTail(text, Config{HeadLines: 20, TailLines: 20})
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[tool output compacted: reducer=%s]\n", ReducerGoTest)
	fmt.Fprintf(&b, "passing packages: %d\n", passingPackages)
	b.WriteString("actionable diagnostics:\n")
	for _, line := range kept {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	out := strings.TrimRight(b.String(), "\n")
	return out, []string{EvidencePassingPkgsCounted, EvidenceFailingTestsKept}
}

func looksLikeBuildDiagnostics(command, text string) bool {
	lowerCommand := strings.ToLower(command)
	commandSuggests := strings.Contains(lowerCommand, "build") ||
		strings.Contains(lowerCommand, "compile") ||
		strings.Contains(lowerCommand, "test")
	outputConfirms := buildDiagnosticLine.MatchString(text) ||
		strings.Contains(strings.ToLower(text), "undefined:") ||
		strings.Contains(strings.ToLower(text), "error:")
	return outputConfirms && commandSuggests
}

func compactBuildDiagnostics(text string) (string, []string) {
	lines := splitLines(text)
	var kept []string
	seen := map[string]bool{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(trimmed, "# "),
			buildDiagnosticLine.MatchString(line),
			strings.HasPrefix(trimmed, "FAIL\t"),
			strings.Contains(lower, "undefined:"),
			strings.Contains(lower, "error:"),
			strings.Contains(lower, "panic:"):
			if !seen[line] {
				kept = append(kept, line)
				seen[line] = true
			}
		}
	}
	if len(kept) == 0 {
		return compactHeadTail(text, Config{HeadLines: 20, TailLines: 20})
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[tool output compacted: reducer=%s]\n", ReducerBuildDiagnostics)
	fmt.Fprintf(&b, "diagnostic lines: %d\n", len(kept))
	b.WriteString("actionable diagnostics:\n")
	for _, line := range kept {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n"), []string{EvidenceDiagnosticsKept}
}

func looksLikeGitStatus(command, text string) bool {
	lowerCommand := strings.ToLower(command)
	if strings.Contains(lowerCommand, "git status") {
		return true
	}
	for _, line := range splitLines(text) {
		if strings.HasPrefix(line, "## ") || gitStatusCode(line) != "" {
			return true
		}
	}
	return false
}

func compactGitStatus(text string) (string, []string) {
	lines := splitLines(text)
	var branch string
	counts := map[string]int{}
	var conflicted []string
	var sample []string
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			branch = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		code := gitStatusCode(line)
		if code == "" {
			continue
		}
		category := gitStatusCategory(code)
		if category == "" {
			continue
		}
		counts[category]++
		if category == "conflicted" {
			conflicted = append(conflicted, line)
		}
		if len(sample) < 8 {
			sample = append(sample, line)
		}
	}

	total := 0
	for _, count := range counts {
		total += count
	}
	if total == 0 && branch == "" {
		return compactHeadTail(text, Config{HeadLines: 20, TailLines: 20})
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[tool output compacted: reducer=%s]\n", ReducerGitStatus)
	if branch != "" {
		fmt.Fprintf(&b, "branch: %s\n", branch)
	}
	fmt.Fprintf(&b, "files: %d\n", total)
	for _, category := range []string{"modified", "added", "deleted", "renamed", "untracked", "conflicted"} {
		if counts[category] > 0 {
			fmt.Fprintf(&b, "%s: %d\n", category, counts[category])
		}
	}
	if len(conflicted) > 0 {
		b.WriteString("conflicted files:\n")
		for _, line := range conflicted {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if len(sample) > 0 {
		b.WriteString("sample files:\n")
		for _, line := range sample {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n"), []string{EvidenceGitStatusCounted}
}

func gitStatusCode(line string) string {
	if len(line) < 3 {
		return ""
	}
	code := line[:2]
	if line[2] != ' ' {
		return ""
	}
	if code == "??" || code == "!!" || strings.TrimSpace(code) != "" {
		return code
	}
	return ""
}

func gitStatusCategory(code string) string {
	if strings.ContainsAny(code, "U") || code == "AA" || code == "DD" {
		return "conflicted"
	}
	if strings.Contains(code, "R") {
		return "renamed"
	}
	if strings.Contains(code, "A") {
		return "added"
	}
	if strings.Contains(code, "D") {
		return "deleted"
	}
	if code == "??" {
		return "untracked"
	}
	if strings.Contains(code, "M") {
		return "modified"
	}
	return ""
}

func compactHeadTail(text string, cfg Config) (string, []string) {
	cfg = cfg.withDefaults()
	lines := splitLines(text)
	if len(lines) <= cfg.HeadLines+cfg.TailLines {
		return text, nil
	}

	omitted := len(lines) - cfg.HeadLines - cfg.TailLines
	var b strings.Builder
	fmt.Fprintf(&b, "[tool output compacted: reducer=%s original_lines=%d omitted_lines=%d]\n", ReducerHeadTail, len(lines), omitted)
	for _, line := range lines[:cfg.HeadLines] {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "... %d line(s) omitted ...\n", omitted)
	for _, line := range lines[len(lines)-cfg.TailLines:] {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	out := strings.TrimRight(b.String(), "\n")
	return out, []string{EvidenceHeadKept, EvidenceTailKept}
}

func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}
