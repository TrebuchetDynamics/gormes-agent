package sessionadapters

import (
	"context"
	"errors"
	"strings"
	"time"

	appsession "github.com/TrebuchetDynamics/gormes-agent/internal/app/session"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

const sessionTreeLimit = 200

func NewSessionTreeFunc(rootCtx context.Context, metadata *sessionpkg.BoltMap) tui.SessionTreeFunc {
	return func(ctx context.Context, req tui.SessionTreeRequest) (tui.SessionTreeResult, error) {
		ctx = contextWithRootFallback(rootCtx, ctx)
		db, err := appsession.OpenSessionDirectoryDB()
		if err != nil {
			if strings.Contains(err.Error(), "memory database not found") {
				return tui.SessionTreeResult{}, nil
			}
			return tui.SessionTreeResult{}, err
		}
		defer db.Close()

		var metas []sessionpkg.Metadata
		var ledgerMeta sessionpkg.LedgerMetadataReader
		if metadata != nil {
			ledgerMeta = metadata
			metas, err = metadata.ListAllMetadata(ctx)
			if err != nil {
				return tui.SessionTreeResult{}, err
			}
		}
		seen := map[string]struct{}{}
		ledgers := map[string]sessionpkg.SessionLedger{}
		directory, err := sessionpkg.ListDirectorySessions(ctx, db, sessionpkg.DirectoryFilter{Limit: sessionTreeLimit})
		if err != nil {
			return tui.SessionTreeResult{}, err
		}
		for _, entry := range directory {
			id := strings.TrimSpace(entry.ID)
			if id == "" {
				continue
			}
			seen[id] = struct{}{}
			ledger, err := sessionpkg.ReadSessionLedger(ctx, db, ledgerMeta, id)
			if err != nil {
				if errors.Is(err, sessionpkg.ErrSessionNotFound) {
					continue
				}
				return tui.SessionTreeResult{}, err
			}
			ledgers[id] = ledger
		}
		for _, meta := range metas {
			id := strings.TrimSpace(meta.SessionID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
		}

		tree := sessionpkg.BuildSessionTree(metas, ledgers, sessionpkg.TreeOptions{ActiveSessionID: req.ActiveSessionID, Filter: sessionpkg.TreeFilter(req.Filter)})
		return sessionTreeResult(tree), nil
	}
}

func NewSessionTreeLabelFunc(rootCtx context.Context, metadata *sessionpkg.BoltMap) tui.SessionTreeLabelFunc {
	if metadata == nil {
		return nil
	}
	return func(ctx context.Context, req tui.SessionTreeLabelRequest) (tui.SessionTreeLabelResult, error) {
		ctx = contextWithRootFallback(rootCtx, ctx)
		sessionID := strings.TrimSpace(req.SessionID)
		meta, _, err := metadata.GetMetadata(ctx, sessionID)
		if err != nil {
			return tui.SessionTreeLabelResult{}, err
		}
		labels := append([]string(nil), meta.Labels...)
		label := strings.TrimSpace(req.Label)
		switch strings.ToLower(strings.TrimSpace(req.Action)) {
		case "set":
			if label != "" && !containsFoldString(labels, label) {
				labels = append(labels, label)
			}
		case "clear":
			if label == "" {
				labels = nil
			} else {
				labels = removeFoldString(labels, label)
			}
		}
		labels, err = metadata.SetLabels(ctx, sessionID, labels, time.Now())
		if err != nil {
			return tui.SessionTreeLabelResult{}, err
		}
		return tui.SessionTreeLabelResult{SessionID: sessionID, Labels: labels}, nil
	}
}

func NewSessionTreeRestoreFunc(rootCtx context.Context) tui.SessionTreeRestoreFunc {
	return func(ctx context.Context, req tui.SessionTreeRestoreRequest) (tui.SessionTreeRestoreResult, error) {
		ctx = contextWithRootFallback(rootCtx, ctx)
		db, err := appsession.OpenSessionDirectoryDB()
		if err != nil {
			return tui.SessionTreeRestoreResult{}, err
		}
		defer db.Close()
		ledger, err := sessionpkg.ReadSessionLedger(ctx, db, nil, req.SessionID)
		if err != nil {
			return tui.SessionTreeRestoreResult{}, err
		}
		text, evidence := sessionpkg.ReplayPromptFromLedger(ledger, req.MessageID)
		return tui.SessionTreeRestoreResult{Text: text, Editable: evidence == "", Evidence: evidence}, nil
	}
}

func sessionTreeResult(tree sessionpkg.SessionTree) tui.SessionTreeResult {
	out := tui.SessionTreeResult{Filter: string(tree.Filter), ActiveSessionID: tree.ActiveSessionID}
	out.Entries = make([]tui.SessionTreeEntry, 0, len(tree.Entries))
	for _, entry := range tree.Entries {
		outEntry := tui.SessionTreeEntry{
			ID:          entry.SessionID,
			ParentID:    entry.ParentSessionID,
			LineageKind: entry.LineageKind,
			Title:       entry.Title,
			Labels:      append([]string(nil), entry.Labels...),
			UpdatedAt:   entry.UpdatedAt,
			Depth:       entry.Depth,
			Active:      entry.Active,
			Status:      entry.Status,
		}
		for _, msg := range entry.Messages {
			outEntry.Messages = append(outEntry.Messages, tui.SessionTreeMessage{
				ID:       msg.ID,
				Role:     msg.Role,
				Content:  msg.Content,
				Evidence: msg.ReplayUnavailable,
			})
		}
		out.Entries = append(out.Entries, outEntry)
	}
	return out
}

func contextWithRootFallback(rootCtx, ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	if rootCtx != nil {
		return rootCtx
	}
	return context.Background()
}

func containsFoldString(values []string, value string) bool {
	for _, candidate := range values {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func removeFoldString(values []string, value string) []string {
	out := values[:0]
	for _, candidate := range values {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(value)) {
			continue
		}
		out = append(out, candidate)
	}
	return out
}
