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
	sessionID := m.resolveStatusSession(ctx, ev, frame)
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

func (m *Manager) resolveStatusSession(ctx context.Context, ev InboundEvent, frame kernel.RenderFrame) string {
	sessionID := ""
	resolved, err := resolveSession(ctx, m.cfg.SessionMap, ev.ChatKey())
	if err != nil {
		m.log.Warn("resolve session for status", "key", ev.ChatKey(), "err", err)
	}
	sessionID = strings.TrimSpace(resolved.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(frame.SessionID)
	}
	if sessionID == "" {
		return ""
	}
	m.persistStatusSession(ctx, ev, sessionID)
	m.ensureStatusSessionMetadata(ctx, ev, sessionID)
	return sessionID
}

func (m *Manager) persistStatusSession(ctx context.Context, ev InboundEvent, sessionID string) {
	if m.cfg.SessionMap == nil {
		return
	}
	key := strings.TrimSpace(ev.ChatKey())
	if key == "" || strings.TrimSpace(sessionID) == "" {
		return
	}
	if err := m.cfg.SessionMap.Put(ctx, key, sessionID); err != nil {
		m.log.Warn("persist session_id for status", "key", key, "session_id", sessionID, "err", err)
	}
}

func (m *Manager) ensureStatusSessionMetadata(ctx context.Context, ev InboundEvent, sessionID string) {
	writer, ok := m.cfg.SessionMap.(sessionMetadataWriter)
	if !ok {
		return
	}
	source := sessionSourceFromInbound(ev)
	meta := session.Metadata{
		SessionID: sessionID,
		UpdatedAt: m.now().Unix(),
	}
	if source.Platform != "" && source.ChatID != "" {
		meta.Source = source.Platform
		meta.ChatID = source.ChatID
		meta.UserID = source.UserID
	}
	if err := writer.PutMetadata(ctx, meta); err != nil {
		m.log.Warn("write session metadata for status", "session_id", sessionID, "err", err)
	}
}

func (m *Manager) lookupSessionMetadata(ctx context.Context, sessionID string) (session.Metadata, bool) {
	store, ok := m.cfg.SessionMap.(sessionMetadataReader)
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
