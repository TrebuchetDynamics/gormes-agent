package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// sessionMetadataLister is optionally implemented by session maps that can
// enumerate all stored metadata. Manager type-asserts to this interface
// during auto-resume startup.
type sessionMetadataLister interface {
	ListAllMetadata(ctx context.Context) ([]session.Metadata, error)
}

// autoResumePendingSessions scans session metadata for sessions marked
// ResumePending and either injects a synthetic empty-text submit event
// into inbox to trigger recovery through the normal handleInbound path,
// or marks the session as non-resumable when the source platform has no
// registered channel.
func autoResumeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (m *Manager) autoResumePendingSessions(ctx context.Context, inbox chan<- InboundEvent) {
	ctx = autoResumeContext(ctx)
	lister, ok := m.cfg.SessionMap.(sessionMetadataLister)
	if !ok {
		return
	}

	all, err := lister.ListAllMetadata(ctx)
	if err != nil {
		m.log.Debug("auto-resume: list metadata", "err", err)
		return
	}

	if err := ctx.Err(); err != nil {
		m.log.Debug("auto-resume: context canceled", "err", err)
		return
	}

	scheduled := 0
	for _, meta := range all {
		if err := ctx.Err(); err != nil {
			m.log.Debug("auto-resume: context canceled", "err", err)
			return
		}
		if !meta.ResumePending {
			continue
		}
		meta.Source = strings.ToLower(strings.TrimSpace(meta.Source))
		meta.ChatID = strings.TrimSpace(meta.ChatID)
		meta.UserID = strings.TrimSpace(meta.UserID)
		meta.SessionID = strings.TrimSpace(meta.SessionID)
		if meta.SessionID == "" || unsafeAutoResumeMetadataText(meta.Source) || unsafeAutoResumeMetadataText(meta.ChatID) || unsafeAutoResumeMetadataText(meta.SessionID) {
			continue
		}

		ch := m.lookupChannel(meta.Source)
		if ch == nil {
			m.autoResumeMarkNonResumable(ctx, meta, session.NonResumableAdapterNotReady)
			continue
		}

		chatID := meta.ChatID
		if chatID == "" {
			m.autoResumeMarkNonResumable(ctx, meta, session.NonResumableAdapterNotReady)
			continue
		}

		sessionKey := meta.Source + ":" + chatID
		if err := m.cfg.SessionMap.Put(ctx, sessionKey, meta.SessionID); err != nil {
			m.log.Debug("auto-resume: ensure session mapping", "key", sessionKey, "err", err)
			continue
		}

		ev := InboundEvent{
			Platform:  meta.Source,
			ChatID:    chatID,
			UserID:    meta.UserID,
			Kind:      EventSubmit,
			Text:      "",
			MsgID:     fmt.Sprintf("auto-resume-%s-%d", meta.SessionID, meta.ResumeMarkedAt),
			AccountID: "",
		}

		select {
		case inbox <- ev:
			scheduled++
		default:
			m.log.Debug("auto-resume: inbox full, skipping", "session_id", meta.SessionID)
		}
	}

	if scheduled > 0 {
		m.log.Info("auto-resume: scheduled recovery for interrupted sessions",
			"count", scheduled)
	}
}

func unsafeAutoResumeMetadataText(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || unicode.Is(unicode.Cf, r) {
			return true
		}
	}
	return false
}

func (m *Manager) autoResumeMarkNonResumable(ctx context.Context, meta session.Metadata, reason string) {
	meta.Source = strings.ToLower(strings.TrimSpace(meta.Source))
	meta.ChatID = strings.TrimSpace(meta.ChatID)
	meta.UserID = strings.TrimSpace(meta.UserID)
	meta.SessionID = strings.TrimSpace(meta.SessionID)
	now := m.now()
	writer, ok := m.cfg.SessionMap.(sessionMetadataWriter)
	if !ok {
		return
	}
	if err := writer.PutMetadata(ctx, session.Metadata{
		SessionID:          meta.SessionID,
		Source:             meta.Source,
		ChatID:             meta.ChatID,
		UserID:             meta.UserID,
		NonResumableReason: reason,
		NonResumableAt:     now.Unix(),
		UpdatedAt:          now.Unix(),
	}); err != nil {
		m.log.Debug("auto-resume: mark non-resumable", "session_id", meta.SessionID, "err", err)
		return
	}
	m.clearResumePending(ctx, meta.SessionID)
	m.writeNonResumableEvidence(ctx, RuntimeNonResumableEvidence{
		SessionKey: meta.Source + ":" + meta.ChatID,
		SessionID:  meta.SessionID,
		Source:     meta.Source,
		ChatID:     meta.ChatID,
		UserID:     meta.UserID,
		Reason:     reason,
		At:         now.Format(time.RFC3339Nano),
	})
}
