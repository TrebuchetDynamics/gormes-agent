package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
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
	platform := strings.ToLower(strings.TrimSpace(target.Platform))
	chatID := strings.TrimSpace(target.ChatID)
	threadID := strings.TrimSpace(target.ThreadID)
	userID := strings.TrimSpace(target.UserID)
	if platform == "" || chatID == "" {
		return session.Metadata{}, false
	}

	matches := make([]session.Metadata, 0, len(candidates))
	for _, meta := range candidates {
		if !strings.EqualFold(strings.TrimSpace(meta.Source), platform) {
			continue
		}
		if !deliveryMirrorChatMatches(strings.TrimSpace(meta.ChatID), chatID, threadID) {
			continue
		}
		matches = append(matches, meta)
	}
	if len(matches) == 0 {
		return session.Metadata{}, false
	}

	if userID != "" {
		exact := matches[:0]
		for _, meta := range matches {
			if strings.TrimSpace(meta.UserID) == userID {
				exact = append(exact, meta)
			}
		}
		if len(exact) > 0 {
			matches = exact
		} else if len(matches) > 1 {
			return session.Metadata{}, false
		}
	} else if deliveryMirrorHasDistinctUsers(matches) {
		return session.Metadata{}, false
	}

	best := matches[0]
	for _, meta := range matches[1:] {
		if meta.UpdatedAt > best.UpdatedAt || (meta.UpdatedAt == best.UpdatedAt && meta.SessionID < best.SessionID) {
			best = meta
		}
	}
	return best, true
}

func MirrorDeliveryToSession(ctx context.Context, st store.Store, candidates []session.Metadata, target DeliveryMirrorTarget, now time.Time) (DeliveryMirrorResult, error) {
	if st == nil {
		return DeliveryMirrorResult{Evidence: DeliveryMirrorStoreUnavailable}, nil
	}
	selected, ok := SelectDeliveryMirrorSession(candidates, target)
	if !ok {
		return DeliveryMirrorResult{Evidence: DeliveryMirrorSessionMissing}, nil
	}
	content := strings.TrimSpace(target.MessageText)
	if content == "" {
		return DeliveryMirrorResult{Evidence: DeliveryMirrorSessionMissing}, nil
	}
	source := strings.TrimSpace(target.SourceLabel)
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

func deliveryMirrorChatMatches(candidate, chatID, threadID string) bool {
	if threadID == "" {
		return candidate == chatID
	}
	return candidate == chatID+":"+threadID
}

func deliveryMirrorHasDistinctUsers(items []session.Metadata) bool {
	seen := map[string]struct{}{}
	for _, meta := range items {
		userID := strings.TrimSpace(meta.UserID)
		if userID == "" {
			continue
		}
		seen[userID] = struct{}{}
		if len(seen) > 1 {
			return true
		}
	}
	return false
}
