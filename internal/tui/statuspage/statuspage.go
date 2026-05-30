package statuspage

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/transientpage"
)

// SlashResult is the behavior-only result for /status.
type SlashResult struct {
	Page          transientpage.State
	Open          bool
	StatusMessage string
}

// HandleSlash validates /status state and builds the read-only status page.
func HandleSlash(frame kernel.RenderFrame, sessionID string) SlashResult {
	if strings.TrimSpace(sessionID) == "" {
		return SlashResult{StatusMessage: "no active session"}
	}
	return SlashResult{Page: Build(frame, sessionID), Open: true, StatusMessage: "status opened"}
}

// Build renders the read-only TUI status page for the current frame. If the
// explicit session ID is empty, the frame session ID is used.
func Build(frame kernel.RenderFrame, sessionID string) transientpage.State {
	if strings.TrimSpace(sessionID) == "" {
		sessionID = frame.SessionID
	}
	lines := []string{"Gormes TUI Status"}
	appendStatusLine := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, fmt.Sprintf("%s: %s", label, value))
	}

	appendStatusLine("Session", sessionID)
	appendStatusLine("Phase", frame.Phase.String())
	appendStatusLine("Model", frame.Model)
	appendStatusLine("Provider", frame.ProviderStatus.Provider)
	appendStatusLine("Runtime", frame.ProviderStatus.Runtime)
	if reasoning := FormatReasoning(frame.ReasoningEffort); reasoning != "" {
		appendStatusLine("Reasoning effort", reasoning)
	}
	if frame.ContextStatus != nil {
		lines = append(lines, FormatContext(*frame.ContextStatus))
		if budget := FormatContextBudget(*frame.ContextStatus); budget != "" {
			appendStatusLine("Budget", budget)
		}
	}
	if telem := FormatTelemetry(frame.Telemetry); telem != "" {
		appendStatusLine("Telemetry", telem)
	}
	lines = append(lines, fmt.Sprintf("History messages: %d", len(frame.History)))
	appendStatusLine("Last error", frame.LastError)
	return transientpage.State{Title: "Status", Body: strings.Join(lines, "\n")}
}

func FormatReasoning(r llm.ReasoningEffortEvidence) string {
	effort := strings.TrimSpace(r.Requested)
	if effort == "" && r.Effort != "" {
		effort = string(r.Effort)
	}
	if effort == "" {
		return ""
	}
	if r.Forwarded {
		return effort + " (forwarded)"
	}
	if r.Reason != "" {
		return effort + " (not forwarded: " + r.Reason + ")"
	}
	return effort
}

func FormatContext(status llm.ContextStatus) string {
	line := fmt.Sprintf("Context: %d / %d tokens", status.LastTotalTokens, status.ContextLength)
	if status.UsagePercent > 0 {
		line += fmt.Sprintf(" (%.1f%%)", status.UsagePercent)
	}
	if status.Engine != "" {
		line += " via " + status.Engine
	}
	return line
}

func FormatContextBudget(status llm.ContextStatus) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(status.Budget.State) != "" {
		parts = append(parts, status.Budget.State)
	}
	if status.Budget.RemainingTokens > 0 {
		parts = append(parts, fmt.Sprintf("%d tokens remaining", status.Budget.RemainingTokens))
	}
	return strings.Join(parts, ", ")
}

func FormatTelemetry(t telemetry.Snapshot) string {
	if t.TokensInTotal == 0 && t.TokensOutTotal == 0 && t.LatencyMsLast == 0 && t.TokensPerSec == 0 {
		return ""
	}
	return fmt.Sprintf("%d in / %d out / %d ms / %.1f tok/s", t.TokensInTotal, t.TokensOutTotal, t.LatencyMsLast, t.TokensPerSec)
}
