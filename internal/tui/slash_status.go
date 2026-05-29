package tui

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func statusSlashHandler(_ string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "status: TUI unavailable"}
	}
	if strings.TrimSpace(model.SessionID()) == "" {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "no active session"}
	}
	page := BuildStatusPage(model.frame, model.SessionID())
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: "status opened"}
}

func BuildStatusPage(frame kernel.RenderFrame, sessionID string) TransientPageState {
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
	if reasoning := formatStatusReasoning(frame.ReasoningEffort); reasoning != "" {
		appendStatusLine("Reasoning effort", reasoning)
	}
	if frame.ContextStatus != nil {
		lines = append(lines, formatStatusContext(*frame.ContextStatus))
		if budget := formatStatusContextBudget(*frame.ContextStatus); budget != "" {
			appendStatusLine("Budget", budget)
		}
	}
	if telem := formatStatusTelemetry(frame.Telemetry); telem != "" {
		appendStatusLine("Telemetry", telem)
	}
	lines = append(lines, fmt.Sprintf("History messages: %d", len(frame.History)))
	appendStatusLine("Last error", frame.LastError)
	return TransientPageState{Title: "Status", Body: strings.Join(lines, "\n")}
}

func formatStatusReasoning(r hermes.ReasoningEffortEvidence) string {
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

func formatStatusContext(status hermes.ContextStatus) string {
	line := fmt.Sprintf("Context: %d / %d tokens", status.LastTotalTokens, status.ContextLength)
	if status.UsagePercent > 0 {
		line += fmt.Sprintf(" (%.1f%%)", status.UsagePercent)
	}
	if status.Engine != "" {
		line += " via " + status.Engine
	}
	return line
}

func formatStatusContextBudget(status hermes.ContextStatus) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(status.Budget.State) != "" {
		parts = append(parts, status.Budget.State)
	}
	if status.Budget.RemainingTokens > 0 {
		parts = append(parts, fmt.Sprintf("%d tokens remaining", status.Budget.RemainingTokens))
	}
	return strings.Join(parts, ", ")
}

func formatStatusTelemetry(t telemetry.Snapshot) string {
	if t.TokensInTotal == 0 && t.TokensOutTotal == 0 && t.LatencyMsLast == 0 && t.TokensPerSec == 0 {
		return ""
	}
	return fmt.Sprintf("%d in / %d out / %d ms / %.1f tok/s", t.TokensInTotal, t.TokensOutTotal, t.LatencyMsLast, t.TokensPerSec)
}
