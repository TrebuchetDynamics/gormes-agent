package mirror

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/address"
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
	platform := address.Platform(target.Platform)
	chatID := address.ID(target.ChatID)
	threadID := address.ID(target.ThreadID)
	userID := address.ID(target.UserID)
	if platform == "" || chatID == "" {
		return session.Metadata{}, false
	}

	matches := make([]session.Metadata, 0, len(candidates))
	for _, meta := range candidates {
		if address.Platform(meta.Source) != platform {
			continue
		}
		if !address.ChatMatches(meta.ChatID, chatID, threadID) {
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
	content := address.ID(target.MessageText)
	if content == "" {
		return DeliveryMirrorResult{Evidence: DeliveryMirrorSessionMissing}, nil
	}
	source := address.ID(target.SourceLabel)
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
	if deliveryMirrorHasDistinctUsers(items) {
		return nil, false
	}
	return items, true
}

func deliveryMirrorHasDistinctUsers(items []session.Metadata) bool {
	seen := map[string]struct{}{}
	for _, meta := range items {
		userID := address.ID(meta.UserID)
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
