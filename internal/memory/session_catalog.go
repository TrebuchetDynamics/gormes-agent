package memory

import (
	"context"
	"database/sql"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/catalog"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

var ErrUserScopeDenied = catalog.ErrUserScopeDenied

type SearchFilter = catalog.SearchFilter

const (
	CrossChatDecisionAllowed  = catalog.CrossChatDecisionAllowed
	CrossChatDecisionDenied   = catalog.CrossChatDecisionDenied
	CrossChatDecisionDegraded = catalog.CrossChatDecisionDegraded
	CrossChatFallbackSameChat = catalog.CrossChatFallbackSameChat
)

type CrossChatSessionEvidence = catalog.CrossChatSessionEvidence
type CrossChatRecallEvidence = catalog.CrossChatRecallEvidence

func ExplainCrossChatRecall(metas []session.Metadata, filter SearchFilter) CrossChatRecallEvidence {
	return catalog.ExplainCrossChatRecall(metas, filter)
}

func DegradedCrossChatRecallEvidence(filter SearchFilter, reason string) CrossChatRecallEvidence {
	return catalog.DegradedCrossChatRecallEvidence(filter, reason)
}

const SearchLineageStatusUnavailable = catalog.SearchLineageStatusUnavailable

type SearchLineage = catalog.SearchLineage
type MessageSearchHit = catalog.MessageSearchHit
type SessionSearchHit = catalog.SessionSearchHit

func SearchMessages(ctx context.Context, db *sql.DB, metas []session.Metadata, filter SearchFilter, limit int) ([]MessageSearchHit, error) {
	return catalog.SearchMessages(ctx, db, metas, filter, limit)
}

func SearchSessions(ctx context.Context, db *sql.DB, metas []session.Metadata, filter SearchFilter, limit int) ([]SessionSearchHit, error) {
	return catalog.SearchSessions(ctx, db, metas, filter, limit)
}
