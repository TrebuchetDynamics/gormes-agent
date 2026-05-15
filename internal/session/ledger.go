package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	TimestampEvidenceUnknownLegacy = "timestamp_unknown_legacy"
	AttachmentEvidenceRedactedTemp = "redacted_non_durable_attachment_ref"
)

// LedgerMetadataReader is the optional session metadata boundary used to add
// session-level timestamps, sender fallbacks, and reset-generation boundaries.
type LedgerMetadataReader interface {
	ListAllMetadata(ctx context.Context) ([]Metadata, error)
}

// SessionLedger is the API-style read model for a persisted transcript
// generation. It is derived from the append-only turns table, not from
// session.updated.
type SessionLedger struct {
	SessionID       string
	CreatedAtUnix   int64
	UpdatedAtUnix   int64
	Messages        []SessionLedgerMessage
	ResetBoundaries []SessionLedgerResetBoundary
}

type SessionLedgerMessage struct {
	ID                int64
	SessionID         string
	Role              string
	Content           string
	CreatedAtUnix     int64
	CreatedAtKnown    bool
	TimestampEvidence string
	ChatID            string
	Source            string
	SenderID          string
	SenderName        string
	Attachments       []SessionLedgerAttachmentRef
}

type SessionLedgerAttachmentRef struct {
	Kind      string
	URL       string
	MediaType string
	FileName  string
	SourceID  string
	Durable   bool
	Evidence  string
}

type SessionLedgerResetBoundary struct {
	BeforeSessionID string
	AfterSessionID  string
	Kind            string
	Reason          string
	CreatedAtUnix   int64
}

type ledgerTurn struct {
	ID        int64
	SessionID string
	Role      string
	Content   string
	TSUnix    int64
	ChatID    string
	MetaJSON  string
}

type ledgerTurnMeta struct {
	Sender      ledgerSender           `json:"sender"`
	SenderID    string                 `json:"sender_id"`
	SenderName  string                 `json:"sender_name"`
	Attachments []ledgerAttachmentMeta `json:"attachments"`
}

type ledgerSender struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

type ledgerAttachmentMeta struct {
	Kind           string `json:"kind"`
	URL            string `json:"url"`
	MediaType      string `json:"media_type"`
	MediaTypeCamel string `json:"mediaType"`
	FileName       string `json:"file_name"`
	FileNameCamel  string `json:"fileName"`
	SourceID       string `json:"source_id"`
	SourceIDCamel  string `json:"sourceId"`
}

// ReadSessionLedger returns a session's persisted transcript with per-message
// timestamps and metadata. Missing legacy timestamps degrade visibly instead of
// inheriting the session.updated timestamp.
func ReadSessionLedger(ctx context.Context, db *sql.DB, meta LedgerMetadataReader, sessionID string) (SessionLedger, error) {
	if err := ctx.Err(); err != nil {
		return SessionLedger{}, err
	}
	if db == nil {
		return SessionLedger{}, errors.New("session: ledger db is nil")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionLedger{}, ErrSessionNotFound
	}

	turns, err := loadLedgerTurns(ctx, db, sessionID)
	if err != nil {
		return SessionLedger{}, err
	}
	if len(turns) == 0 {
		return SessionLedger{}, ErrSessionNotFound
	}

	var allMeta []Metadata
	var sessionMeta Metadata
	if meta != nil {
		allMeta, err = meta.ListAllMetadata(ctx)
		if err != nil {
			return SessionLedger{}, fmt.Errorf("session: ledger metadata: %w", err)
		}
		for _, item := range allMeta {
			if item.SessionID == sessionID {
				sessionMeta = item
				break
			}
		}
	}

	ledger := SessionLedger{
		SessionID:     sessionID,
		CreatedAtUnix: sessionMeta.CreatedAt,
		UpdatedAtUnix: sessionMeta.UpdatedAt,
	}
	for _, turn := range turns {
		msg, err := ledgerMessageFromTurn(turn, sessionMeta)
		if err != nil {
			return SessionLedger{}, err
		}
		ledger.Messages = append(ledger.Messages, msg)
		if ledger.CreatedAtUnix == 0 && msg.CreatedAtKnown {
			ledger.CreatedAtUnix = msg.CreatedAtUnix
		}
		if msg.CreatedAtKnown && msg.CreatedAtUnix > ledger.UpdatedAtUnix {
			ledger.UpdatedAtUnix = msg.CreatedAtUnix
		}
	}
	ledger.ResetBoundaries = ledgerResetBoundaries(sessionID, allMeta)
	return ledger, nil
}

