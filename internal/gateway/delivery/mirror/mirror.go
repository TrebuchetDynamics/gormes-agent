package mirror

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/address"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

const (
	DeliveryMirrorSessionMissing   = "delivery_mirror_session_missing"
	DeliveryMirrorStoreUnavailable = "delivery_mirror_store_unavailable"
)

type DeliveryMirrorTarget struct {
	Platform    string
	ChatID      string
	ThreadID    string
	UserID      string
	MessageText string
	SourceLabel string
}

type DeliveryMirrorResult struct {
	Mirrored  bool
	SessionID string
	Evidence  string
}

func SelectDeliveryMirrorSession(candidates []session.Metadata, target DeliveryMirrorTarget) (session.Metadata, bool) {
	platform := address.Platform(target.Platform)
	chatID := address.ID(target.ChatID)
	threadID := address.ID(target.ThreadID)
	userID := address.ID(target.UserID)
	if platform == "" || chatID == "" || containsMirrorControlRune(platform) || containsMirrorControlRune(chatID) || containsMirrorControlRune(threadID) || containsMirrorControlRune(userID) {
		return session.Metadata{}, false
	}

	matches := make([]session.Metadata, 0, len(candidates))
	for _, meta := range candidates {
		sessionID := address.ID(meta.SessionID)
		metaSource := address.Platform(meta.Source)
		metaChatID := address.ID(meta.ChatID)
		metaUserID := address.ID(meta.UserID)
		if sessionID == "" || containsMirrorControlRune(sessionID) || containsMirrorControlRune(metaSource) || containsMirrorControlRune(metaChatID) || containsMirrorControlRune(metaUserID) {
			continue
		}
		if metaSource != platform {
			continue
		}
		if !address.ChatMatches(metaChatID, chatID, threadID) {
			continue
		}
		matches = append(matches, meta)
	}
	if len(matches) == 0 {
		return session.Metadata{}, false
	}

	var ok bool
	matches, ok = deliveryMirrorFilterByUser(matches, userID)
	if !ok {
		return session.Metadata{}, false
	}

	best := matches[0]
	for _, meta := range matches[1:] {
		if meta.UpdatedAt > best.UpdatedAt || (meta.UpdatedAt == best.UpdatedAt && meta.SessionID < best.SessionID) {
			best = meta
		}
	}
	return normalizeSelectedMetadata(best), true
}

func containsMirrorControlRune(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return true
		}
	}
	return false
}

func normalizeSelectedMetadata(meta session.Metadata) session.Metadata {
	meta.SessionID = address.ID(meta.SessionID)
	meta.Source = address.Platform(meta.Source)
	meta.ChatID = address.ID(meta.ChatID)
	meta.UserID = address.ID(meta.UserID)
	return meta
}

func MirrorDeliveryToSession(ctx context.Context, st store.Store, candidates []session.Metadata, target DeliveryMirrorTarget, now time.Time) (DeliveryMirrorResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return DeliveryMirrorResult{}, err
	}
	if st == nil {
		return DeliveryMirrorResult{Evidence: DeliveryMirrorStoreUnavailable}, nil
	}
	selected, ok := SelectDeliveryMirrorSession(candidates, target)
	if !ok {
		return DeliveryMirrorResult{Evidence: DeliveryMirrorSessionMissing}, nil
	}
	content := target.MessageText
	if strings.TrimSpace(content) == "" {
		return DeliveryMirrorResult{Evidence: DeliveryMirrorSessionMissing}, nil
	}
	source := sanitizeMirrorSourceLabel(target.SourceLabel)
	if source == "" {
		source = "gormes"
	}
	meta, err := json.Marshal(map[string]any{
		"mirror":        true,
		"mirror_source": source,
	})
	if err != nil {
		return DeliveryMirrorResult{}, err
	}
	payload, err := json.Marshal(map[string]any{
		"session_id": selected.SessionID,
		"content":    content,
		"ts_unix":    now.Unix(),
		"chat_id":    selected.ChatID,
		"meta_json":  string(meta),
	})
	if err != nil {
		return DeliveryMirrorResult{}, err
	}
	if _, err := st.Exec(ctx, store.Command{Kind: store.FinalizeAssistantTurn, Payload: payload}); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return DeliveryMirrorResult{}, err
		}
		return DeliveryMirrorResult{}, err
	}
	return DeliveryMirrorResult{Mirrored: true, SessionID: selected.SessionID}, nil
}

func sanitizeMirrorSourceLabel(value string) string {
	value = address.ID(value)
	value = redaction.RedactSecrets(value)
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	fields := strings.Fields(b.String())
	for i, field := range fields {
		lower := strings.ToLower(field)
		if strings.Contains(lower, "[redacted]") && mirrorSecretField(lower) {
			fields[i] = "[redacted]"
		}
	}
	return strings.Join(fields, " ")
}

func mirrorSecretField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
}

func deliveryMirrorFilterByUser(items []session.Metadata, userID string) ([]session.Metadata, bool) {
	if userID != "" {
		exact := items[:0]
		for _, meta := range items {
			if address.ID(meta.UserID) == userID {
				exact = append(exact, meta)
			}
		}
		return exact, len(exact) > 0
	}
	if deliveryMirrorHasAmbiguousUserProvenance(items) {
		return nil, false
	}
	return items, true
}

func deliveryMirrorHasAmbiguousUserProvenance(items []session.Metadata) bool {
	knownUsers := map[string]struct{}{}
	unknownUsers := 0
	for _, meta := range items {
		userID := address.ID(meta.UserID)
		if userID == "" {
			unknownUsers++
			continue
		}
		knownUsers[userID] = struct{}{}
		if len(knownUsers) > 1 {
			return true
		}
	}
	return unknownUsers > 0 && (len(knownUsers) > 0 || unknownUsers > 1)
}
