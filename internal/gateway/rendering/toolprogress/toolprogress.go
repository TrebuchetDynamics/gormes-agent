package toolprogress

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rendering/textlimit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/trace"
)

// ToolProgressStatus is the channel-neutral lifecycle state for structured
// tool progress. Text-only channels can ignore it; first-party channels such
// as Navivox can render it as native UI instead of assistant prose.
type Status string

const (
	Started  Status = "started"
	Updated  Status = "updated"
	Finished Status = "finished"
	Failed   Status = "failed"
)

// ToolProgressEvent carries redacted, bounded tool-progress evidence. It must
// never include raw tool arguments, stdout, credentials, or full logs.
type Event struct {
	ID       string
	ToolName string
	Status   Status
	Summary  string
	Metadata map[string]any
}

// FormatToolProgressPlain renders the persistent Hermes-style tool progress
// transcript for gateway platforms that can edit progress messages.
func FormatPlain(f kernel.RenderFrame) string {
	return FormatPlainMode(f, "all")
}

// FormatToolProgressPlainMode renders tool progress with Hermes gateway
// display.tool_progress semantics for the compact progress transcript.
func FormatPlainMode(f kernel.RenderFrame, mode string) string {
	mode = normalizeGatewayToolProgressMode(mode)
	if mode == "off" {
		return ""
	}
	return truncate(formatToolTraceBlockPlainMode(f.SoulEvents, mode))
}

// FormatToolProgressEvents extracts bounded structured tool progress for
// first-party channels. Unlike text progress, summaries deliberately avoid raw
// tool arguments so URLs, command lines, and credentials cannot become chat
// prose.
func Events(f kernel.RenderFrame, mode, requestID string) []Event {
	mode = normalizeGatewayToolProgressMode(mode)
	if mode == "off" {
		return nil
	}
	status := toolProgressStatusForPhase(f.Phase)
	events := make([]Event, 0, len(f.SoulEvents))
	var lastTool string
	var sawTool bool
	var lastRaw string
	for _, event := range f.SoulEvents {
		raw := strings.TrimSpace(event.Text)
		if !strings.HasPrefix(raw, "tool") {
			continue
		}
		if trace.FormatPlain(raw) == "" {
			continue
		}
		name := toolTraceName(raw)
		if !isKnownToolTraceName(name) {
			name = "tool_progress"
		}
		if mode == "new" && sawTool && name == lastTool {
			continue
		}
		if raw == lastRaw {
			continue
		}
		lastTool = name
		lastRaw = raw
		sawTool = true
		index := len(events) + 1
		events = append(events, Event{
			ID:       stableToolProgressID(requestID, name, index),
			ToolName: name,
			Status:   status,
			Summary:  toolProgressSummary(name, status),
		})
	}
	return events
}

func formatToolTraceBlockPlainMode(events []kernel.SoulEntry, mode string) string {
	texts := make([]string, 0, len(events))
	for _, event := range events {
		texts = append(texts, event.Text)
	}
	return trace.FormatBlockMode(texts, mode)
}

func toolTraceName(text string) string {
	payload := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "tool: "))
	name, _, ok := strings.Cut(payload, ":")
	if ok {
		return strings.TrimSpace(name)
	}
	return payload
}

func toolProgressStatusForPhase(phase kernel.Phase) Status {
	switch phase {
	case kernel.PhaseIdle:
		return Finished
	case kernel.PhaseFailed, kernel.PhaseCancelling:
		return Failed
	default:
		return Started
	}
}

func Summary(name string, status Status) string {
	switch status {
	case Finished:
		return name + " finished"
	case Failed:
		return name + " failed"
	case Updated:
		return name + " updated"
	default:
		return name + " started"
	}
}

func toolProgressSummary(name string, status Status) string {
	return Summary(name, status)
}

func stableToolProgressID(requestID, toolName string, index int) string {
	requestID = slugToolProgressIDPart(requestID)
	if requestID == "" {
		requestID = "turn"
	}
	toolName = slugToolProgressIDPart(toolName)
	if toolName == "" {
		toolName = "tool"
	}
	return fmt.Sprintf("%s-%s-%d", requestID, toolName, index)
}

func slugToolProgressIDPart(raw string) string {
	raw = strings.TrimSpace(raw)
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			if b.Len() > 0 {
				b.WriteByte('-')
			}
		}
		if b.Len() >= 80 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

func NormalizeMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "off", "new", "all", "verbose":
		return normalized
	default:
		return "all"
	}
}

func normalizeGatewayToolProgressMode(mode string) string {
	return NormalizeMode(mode)
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

func truncate(s string) string {
	return textlimit.TruncateMarkdownV2Safe(s, 4000)
}
