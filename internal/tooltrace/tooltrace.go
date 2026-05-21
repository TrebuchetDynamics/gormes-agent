package tooltrace

import (
	"fmt"
	"strings"
)

// FormatPlain renders one kernel soul tool-start entry as operator progress.
// Non-start completion/status events intentionally return empty output because
// gateway progress is keyed off tool.started.
func FormatPlain(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if strings.HasPrefix(text, "tool done:") ||
		strings.HasPrefix(text, "tool completed:") ||
		strings.HasPrefix(text, "tool cancelled:") ||
		strings.HasPrefix(text, "tool error:") ||
		strings.HasPrefix(text, "tool status:") {
		return ""
	}
	if strings.HasPrefix(text, "tool: ") {
		payload := strings.TrimSpace(strings.TrimPrefix(text, "tool: "))
		name, arg, ok := strings.Cut(payload, ":")
		if ok {
			name = strings.TrimSpace(name)
			arg = strings.TrimSpace(arg)
			if !isKnownToolTraceName(name) {
				return progressLine("ACTION", "tool", "Running tool task")
			}
			if line, ok := semanticProgressLine(name, arg); ok {
				return line
			}
			if suppressToolTraceArgs(name) {
				return toolTraceIcon(name) + " " + name + "..."
			}
			if arg == "" {
				return toolTraceIcon(name) + " " + name + "..."
			}
			return toolTraceIcon(name) + " " + name + ": " + quoteAndTruncate(name, arg, 40)
		}
		name = strings.TrimSpace(payload)
		if isKnownToolTraceName(name) {
			if line, ok := semanticProgressLine(name, ""); ok {
				return line
			}
			return toolTraceIcon(name) + " " + name + "..."
		}
		return progressLine("ACTION", "tool", "Running tool task")
	}
	return "🔧 " + text
}

// FormatBlock renders a persistent Hermes-style tool transcript and collapses
// consecutive identical rendered lines as "(×N)".
func FormatBlock(events []string) string {
	return FormatBlockMode(events, "all")
}

// FormatBlockMode renders tool progress using Hermes gateway tool_progress
// modes. "new" suppresses consecutive calls to the same tool name; "all" and
// "verbose" currently share the compact preview renderer.
func FormatBlockMode(events []string, mode string) string {
	var lines []string
	var last string
	var lastTool string
	var sawTool bool
	repeats := 1
	flush := func() {
		if last == "" {
			return
		}
		if repeats > 1 {
			lines = append(lines, fmt.Sprintf("%s (×%d)", last, repeats))
		} else {
			lines = append(lines, last)
		}
	}
	for _, event := range events {
		text := strings.TrimSpace(event)
		if !strings.HasPrefix(text, "tool") {
			continue
		}
		line := FormatPlain(text)
		if line == "" {
			continue
		}
		toolName := toolTraceName(text)
		if mode == "new" && sawTool && toolName == lastTool {
			continue
		}
		lastTool = toolName
		sawTool = true
		if line == last {
			repeats++
			continue
		}
		flush()
		last = line
		repeats = 1
	}
	flush()
	return strings.Join(lines, "\n")
}

func toolTraceName(text string) string {
	payload := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "tool: "))
	name, _, ok := strings.Cut(payload, ":")
	if ok {
		return strings.TrimSpace(name)
	}
	return payload
}

func isKnownToolTraceName(name string) bool {
	switch strings.TrimSpace(name) {
	case "memory", "honcho_context", "honcho_search", "honcho_profile", "honcho_conclude", "session_search",
		"skill_view", "skills_list", "skill_manage", "todo", "cronjob",
		"search_files", "web_search", "web_extract", "web_crawl",
		"browser_navigate", "browser_snapshot", "browser_click", "browser_type", "browser_scroll",
		"browser_back", "browser_press", "browser_get_images", "browser_vision", "browser_cdp", "browser_dialog",
		"read_file", "patch", "write_file", "terminal", "execute_code", "process",
		"transcribe_audio", "text_to_speech":
		return true
	default:
		return false
	}
}

func toolTraceIcon(name string) string {
	switch strings.TrimSpace(name) {
	case "memory", "honcho_context", "honcho_search", "honcho_profile", "honcho_conclude", "session_search":
		return "🧠"
	case "skill_view", "skills_list", "skill_manage":
		return "📚"
	case "todo":
		return "📋"
	case "cronjob":
		return "⏰"
	case "search_files":
		return "🔎"
	case "web_search":
		return "🔍"
	case "web_extract":
		return "📄"
	case "web_crawl":
		return "🕸️"
	case "browser_navigate":
		return "🌐"
	case "browser_snapshot":
		return "📸"
	case "browser_click":
		return "👆"
	case "browser_type", "browser_press":
		return "⌨️"
	case "browser_scroll":
		return "📜"
	case "browser_back":
		return "◀️"
	case "browser_get_images":
		return "🖼️"
	case "browser_vision":
		return "👁️"
	case "browser_cdp", "browser_dialog":
		return "🖥️"
	case "read_file":
		return "📖"
	case "patch", "write_file":
		return "🔧"
	case "terminal", "process":
		return "💻"
	case "execute_code":
		return "💻"
	case "transcribe_audio":
		return "🎙️"
	case "text_to_speech":
		return "🔊"
	default:
		return "🔧"
	}
}

