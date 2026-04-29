package gateway

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func (m *Manager) handleStatusCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, m.formatGatewayStatus(ctx, ev))
}

func (m *Manager) formatGatewayStatus(ctx context.Context, ev InboundEvent) string {
	frame := m.lastUsageFrameSnapshot()
	sessionID := m.resolveStatusSession(ctx, ev, frame)
	if sessionID == "" {
		sessionID = "(none)"
	}

	created := statusCreatedAt(sessionID)
	lastActivity := "(unknown)"
	title := ""
	if meta, ok := m.lookupSessionMetadata(ctx, sessionID); ok {
		title = strings.TrimSpace(meta.Title)
		if meta.CreatedAt != 0 {
			created = formatStatusTime(time.Unix(meta.CreatedAt, 0))
		}
		if meta.UpdatedAt != 0 {
			lastActivity = formatStatusTime(time.Unix(meta.UpdatedAt, 0))
		}
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

	lines := []string{
		"📊 Gormes Gateway Status",
		"",
		"Session ID:",
		sessionID,
	}
	if title != "" {
		lines = append(lines, "Title: "+title)
	}
	lines = append(lines,
		"Created: "+created,
		"Last Activity: "+lastActivity,
		fmt.Sprintf("Tokens: %d", tokens),
		"Agent Running: "+agentRunning,
		"",
		"Connected Platforms: "+connected,
	)
	return strings.Join(lines, "\n")
}

func (m *Manager) resolveStatusSession(ctx context.Context, ev InboundEvent, frame kernel.RenderFrame) string {
	key := strings.TrimSpace(ev.ChatKey())
	sessionID := ""
	if key != "" && m.cfg.SessionMap != nil {
		stored, err := m.cfg.SessionMap.Get(ctx, key)
		if err != nil {
			m.log.Warn("resolve session for status", "key", key, "err", err)
		} else if stored = strings.TrimSpace(stored); stored != "" && stored != key {
			resolved, err := resolveSession(ctx, m.cfg.SessionMap, key)
			if err != nil {
				m.log.Warn("resolve session lineage for status", "key", key, "err", err)
			}
			sessionID = strings.TrimSpace(resolved.SessionID)
		}
	}
	if sessionID == "" {
		sessionID = strings.TrimSpace(frame.SessionID)
	}
	if sessionID == "" && key != "" {
		sessionID = generateStatusSessionID(m.now(), ev)
	}
	if sessionID == "" {
		return ""
	}
	m.persistStatusSession(ctx, ev, sessionID)
	m.ensureStatusSessionMetadata(ctx, ev, sessionID)
	return sessionID
}

func generateStatusSessionID(now time.Time, ev InboundEvent) string {
	stamp := now.Format("20060102_150405")
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(ev.ChatKey())))
	_, _ = h.Write([]byte("\x00"))
	_, _ = h.Write([]byte(strings.TrimSpace(ev.UserID)))
	return fmt.Sprintf("%s_%08x", stamp, h.Sum32())
}

func statusCreatedAt(sessionID string) string {
	parts := strings.Split(strings.TrimSpace(sessionID), "_")
	if len(parts) < 2 {
		return "(unknown)"
	}
	t, err := time.ParseInLocation("20060102_150405", parts[0]+"_"+parts[1], time.Local)
	if err != nil {
		return "(unknown)"
	}
	return formatStatusTime(t)
}

func formatStatusTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04")
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
	title := statusSessionTitle(ev)
	if existing, ok := m.lookupSessionMetadata(ctx, sessionID); ok && strings.TrimSpace(existing.Title) != "" {
		title = strings.TrimSpace(existing.Title)
	}
	meta := session.Metadata{
		SessionID: sessionID,
		Title:     title,
		CreatedAt: m.now().Unix(),
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

func statusSessionTitle(ev InboundEvent) string {
	platform := strings.TrimSpace(ev.Platform)
	if platform == "" {
		platform = "gateway"
	}
	platform = strings.ToUpper(platform[:1]) + platform[1:]
	if userID := strings.TrimSpace(ev.UserID); userID != "" {
		return platform + " conversation with " + userID
	}
	if chatID := strings.TrimSpace(ev.ChatID); chatID != "" {
		return platform + " chat " + chatID
	}
	return platform + " gateway session"
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
