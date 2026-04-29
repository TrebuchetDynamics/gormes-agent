package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// AccountUsageProvider is the gateway seam for provider account-limit usage.
// Implementations must be fakeable and must not mutate provider/client state.
type AccountUsageProvider func(context.Context, InboundEvent) (hermes.AccountUsageSnapshot, error)

type usageFrameSource string

const (
	usageFrameSourceNone    usageFrameSource = "unavailable"
	usageFrameSourceRunning usageFrameSource = "running turn"
	usageFrameSourceCached  usageFrameSource = "cached turn"
)

type usageFrameSnapshot struct {
	Frame  kernel.RenderFrame
	Source usageFrameSource
}

func (m *Manager) handleUsageCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	lines := []string{"Usage"}
	frame := m.usageFrameForCommand()
	lines = append(lines, renderUsageFrameLines(frame)...)

	if m.cfg.AccountUsage == nil {
		lines = append(lines, "Provider: unavailable", "Usage unavailable: account usage provider is not configured")
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, strings.Join(lines, "\n"))
		return
	}
	snapshot, err := m.cfg.AccountUsage(ctx, ev)
	if err != nil {
		lines = append(lines, "Provider: unavailable", "Usage unavailable: "+err.Error())
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, strings.Join(lines, "\n"))
		return
	}
	lines = append(lines, hermes.RenderAccountUsageLines(snapshot, hermes.AccountUsageRenderOptions{})...)
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, strings.Join(lines, "\n"))
}

func (m *Manager) rememberUsageFrame(f kernel.RenderFrame) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.lastUsageFrame = f
}

func (m *Manager) usageFrameForCommand() usageFrameSnapshot {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	if m.turnPlatform != "" && m.turnChatID != "" && m.lastUsageFrame.Model != "" {
		return usageFrameSnapshot{Frame: m.lastUsageFrame, Source: usageFrameSourceRunning}
	}
	if m.lastUsageFrame.Model != "" || m.lastUsageFrame.SessionID != "" || m.lastUsageFrame.Telemetry.TokensInTotal != 0 || m.lastUsageFrame.Telemetry.TokensOutTotal != 0 {
		return usageFrameSnapshot{Frame: m.lastUsageFrame, Source: usageFrameSourceCached}
	}
	return usageFrameSnapshot{Source: usageFrameSourceNone}
}

func renderUsageFrameLines(snapshot usageFrameSnapshot) []string {
	lines := []string{"Usage source: " + string(snapshot.Source)}
	if snapshot.Source == usageFrameSourceNone {
		return append(lines, "Runtime usage unavailable: no running or cached turn telemetry")
	}
	frame := snapshot.Frame
	model := strings.TrimSpace(frame.Model)
	if model == "" {
		model = "unknown"
	}
	sessionID := strings.TrimSpace(frame.SessionID)
	if sessionID == "" {
		sessionID = "unknown"
	}
	lines = append(lines,
		"Model: "+model,
		"Session: "+sessionID,
		fmt.Sprintf("Tokens: %d in / %d out", frame.Telemetry.TokensInTotal, frame.Telemetry.TokensOutTotal),
	)
	if frame.Telemetry.LatencyMsLast > 0 {
		lines = append(lines, fmt.Sprintf("Last latency: %d ms", frame.Telemetry.LatencyMsLast))
	}
	if frame.Telemetry.TokensPerSec > 0 {
		lines = append(lines, fmt.Sprintf("Speed: %.2f tokens/sec", frame.Telemetry.TokensPerSec))
	}
	return lines
}