func loadLedgerTurns(ctx context.Context, db *sql.DB, sessionID string) ([]ledgerTurn, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, session_id, role, content, ts_unix, COALESCE(chat_id, ''), COALESCE(meta_json, '')
		FROM turns
		WHERE session_id = ?
		ORDER BY ts_unix ASC, id ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session: query ledger %q: %w", sessionID, err)
	}
	defer rows.Close()

	var out []ledgerTurn
	for rows.Next() {
		var row ledgerTurn
		if err := rows.Scan(&row.ID, &row.SessionID, &row.Role, &row.Content, &row.TSUnix, &row.ChatID, &row.MetaJSON); err != nil {
			return nil, fmt.Errorf("session: scan ledger %q: %w", sessionID, err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("session: iterate ledger %q: %w", sessionID, err)
	}
	return out, nil
}

func ledgerMessageFromTurn(turn ledgerTurn, sessionMeta Metadata) (SessionLedgerMessage, error) {
	msg := SessionLedgerMessage{
		ID:             turn.ID,
		SessionID:      strings.TrimSpace(turn.SessionID),
		Role:           strings.TrimSpace(turn.Role),
		Content:        turn.Content,
		CreatedAtUnix:  turn.TSUnix,
		ChatID:         strings.TrimSpace(turn.ChatID),
		Source:         sourceFromDirectoryChatID(turn.ChatID),
		CreatedAtKnown: turn.TSUnix > 0,
	}
	if !msg.CreatedAtKnown {
		msg.CreatedAtUnix = 0
		msg.TimestampEvidence = TimestampEvidenceUnknownLegacy
	}

	meta, err := parseLedgerTurnMeta(turn.MetaJSON)
	if err != nil {
		return SessionLedgerMessage{}, fmt.Errorf("session: decode ledger meta for turn %d: %w", turn.ID, err)
	}
	msg.SenderID = firstNonEmpty(meta.Sender.ID, meta.Sender.UserID, meta.SenderID)
	msg.SenderName = firstNonEmpty(meta.Sender.Name, meta.Sender.UserName, meta.SenderName)
	if msg.Role == "user" && msg.SenderID == "" {
		msg.SenderID = strings.TrimSpace(sessionMeta.UserID)
	}
	for _, attachment := range meta.Attachments {
		msg.Attachments = append(msg.Attachments, ledgerAttachmentRef(attachment))
	}
	return msg, nil
}

func parseLedgerTurnMeta(raw string) (ledgerTurnMeta, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ledgerTurnMeta{}, nil
	}
	var meta ledgerTurnMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return ledgerTurnMeta{}, err
	}
	return meta, nil
}

func ledgerAttachmentRef(meta ledgerAttachmentMeta) SessionLedgerAttachmentRef {
	url := strings.TrimSpace(meta.URL)
	out := SessionLedgerAttachmentRef{
		Kind:      strings.TrimSpace(meta.Kind),
		URL:       url,
		MediaType: firstNonEmpty(meta.MediaType, meta.MediaTypeCamel),
		FileName:  firstNonEmpty(meta.FileName, meta.FileNameCamel),
		SourceID:  firstNonEmpty(meta.SourceID, meta.SourceIDCamel),
		Durable:   isDurableAttachmentRef(url),
	}
	if url != "" && !out.Durable {
		out.URL = ""
		out.Evidence = AttachmentEvidenceRedactedTemp
	}
	return out
}

func isDurableAttachmentRef(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case lower == "":
		return false
	case strings.HasPrefix(lower, "gormes://"):
		return true
	case strings.HasPrefix(lower, "https://"):
		return true
	case strings.HasPrefix(lower, "http://"):
		return true
	case strings.HasPrefix(lower, "ipfs://"):
		return true
	case strings.HasPrefix(lower, "s3://"):
		return true
	case strings.HasPrefix(lower, "gs://"):
		return true
	default:
		return false
	}
}

func ledgerResetBoundaries(sessionID string, items []Metadata) []SessionLedgerResetBoundary {
	var out []SessionLedgerResetBoundary
	for _, item := range items {
		item = finalizeMetadata(item)
		if item.ParentSessionID != sessionID || item.LineageKind != LineageKindCompression {
			continue
		}
		out = append(out, SessionLedgerResetBoundary{
			BeforeSessionID: sessionID,
			AfterSessionID:  item.SessionID,
			Kind:            item.LineageKind,
			Reason:          "fresh_session_reset",
			CreatedAtUnix:   item.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAtUnix != out[j].CreatedAtUnix {
			return out[i].CreatedAtUnix < out[j].CreatedAtUnix
		}
		return out[i].AfterSessionID < out[j].AfterSessionID
	})
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
