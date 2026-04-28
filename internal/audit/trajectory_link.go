package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

// TrajectoryCompressionLinkInput collects lineage evidence from a transcript
// compression pass before it is written into the audit stream.
type TrajectoryCompressionLinkInput struct {
	ParentSessionID  string
	ChildSessionID   string
	StartIndex       int
	EndIndex         int
	OriginalTurns    int
	CompressedTurns  int
	OriginalTokens   int
	CompressedTokens int
	TokensSaved      int
	Summary          string
	CompressedAt     time.Time
}

// TrajectoryCompressionLink is the redacted evidence envelope for one
// compressed trajectory. It hashes summary text instead of storing it.
type TrajectoryCompressionLink struct {
	ParentSessionID  string    `json:"parent_session_id"`
	ChildSessionID   string    `json:"child_session_id"`
	StartIndex       int       `json:"start_index"`
	EndIndex         int       `json:"end_index"`
	OriginalTurns    int       `json:"original_turns"`
	CompressedTurns  int       `json:"compressed_turns"`
	OriginalTokens   int       `json:"original_tokens"`
	CompressedTokens int       `json:"compressed_tokens"`
	TokensSaved      int       `json:"tokens_saved"`
	SummarySHA256    string    `json:"summary_sha256"`
	CompressedAt     time.Time `json:"compressed_at"`
}

// NewTrajectoryCompressionLink normalizes session ids and computes the summary
// hash used as append-only compressed-evidence lineage.
func NewTrajectoryCompressionLink(in TrajectoryCompressionLinkInput) TrajectoryCompressionLink {
	compressedAt := in.CompressedAt
	if compressedAt.IsZero() {
		compressedAt = time.Now().UTC()
	} else {
		compressedAt = compressedAt.UTC()
	}
	sum := sha256.Sum256([]byte(in.Summary))
	return TrajectoryCompressionLink{
		ParentSessionID:  strings.TrimSpace(in.ParentSessionID),
		ChildSessionID:   strings.TrimSpace(in.ChildSessionID),
		StartIndex:       in.StartIndex,
		EndIndex:         in.EndIndex,
		OriginalTurns:    in.OriginalTurns,
		CompressedTurns:  in.CompressedTurns,
		OriginalTokens:   in.OriginalTokens,
		CompressedTokens: in.CompressedTokens,
		TokensSaved:      in.TokensSaved,
		SummarySHA256:    hex.EncodeToString(sum[:]),
		CompressedAt:     compressedAt,
	}
}

// TrajectoryCompressionAuditRecord converts lineage evidence into the existing
// JSONL audit envelope.
func TrajectoryCompressionAuditRecord(link TrajectoryCompressionLink) Record {
	args, err := json.Marshal(link)
	if err != nil {
		args = json.RawMessage(`null`)
	}
	return Record{
		Timestamp:  link.CompressedAt,
		Source:     "trajectory_compressor",
		SessionID:  link.ChildSessionID,
		Tool:       "trajectory_compress",
		Args:       args,
		Status:     "completed",
		Error:      "",
		DurationMs: 0,
	}
}
