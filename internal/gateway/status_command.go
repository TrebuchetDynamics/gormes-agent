package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func (m *Manager) handleStatusCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, m.formatGatewayStatus(ctx, ev))
}

func (m *Manager) formatGatewayStatus(ctx context.Context, ev InboundEvent) string {
	frame := m.lastUsageFrameSnapshot()
	sessionID := ""
	if m.cfg.SessionMap != nil {
		if mapped, err := m.cfg.SessionMap.Get(ctx, ev.ChatKey()); err == nil {
			sessionID = strings.TrimSpace(mapped)
		} else {
			m.log.Warn("load session mapping for status", "key", ev.ChatKey(), "err", err)
		}
	}
	if sessionID == "" {
		sessionID = strings.TrimSpace(frame.SessionID)
	}
	if sessionID == "" {
		sessionID = "(none)"
	}

	lastActivity := "(unknown)"
	if meta, ok := m.lookupSessionMetadata(ctx, sessionID); ok && meta.UpdatedAt != 0 {
		lastActivity = time.Unix(meta.UpdatedAt, 0).Format("2006-01-02 15:04")
	}

	tokens := frame.Telemetry.TokensInTotal + frame.Telemetry.TokensOutTotal
	agentRunning := "No"
	if m.hasActiveTurn() {
		agentRunning = "Yes"
	}
	platforms := m.connectedPlatforms()
	if len(platforms) == 0 && strings.TrimSpace(ev.Platform) != "" {
		platforms = []string{strings.TrimSpace(ev.Platform)}
	}
	connected := "(none)"
	if len(platforms) > 0 {
		connected = strings.Join(platforms, ", ")
	}

	return fmt.Sprintf("📊 Gormes Gateway Status\n\nSession ID:\n%s\nTitle: (untitled)\nCreated: (unknown)\nLast Activity: %s\nTokens: %d\nAgent Running: %s\n\nConnected Platforms: %s", sessionID, lastActivity, tokens, agentRunning, connected)
}

type sessionMetadataGetter interface {
	GetMetadata(context.Context, string) (session.Metadata, bool, error)
}

func (m *Manager) lookupSessionMetadata(ctx context.Context, sessionID string) (session.Metadata, bool) {
	store, ok := m.cfg.SessionMap.(sessionMetadataGetter)
	if !ok {
		return session.Metadata{}, false
	}
	meta, ok, err := store.GetMetadata(ctx, sessionID)
	if err != nil {
		m.log.Warn("load session metadata for status", "session_id", sessionID, "err", err)
		return session.Metadata{}, false
	}
	return meta, ok
}

func (m *Manager) lastUsageFrameSnapshot() kernel.RenderFrame {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	return m.lastUsageFrame
}
