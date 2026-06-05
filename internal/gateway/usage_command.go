package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/usagecmd"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// AccountUsageProvider is the gateway seam for provider account-limit usage.
// Implementations must be fakeable and must not mutate provider/client state.
type AccountUsageProvider func(context.Context, InboundEvent) (llm.AccountUsageSnapshot, error)

type usageFrameSource = usagecmd.FrameSource

const (
	usageFrameSourceNone    usageFrameSource = usagecmd.FrameSourceNone
	usageFrameSourceRunning usageFrameSource = usagecmd.FrameSourceRunning
	usageFrameSourceCached  usageFrameSource = usagecmd.FrameSourceCached
)

type usageFrameSnapshot = usagecmd.FrameSnapshot

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
	lines = append(lines, llm.RenderAccountUsageLines(snapshot, llm.AccountUsageRenderOptions{})...)
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, strings.Join(lines, "\n"))
}

func (m *Manager) rememberUsageFrame(f kernel.RenderFrame) {
	m.turnMu.Lock()
	m.lastUsageFrame = f
	m.turnMu.Unlock()
	m.persistUsageFrameTokens(context.Background(), f)
}

func (m *Manager) persistUsageFrameTokens(ctx context.Context, f kernel.RenderFrame) {
	sessionID := strings.TrimSpace(f.SessionID)
	if sessionID == "" {
		return
	}
	in := f.Telemetry.TokensInTotal
	out := f.Telemetry.TokensOutTotal
	if in <= 0 && out <= 0 {
		return
	}
	writer, ok := m.cfg.SessionMap.(sessionMetadataWriter)
	if !ok {
		return
	}
	if in < 0 {
		in = 0
	}
	if out < 0 {
		out = 0
	}
	if err := writer.PutMetadata(ctx, session.Metadata{
		SessionID:      sessionID,
		UpdatedAt:      m.now().Unix(),
		TokensInTotal:  in,
		TokensOutTotal: out,
	}); err != nil {
		m.log.Warn("persist session token usage", "session_id", sessionID, "err", err)
	}
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
	return usagecmd.RenderFrameLines(snapshot)
}