func suppressToolTraceArgs(name string) bool {
	switch strings.TrimSpace(name) {
	case "transcribe_audio", "text_to_speech":
		return true
	default:
		return false
	}
}

func semanticProgressLine(name, arg string) (string, bool) {
	switch strings.TrimSpace(name) {
	case "memory":
		return semanticMemoryProgress(arg), true
	case "honcho_context", "honcho_search":
		return progressLine("INFO", "memory", "Searching local memory"), true
	case "honcho_profile":
		return progressLine("INFO", "memory", "Loading memory profile"), true
	case "honcho_conclude":
		return progressLine("ACTION", "memory", "Updating local memory index"), true
	case "session_search":
		return progressLine("INFO", "memory", "Searching session history"), true
	case "terminal", "process":
		return semanticTerminalProgress(arg), true
	case "execute_code":
		return progressLine("ACTION", "runtime", "Running code block"), true
	case "transcribe_audio":
		return progressLine("ACTION", "audio", "Transcribing voice input"), true
	case "text_to_speech":
		return progressLine("ACTION", "audio", "Generating voice reply"), true
	default:
		return "", false
	}
}

func progressLine(level, subsystem, message string) string {
	return fmt.Sprintf("%-6s [%s] %s", level, subsystem, message)
}

func semanticMemoryProgress(arg string) string {
	lower := strings.ToLower(arg)
	switch {
	case strings.Contains(lower, "add"), strings.Contains(lower, "update"), strings.Contains(lower, "write"):
		return progressLine("ACTION", "memory", "Updating local memory index")
	case strings.Contains(lower, "summar"):
		return progressLine("ACTION", "memory", "Summarizing prior context")
	default:
		return progressLine("INFO", "memory", "Loading session memory")
	}
}

func semanticTerminalProgress(arg string) string {
	lower := strings.ToLower(arg)
	switch {
	case strings.Contains(lower, "oversized skill"):
		return progressLine("ACTION", "skills", "Checking oversized skill events")
	case strings.Contains(lower, "active_profile"):
		return progressLine("ACTION", "profile", "Inspecting active Gormes profile")
	case strings.Contains(lower, "profile config candida"):
		return progressLine("ACTION", "config", "Verifying profile config candidates")
	case strings.Contains(lower, "wc -c") && strings.Contains(lower, ".gormes/profiles"):
		return progressLine("ACTION", "profile", "Measuring profile state size")
	case strings.Contains(lower, "python3") && (strings.Contains(lower, "profile") || strings.Contains(lower, "state") || strings.Contains(lower, "payload")):
		return progressLine("ACTION", "profile", "Parsing profile state")
	case strings.Contains(lower, ".gormes profile"):
		return progressLine("ACTION", "profile", "Inspecting Gormes profile state")
	case strings.Contains(lower, "git status"):
		return progressLine("ACTION", "repo", "Inspecting repository status")
	case strings.Contains(lower, "go test"):
		return progressLine("ACTION", "runtime", "Running test suite")
	case strings.Contains(lower, "ffmpeg") || strings.Contains(lower, "wav"):
		return progressLine("ACTION", "audio", "Processing audio")
	case strings.Contains(lower, "curl ") || strings.Contains(lower, "http://") || strings.Contains(lower, "https://"):
		return progressLine("ACTION", "network", "Fetching remote content")
	default:
		return progressLine("ACTION", "runtime", "Running system check")
	}
}

func quoteAndTruncate(toolName, s string, limit int) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if limit > 0 {
		runes := []rune(s)
		if len(runes) > limit {
			if limit <= 3 {
				if rightEdgePreviewTool(toolName) {
					s = string(runes[len(runes)-limit:])
				} else {
					s = string(runes[:limit])
				}
			} else {
				kept := limit - 3
				if rightEdgePreviewTool(toolName) {
					s = "..." + string(runes[len(runes)-kept:])
				} else {
					s = string(runes[:kept]) + "..."
				}
			}
		}
	}
	return `"` + s + `"`
}

func rightEdgePreviewTool(name string) bool {
	// File-path arguments keep the tail (filename matters more than the
	// directory chain). URLs and search queries keep the head: the domain
	// or the first words of the query are the user-meaningful part, and a
	// `"...et.com/market-data/..."` preview hides which site Gormes is
	// actually visiting.
	switch strings.TrimSpace(name) {
	case "read_file", "write_file", "patch":
		return true
	default:
		return false
	}
}
