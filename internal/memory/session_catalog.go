package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

var ErrUserScopeDenied = errors.New("memory: user scope denied")

// SearchFilter narrows cross-session search to one canonical user and an
// optional set of transport sources.
type SearchFilter struct {
	UserID           string
	Sources          []string
	SessionIDs       []string
	Query            string
	CurrentSessionID string
	CurrentChatKey   string
}

const (
	// CrossChatDecisionAllowed means a user-scoped recall request had enough
	// canonical identity evidence to widen beyond the current chat/session.
	CrossChatDecisionAllowed = "allowed"
	// CrossChatDecisionDenied means widening was refused and callers must use
	// the same-chat fallback scope.
	CrossChatDecisionDenied = "denied"
	// CrossChatDecisionDegraded means the operator surface could not inspect a
	// dependency needed for user-scoped widening.
	CrossChatDecisionDegraded = "degraded"

	CrossChatFallbackSameChat = "same-chat"
)

// CrossChatSessionEvidence is the operator-readable identity for one session
// considered by a user-scoped cross-chat recall request.
type CrossChatSessionEvidence struct {
	SessionID string `json:"session_id"`
	Source    string `json:"source,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	ChatKey   string `json:"chat_key,omitempty"`
	Current   bool   `json:"current,omitempty"`
}

// CrossChatRecallEvidence explains why a user-scoped recall/search request was
// allowed, denied, or degraded before the query result set is rendered.
type CrossChatRecallEvidence struct {
	Decision                  string                     `json:"decision"`
	Scope                     string                     `json:"scope"`
	FallbackScope             string                     `json:"fallback_scope,omitempty"`
	Reason                    string                     `json:"reason,omitempty"`
	UserID                    string                     `json:"user_id,omitempty"`
	CurrentSessionID          string                     `json:"current_session_id,omitempty"`
	CurrentChatKey            string                     `json:"current_chat_key,omitempty"`
	CurrentBinding            *CrossChatSessionEvidence  `json:"current_binding,omitempty"`
	SourceAllowlist           []string                   `json:"source_allowlist,omitempty"`
	SessionsConsidered        int                        `json:"sessions_considered"`
	WidenedSessionsConsidered int                        `json:"widened_sessions_considered"`
	Sessions                  []CrossChatSessionEvidence `json:"sessions,omitempty"`
}

// ExplainCrossChatRecall mirrors the session catalog scope selection logic and
// returns the evidence operators need to audit a user-scoped widening decision.
func ExplainCrossChatRecall(metas []session.Metadata, filter SearchFilter) CrossChatRecallEvidence {
	evidence := CrossChatRecallEvidence{
		Scope:            "user",
		UserID:           strings.TrimSpace(filter.UserID),
		CurrentSessionID: strings.TrimSpace(filter.CurrentSessionID),
		CurrentChatKey:   strings.TrimSpace(filter.CurrentChatKey),
		SourceAllowlist:  normalizeSources(filter.Sources),
	}
	if evidence.UserID == "" {
		return deniedCrossChatEvidence(evidence, "unresolved user_id; same-chat fallback scope used")
	}

	requireCurrentBinding := evidence.CurrentSessionID != "" || evidence.CurrentChatKey != ""
	currentBindingMatched := !requireCurrentBinding
	for _, meta := range metas {
		if !metadataMatchesCurrent(meta, evidence.CurrentSessionID, evidence.CurrentChatKey) {
			continue
		}
		binding := crossChatSessionEvidence(meta, true)
		evidence.CurrentBinding = &binding
		metaUserID := strings.TrimSpace(meta.UserID)
		if metaUserID == "" {
			return deniedCrossChatEvidence(evidence, "unresolved current binding: current chat/session has no user_id; same-chat fallback scope used")
		}
		if metaUserID != evidence.UserID {
			return deniedCrossChatEvidence(evidence, fmt.Sprintf("conflicting current binding: current binding belongs to %q; same-chat fallback scope used", metaUserID))
		}
		currentBindingMatched = true
	}
	if !currentBindingMatched {
		return deniedCrossChatEvidence(evidence, "unknown current binding: current chat/session is not bound to the requested user_id; same-chat fallback scope used")
	}

	allowedSources := normalizeSources(filter.Sources)
	allowedSessions := normalizeSessionIDs(filter.SessionIDs)
	for _, meta := range metas {
		metaUserID := strings.TrimSpace(meta.UserID)
		if metaUserID != evidence.UserID {
			continue
		}
		if len(allowedSources) > 0 && !slices.Contains(allowedSources, strings.ToLower(strings.TrimSpace(meta.Source))) {
			continue
		}
		if len(allowedSessions) > 0 && !slices.Contains(allowedSessions, strings.TrimSpace(meta.SessionID)) {
			continue
		}
		item := crossChatSessionEvidence(meta, metadataMatchesCurrent(meta, evidence.CurrentSessionID, evidence.CurrentChatKey))
		evidence.Sessions = append(evidence.Sessions, item)
		if !item.Current {
			evidence.WidenedSessionsConsidered++
		}
	}
	if len(evidence.Sessions) == 0 {
		return deniedCrossChatEvidence(evidence, fmt.Sprintf("unresolved user binding: no sessions matched user_id %q and source/session filters; same-chat fallback scope used", evidence.UserID))
	}

	evidence.Decision = CrossChatDecisionAllowed
	evidence.Reason = "resolved user_id and current binding; user-scope recall may widen across listed sessions"
	evidence.SessionsConsidered = len(evidence.Sessions)
	return evidence
}

// DegradedCrossChatRecallEvidence reports that a caller asked for user scope
// but the diagnostic path could not inspect the session binding dependency.
func DegradedCrossChatRecallEvidence(filter SearchFilter, reason string) CrossChatRecallEvidence {
	evidence := CrossChatRecallEvidence{
		Decision:         CrossChatDecisionDegraded,
		Scope:            "user",
		FallbackScope:    CrossChatFallbackSameChat,
		Reason:           strings.TrimSpace(reason),
		UserID:           strings.TrimSpace(filter.UserID),
		CurrentSessionID: strings.TrimSpace(filter.CurrentSessionID),
		CurrentChatKey:   strings.TrimSpace(filter.CurrentChatKey),
		SourceAllowlist:  normalizeSources(filter.Sources),
	}
	if evidence.Reason == "" {
		evidence.Reason = "cross-chat recall evidence unavailable; same-chat fallback scope used"
	}
	return evidence
}

func deniedCrossChatEvidence(evidence CrossChatRecallEvidence, reason string) CrossChatRecallEvidence {
	evidence.Decision = CrossChatDecisionDenied
	evidence.FallbackScope = CrossChatFallbackSameChat
	evidence.Reason = reason
	evidence.Sessions = nil
	evidence.SessionsConsidered = 0
	evidence.WidenedSessionsConsidered = 0
	return evidence
}

func crossChatSessionEvidence(meta session.Metadata, current bool) CrossChatSessionEvidence {
	item := CrossChatSessionEvidence{
		SessionID: strings.TrimSpace(meta.SessionID),
		Source:    strings.TrimSpace(meta.Source),
		ChatID:    strings.TrimSpace(meta.ChatID),
		Current:   current,
	}
	item.ChatKey = canonicalChatKey(meta)
	return item
}

// SearchLineageStatusUnavailable means the hit matched through a chat key but
// there was no session-specific metadata row to prove a lineage chain.
const SearchLineageStatusUnavailable = "unavailable"

// SearchLineage is the lineage evidence attached to one matched session.
type SearchLineage struct {
	ParentSessionID string
	LineageKind     string
	ChildSessionIDs []string
	Status          string
}

// MessageSearchHit is one turn-level result from the session catalog.
type MessageSearchHit struct {
	SessionID string
	ChatID    string
	Source    string
	Role      string
	Content   string
	TSUnix    int64
	Lineage   SearchLineage
}

// SessionSearchHit is one session-level result ordered by latest matching turn.
type SessionSearchHit struct {
	SessionID      string
	ChatID         string
	Source         string
	LatestTurnUnix int64
	Lineage        SearchLineage
}

// SearchMessages returns matching turns across the canonical sessions bound to
// one user, optionally narrowed to a subset of sources.
func SearchMessages(ctx context.Context, db *sql.DB, metas []session.Metadata, filter SearchFilter, limit int) ([]MessageSearchHit, error) {
	selected, err := selectMetadata(metas, filter)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 || limit == 0 {
		return nil, nil
	}

	sessionIDs, chatKeys, metaBySession, metaByChat := metadataIndexes(selected)
	lineage := buildSearchLineageIndex(selected)
	query, args := buildTurnSearchQuery(filter.Query, sessionIDs, chatKeys, limit, false)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("session catalog: search messages: %w", err)
	}
	defer rows.Close()

	var hits []MessageSearchHit
	for rows.Next() {
		var hit MessageSearchHit
		if err := rows.Scan(&hit.SessionID, &hit.ChatID, &hit.Role, &hit.Content, &hit.TSUnix); err != nil {
			return nil, fmt.Errorf("session catalog: scan message hit: %w", err)
		}
		if meta, ok := metaBySession[hit.SessionID]; ok {
			hit.Source = meta.Source
		} else if meta, ok := metaByChat[hit.ChatID]; ok {
			hit.Source = meta.Source
		}
		hit.Lineage = lineage.contextFor(hit.SessionID)
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("session catalog: iterate message hits: %w", err)
	}
	return hits, nil
}

// SearchSessions returns one row per matching session ordered by latest turn.
func SearchSessions(ctx context.Context, db *sql.DB, metas []session.Metadata, filter SearchFilter, limit int) ([]SessionSearchHit, error) {
	selected, err := selectMetadata(metas, filter)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 || limit == 0 {
		return nil, nil
	}

	sessionIDs, chatKeys, metaBySession, metaByChat := metadataIndexes(selected)
	lineage := buildSearchLineageIndex(selected)
	query, args := buildTurnSearchQuery(filter.Query, sessionIDs, chatKeys, limit, true)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("session catalog: search sessions: %w", err)
	}
	defer rows.Close()

	var hits []SessionSearchHit
	for rows.Next() {
		var hit SessionSearchHit
		if err := rows.Scan(&hit.SessionID, &hit.ChatID, &hit.LatestTurnUnix); err != nil {
			return nil, fmt.Errorf("session catalog: scan session hit: %w", err)
		}
		if meta, ok := metaBySession[hit.SessionID]; ok {
			hit.Source = meta.Source
		} else if meta, ok := metaByChat[hit.ChatID]; ok {
			hit.Source = meta.Source
		}
		hit.Lineage = lineage.contextFor(hit.SessionID)
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("session catalog: iterate session hits: %w", err)
	}
	return hits, nil
}

func selectMetadata(metas []session.Metadata, filter SearchFilter) ([]session.Metadata, error) {
	userID := strings.TrimSpace(filter.UserID)
	if userID == "" {
		return nil, nil
	}
	currentSessionID := strings.TrimSpace(filter.CurrentSessionID)
	currentChatKey := strings.TrimSpace(filter.CurrentChatKey)
	requireCurrentBinding := currentSessionID != "" || currentChatKey != ""
	currentBindingMatched := !requireCurrentBinding

	allowedSources := normalizeSources(filter.Sources)
	allowedSessions := normalizeSessionIDs(filter.SessionIDs)
	selected := make([]session.Metadata, 0, len(metas))
	for _, meta := range metas {
		metaUserID := strings.TrimSpace(meta.UserID)
		if metadataMatchesCurrent(meta, currentSessionID, currentChatKey) {
			if metaUserID != userID {
				return nil, fmt.Errorf("%w: current binding belongs to %q", ErrUserScopeDenied, metaUserID)
			}
			currentBindingMatched = true
		}
		if metaUserID != userID {
			continue
		}
		if len(allowedSources) > 0 && !slices.Contains(allowedSources, strings.ToLower(strings.TrimSpace(meta.Source))) {
			continue
		}
		if len(allowedSessions) > 0 && !slices.Contains(allowedSessions, strings.TrimSpace(meta.SessionID)) {
			continue
		}
		selected = append(selected, meta)
	}
	if !currentBindingMatched {
		return nil, ErrUserScopeDenied
	}
	return selected, nil
}

func metadataMatchesCurrent(meta session.Metadata, currentSessionID, currentChatKey string) bool {
	if currentSessionID != "" && strings.TrimSpace(meta.SessionID) == currentSessionID {
		return true
	}
	return currentChatKey != "" && sameChatKey(canonicalChatKey(meta), currentChatKey)
}

func normalizeSources(sources []string) []string {
	if len(sources) == 0 {
		return nil
	}
	out := make([]string, 0, len(sources))
	for _, src := range sources {
		src = strings.ToLower(strings.TrimSpace(src))
		if src == "" || slices.Contains(out, src) {
			continue
		}
		out = append(out, src)
	}
	return out
}

func normalizeSessionIDs(sessionIDs []string) []string {
	if len(sessionIDs) == 0 {
		return nil
	}
	out := make([]string, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			continue
		}
		if sessionID == "*" {
			return nil
		}
		if !slices.Contains(out, sessionID) {
			out = append(out, sessionID)
		}
	}
	return out
}

func metadataIndexes(metas []session.Metadata) ([]string, []string, map[string]session.Metadata, map[string]session.Metadata) {
	sessionIDs := make([]string, 0, len(metas))
	chatKeys := make([]string, 0, len(metas))
	metaBySession := make(map[string]session.Metadata, len(metas))
	metaByChat := make(map[string]session.Metadata, len(metas))
	for _, meta := range metas {
		if sessionID := strings.TrimSpace(meta.SessionID); sessionID != "" {
			sessionIDs = append(sessionIDs, sessionID)
			metaBySession[sessionID] = meta
		}
		if chatKey := canonicalChatKey(meta); chatKey != "" {
			chatKeys = append(chatKeys, chatKey)
			metaByChat[chatKey] = meta
		}
	}
	return sessionIDs, chatKeys, metaBySession, metaByChat
}

func canonicalChatKey(meta session.Metadata) string {
	source := strings.TrimSpace(meta.Source)
	chatID := strings.TrimSpace(meta.ChatID)
	if source == "" || chatID == "" {
		return ""
	}
	return source + ":" + chatID
}

type searchLineageIndex struct {
	bySession map[string]session.Metadata
	children  map[string][]string
}

func buildSearchLineageIndex(metas []session.Metadata) searchLineageIndex {
	idx := searchLineageIndex{
		bySession: make(map[string]session.Metadata, len(metas)),
		children:  make(map[string][]string, len(metas)),
	}
	for _, meta := range metas {
		meta = normalizeSearchLineageMetadata(meta)
		if meta.SessionID == "" {
			continue
		}
		idx.bySession[meta.SessionID] = meta
	}
	for _, meta := range idx.bySession {
		if meta.ParentSessionID == "" {
			continue
		}
		childIDs := idx.children[meta.ParentSessionID]
		if !slices.Contains(childIDs, meta.SessionID) {
			idx.children[meta.ParentSessionID] = append(childIDs, meta.SessionID)
		}
	}
	for parentID := range idx.children {
		slices.Sort(idx.children[parentID])
	}
	return idx
}

func normalizeSearchLineageMetadata(meta session.Metadata) session.Metadata {
	meta.SessionID = strings.TrimSpace(meta.SessionID)
	meta.ParentSessionID = strings.TrimSpace(meta.ParentSessionID)
	meta.LineageKind = strings.ToLower(strings.TrimSpace(meta.LineageKind))
	return meta
}

func (idx searchLineageIndex) contextFor(sessionID string) SearchLineage {
	sessionID = strings.TrimSpace(sessionID)
	meta, ok := idx.bySession[sessionID]
	if !ok {
		return SearchLineage{Status: SearchLineageStatusUnavailable}
	}
	children := append([]string(nil), idx.children[sessionID]...)
	return SearchLineage{
		ParentSessionID: meta.ParentSessionID,
		LineageKind:     searchLineageKind(meta),
		ChildSessionIDs: children,
		Status:          idx.statusFor(sessionID),
	}
}

func searchLineageKind(meta session.Metadata) string {
	if meta.LineageKind == "" {
		return session.LineageKindPrimary
	}
	return meta.LineageKind
}

func (idx searchLineageIndex) statusFor(sessionID string) string {
	meta, ok := idx.bySession[sessionID]
	if !ok {
		return SearchLineageStatusUnavailable
	}
	seen := map[string]struct{}{sessionID: {}}
	for current := meta.ParentSessionID; current != ""; {
		if _, ok := seen[current]; ok {
			return session.LineageStatusLoop
		}
		seen[current] = struct{}{}

		parent, ok := idx.bySession[current]
		if !ok {
			return session.LineageStatusOrphan
		}
		current = parent.ParentSessionID
	}
	return session.LineageStatusOK
}

func buildTurnSearchQuery(rawQuery string, sessionIDs, chatKeys []string, limit int, sessionsOnly bool) (string, []any) {
	var b strings.Builder
	args := make([]any, 0, len(sessionIDs)+len(chatKeys)+2)
	if sessionsOnly {
		b.WriteString(`SELECT t.session_id, t.chat_id, MAX(t.ts_unix) AS latest_turn_unix FROM turns t`)
	} else {
		b.WriteString(`SELECT t.session_id, t.chat_id, t.role, t.content, t.ts_unix FROM turns t`)
	}

	query := sanitizeFTS5Pattern(rawQuery)
	if query != "" {
		b.WriteString(` JOIN turns_fts fts ON fts.rowid = t.id WHERE turns_fts MATCH ?`)
		args = append(args, query)
	} else {
		b.WriteString(` WHERE 1=1`)
	}

	b.WriteString(` AND (`)
	appendInClause(&b, "t.session_id", sessionIDs, &args)
	if len(chatKeys) > 0 {
		b.WriteString(` OR `)
		appendInClause(&b, "t.chat_id", chatKeys, &args)
	}
	b.WriteString(`)`)
	b.WriteString(` AND t.memory_sync_status = 'ready'`)

	if sessionsOnly {
		b.WriteString(` GROUP BY t.session_id, t.chat_id ORDER BY latest_turn_unix DESC, t.session_id ASC LIMIT ?`)
	} else {
		b.WriteString(` ORDER BY t.ts_unix DESC, t.id DESC LIMIT ?`)
	}
	args = append(args, limit)
	return b.String(), args
}

func appendInClause(b *strings.Builder, column string, values []string, args *[]any) {
	b.WriteString(column)
	b.WriteString(` IN (`)
	for i, value := range values {
		if i > 0 {
			b.WriteString(`,`)
		}
		b.WriteString(`?`)
		*args = append(*args, value)
	}
	b.WriteString(`)`)
}
