package contextfiles

import (
	"os"
	"path/filepath"
	"strings"
)

const durableUserContextDefaultMaxChars = 20000

// DurableUserContextOptions controls deterministic discovery of the durable
// user / memory profile files (USER.md and MEMORY.md) inside a memory dir.
// Tests inject a hermetic temp dir; production callers pass
// <GormesHome>/memory/.
type DurableUserContextOptions struct {
	// MemoryDir is the directory containing USER.md and MEMORY.md. Empty
	// returns an empty block with both files marked Missing in the report.
	MemoryDir string
	// MaxChars caps the rendered length of each file. Default
	// durableUserContextDefaultMaxChars (20000).
	MaxChars int
}

// DurableUserContextReport surfaces deterministic per-file evidence using the
// same shape as ContextFileEvidence so existing telemetry shapes stay
// uniform.
type DurableUserContextReport struct {
	User   ContextFileEvidence
	Memory ContextFileEvidence
}

// BuildDurableUserContextPrompt loads USER.md and MEMORY.md from the
// configured memory dir, threat-scans and truncates each file using the same
// helpers as the project context block, and renders a single block with
// USER content before MEMORY content.
//
// Behavior:
//   - Empty MemoryDir or both files missing → ("", report) with both
//     report.User.Missing=true and report.Memory.Missing=true so callers can
//     append unconditionally without changing slice-1 byte output.
//   - A file that trips the threat scan is replaced with a [BLOCKED:
//     ... ] marker; the raw threat content is never emitted.
//   - A file exceeding MaxChars is truncated with the existing
//     [...truncated ... kept H+T of N chars ...] marker.
//   - File read errors are recorded in the per-file evidence Error and that
//     file is dropped from the rendered block; the other file still renders.
func BuildDurableUserContextPrompt(opts DurableUserContextOptions) (string, DurableUserContextReport) {
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = durableUserContextDefaultMaxChars
	}

	var report DurableUserContextReport
	memoryDir := strings.TrimSpace(opts.MemoryDir)
	if memoryDir == "" {
		report.User = ContextFileEvidence{Source: "USER.md", Missing: true}
		report.Memory = ContextFileEvidence{Source: "MEMORY.md", Missing: true}
		return "", report
	}

	userContent, userEv := loadDurableContextFile(memoryDir, "USER.md", maxChars)
	memoryContent, memoryEv := loadDurableContextFile(memoryDir, "MEMORY.md", maxChars)
	report.User = userEv
	report.Memory = memoryEv

	sections := make([]string, 0, 2)
	if userContent != "" {
		sections = append(sections, "## USER.md\n\n"+userContent)
	}
	if memoryContent != "" {
		sections = append(sections, "## MEMORY.md\n\n"+memoryContent)
	}
	if len(sections) == 0 {
		return "", report
	}
	return "# Durable User Context\n\nThe following durable user/memory files have been loaded and should be considered when responding:\n\n" + strings.Join(sections, "\n\n"), report
}

// loadDurableContextFile reads one of USER.md or MEMORY.md from memoryDir,
// runs the same threat-scan + truncate pipeline as the project context
// helpers, and returns rendered content plus per-file evidence. Missing
// files return ("", evidence{Missing: true}) so callers can decide to drop
// the section.
func loadDurableContextFile(memoryDir, name string, maxChars int) (string, ContextFileEvidence) {
	ev := ContextFileEvidence{Source: name}
	if memoryDir == "" {
		ev.Missing = true
		return "", ev
	}
	path := filepath.Join(memoryDir, name)
	info := fileInfo(path)
	if info == nil || info.IsDir() {
		ev.Missing = true
		return "", ev
	}
	ev.Path = path
	data, err := os.ReadFile(path)
	if err != nil {
		ev.Error = err.Error()
		return "", ev
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		ev.Missing = true
		return "", ev
	}
	ev.Loaded = true
	ev.OriginalLength = len([]rune(content))
	content, ev = scanContextContent(content, name, ev)
	content, ev = truncateContextContent(content, name, maxChars, ev)
	return content, ev
}
