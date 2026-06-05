package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/titlecmd"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

const maxSessionTitleRunes = titlecmd.MaxSessionTitleRunes

func (m *Manager) handleTitleCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	sessionID := m.resolveStatusSession(ctx, ev, m.lastUsageFrameSnapshot())
	if sessionID == "" {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Session database not available.")
		return
	}

	titleArg, hasArg := parseTitleCommandArg(ev.Text)
	if !hasArg {
		m.handleTitleShow(ctx, ch, ev, sessionID)
		return
	}

	title, err := sanitizeSessionTitle(titleArg)
	if err != nil {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, err.Error())
		return
	}
	if title == "" {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Title is empty after cleanup. Please use printable characters.")
		return
	}
	writer, ok := m.cfg.SessionMap.(sessionMetadataWriter)
	if !ok {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Session database not available.")
		return
	}

	meta := session.Metadata{
		SessionID:        sessionID,
		Title:            title,
		TitleManuallySet: true,
		UpdatedAt:        m.now().Unix(),
		LineageKind:      session.LineageKindPrimary,
	}
	if existing, ok := m.lookupSessionMetadata(ctx, sessionID); ok {
		meta.Source = existing.Source
		meta.ChatID = existing.ChatID
		meta.UserID = existing.UserID
		meta.CreatedAt = existing.CreatedAt
		meta.ParentSessionID = existing.ParentSessionID
		meta.LineageKind = existing.LineageKind
	}
	if meta.Source == "" || meta.ChatID == "" {
		source := sessionSourceFromInbound(ev)
		meta.Source = source.Platform
		meta.ChatID = source.ChatID
		meta.UserID = source.UserID
	}
	if meta.CreatedAt == 0 {
		meta.CreatedAt = m.now().Unix()
	}
	if err := writer.PutMetadata(ctx, meta); err != nil {
		m.log.Warn("set session title", "session_id", sessionID, "err", err)
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Session title failed: "+err.Error())
		return
	}
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Session title set: "+title)
}

func (m *Manager) handleTitleShow(ctx context.Context, ch Channel, ev InboundEvent, sessionID string) {
	lines := []string{"Session ID: " + sessionID}
	if meta, ok := m.lookupSessionMetadata(ctx, sessionID); ok && strings.TrimSpace(meta.Title) != "" {
		lines = append(lines, "Title: "+strings.TrimSpace(meta.Title))
	} else {
		lines = append(lines, "No title set. Usage: /title <your session title>")
	}
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, strings.Join(lines, "\n"))
}

func parseTitleCommandArg(text string) (string, bool) {
	return titlecmd.ParseArg(text)
}

func sanitizeSessionTitle(title string) (string, error) {
	return titlecmd.Sanitize(title)
}
