package gateway

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/statuscmd"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// statusTitleUnavailable is the documented degraded-mode sentinel rendered
// when the session has no metadata title and no auto-title generation has
// produced one. Holding the field in the response (rather than silently
// omitting it) preserves Hermes /status field-order parity and surfaces the
// auto-title gap as visible operator evidence.
const statusTitleUnavailable = "title_unavailable"

func (m *Manager) handleStatusCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, m.formatGatewayStatus(ctx, ev))
}

func (m *Manager) formatGatewayStatus(ctx context.Context, ev InboundEvent) string {
	frame := m.lastUsageFrameSnapshot()
	route := m.agentRouteForInbound(ev)
	sessionID := m.resolveStatusSession(ctx, ev, frame)
	if sessionID == "" {
		sessionID = "(none)"
	}

	created := statusCreatedAt(sessionID)
	lastActivity := "(unknown)"
	title := ""
	metadataTokens := 0
	if meta, ok := m.lookupSessionMetadata(ctx, sessionID); ok {
		title = strings.TrimSpace(meta.Title)
		metadataTokens = meta.TokensInTotal + meta.TokensOutTotal
		if meta.CreatedAt != 0 {
			created = formatStatusTime(time.Unix(meta.CreatedAt, 0))
		}
		if meta.UpdatedAt != 0 {
			lastActivity = formatStatusTime(time.Unix(meta.UpdatedAt, 0))
		}
	}
	if title == "" {
		title = statusTitleUnavailable
	}

	tokens := frame.Telemetry.TokensInTotal + frame.Telemetry.TokensOutTotal
	if metadataTokens > tokens {
		tokens = metadataTokens
	}
	agentRunning := "No"
	if m.hasActiveTurn() {
		agentRunning = "Yes ⚡"
	}
	platforms := m.connectedPlatforms()
	if len(platforms) == 0 && strings.TrimSpace(ev.Platform) != "" {
		platforms = []string{strings.TrimSpace(ev.Platform)}
	}
	connected := "(none)"
	if len(platforms) > 0 {
		connected = strings.Join(platforms, ", ")
	}

	esc := func(s string) string { return tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, s) }

	lines := []string{
		"📊 **Gormes Gateway Status**",
		"",
		"**Session ID:** `" + sessionID + "`",
		"**Title:** " + esc(title),
		"**Created:** " + esc(created),
		"**Last Activity:** " + esc(lastActivity),
		fmt.Sprintf("**Cumulative API tokens (re-sent each call):** %s", formatStatusTokenTotal(tokens)),
		"**Agent Running:** " + agentRunning,
	}
	if queueDepth := m.followUpQueueDepth(); queueDepth > 0 {
		lines = append(lines, fmt.Sprintf("**Queued follow-ups:** %d", queueDepth))
	}
	if route.Enabled {
		lines = append(lines,
			"**Agent ID:** `"+strings.TrimSpace(route.Decision.AgentID)+"`",
			"**Agent Binding:** `"+string(route.Decision.BindingTier)+"`",
		)
	}
	if kanbanStatus, ok := m.kanbanDispatcherStatus(ctx); ok {
		lines = append(lines, statuscmd.FormatKanbanDispatcherLines(statuscmd.KanbanDispatcherStatus{
			State:       string(kanbanStatus.State),
			LastTickAt:  kanbanStatus.LastTickAt,
			LastError:   kanbanStatus.LastError,
			Spawned:     kanbanStatus.Spawned,
			SpawnFailed: kanbanStatus.SpawnFailed,
			AutoBlocked: kanbanStatus.AutoBlocked,
		}, esc)...)
	}
	lines = append(lines,
		"",
		"**Connected Platforms:** "+esc(connected),
	)
	return strings.Join(lines, "\n")
}

func (m *Manager) kanbanDispatcherStatus(ctx context.Context) (KanbanDispatcherStatus, bool) {
	reader, ok := m.cfg.RuntimeStatus.(interface {
		ReadRuntimeStatus(context.Context) (RuntimeStatus, error)
	})
	if !ok {
		return KanbanDispatcherStatus{}, false
	}
	status, err := reader.ReadRuntimeStatus(ctx)
	if err != nil {
		m.log.Debug("read kanban dispatcher status", "err", err)
		return KanbanDispatcherStatus{}, false
	}
	kanbanStatus := status.KanbanDispatcher
	if kanbanStatus.State == "" &&
		kanbanStatus.LastTickAt == "" &&
		kanbanStatus.LastError == "" &&
		kanbanStatus.Spawned == 0 &&
		kanbanStatus.SpawnFailed == 0 &&
		kanbanStatus.AutoBlocked == 0 {
		return KanbanDispatcherStatus{}, false
	}
	return kanbanStatus, true
}

func (m *Manager) resolveStatusSession(ctx context.Context, ev InboundEvent, frame kernel.RenderFrame) string {
	key := strings.TrimSpace(m.sessionKeyForInbound(ev))
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
		sessionID = generateStatusSessionIDForKey(m.now(), ev, key)
	}
	if sessionID == "" {
		return ""
	}
	m.persistStatusSession(ctx, key, sessionID)
	m.ensureStatusSessionMetadata(ctx, ev, sessionID)
	return sessionID
}

func generateStatusSessionIDForKey(now time.Time, ev InboundEvent, key string) string {
	stamp := now.Format("20060102_150405")
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(key)))
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

func formatStatusTokenTotal(tokens int) string {
	raw := fmt.Sprintf("%d", tokens)
	negative := strings.HasPrefix(raw, "-")
	digits := raw
	if negative {
		digits = strings.TrimPrefix(raw, "-")
	}
	if len(digits) <= 3 {
		return raw
	}
	leading := len(digits) % 3
	if leading == 0 {
		leading = 3
	}
	var out strings.Builder
	out.Grow(len(raw) + len(digits)/3)
	if negative {
		out.WriteByte('-')
	}
	out.WriteString(digits[:leading])
	for i := leading; i < len(digits); i += 3 {
		out.WriteByte(',')
		out.WriteString(digits[i : i+3])
	}
	return out.String()
}

func (m *Manager) persistStatusSession(ctx context.Context, sessionKey, sessionID string) {
	if m.cfg.SessionMap == nil {
		return
	}
	key := strings.TrimSpace(sessionKey)
	if key == "" || strings.TrimSpace(sessionID) == "" {
		return
	}
	if err := m.cfg.SessionMap.Put(ctx, key, sessionID); err != nil {
		m.log.Warn("persist session_id for status", "key", key, "session_id", sessionID, "err", err)
	}
}

// ensureStatusSessionMetadata writes durable identity for the session row the
// /status command resolved. It deliberately does NOT seed a synthetic
// "Telegram conversation with X" title: the row contract says missing titles
// must surface the title_unavailable degraded-mode sentinel instead of a
// hardcoded fake. Auto-title generation is the eventual fix and is tracked
// as a separate progress row.
func (m *Manager) ensureStatusSessionMetadata(ctx context.Context, ev InboundEvent, sessionID string) {
	writer, ok := m.cfg.SessionMap.(sessionMetadataWriter)
	if !ok {
		return
	}
	source := sessionSourceFromInbound(ev)
	title := ""
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
