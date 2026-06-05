package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTrajectoryCompressionAuditRecordCarriesLineageWithoutSummaryText(t *testing.T) {
	compressedAt := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	link := NewTrajectoryCompressionLink(TrajectoryCompressionLinkInput{
		ParentSessionID:  " parent-session ",
		ChildSessionID:   " child-session ",
		StartIndex:       4,
		EndIndex:         8,
		OriginalTurns:    12,
		CompressedTurns:  9,
		OriginalTokens:   1200,
		CompressedTokens: 700,
		TokensSaved:      500,
		Summary:          "[CONTEXT SUMMARY]: secret details should not leak",
		CompressedAt:     compressedAt,
	})

	if link.ParentSessionID != "parent-session" || link.ChildSessionID != "child-session" {
		t.Fatalf("session ids not normalized: %+v", link)
	}
	if link.SummarySHA256 == "" {
		t.Fatalf("SummarySHA256 empty: %+v", link)
	}

	rec := TrajectoryCompressionAuditRecord(link)
	if rec.Source != "trajectory_compressor" || rec.Tool != "trajectory_compress" || rec.Status != "completed" {
		t.Fatalf("audit record identity = %+v", rec)
	}
	if rec.SessionID != "child-session" {
		t.Fatalf("record SessionID = %q, want child-session", rec.SessionID)
	}
	if strings.Contains(string(rec.Args), "secret details") {
		t.Fatalf("record args leaked summary text: %s", rec.Args)
	}

	var args TrajectoryCompressionLink
	if err := json.Unmarshal(rec.Args, &args); err != nil {
		t.Fatalf("decode args: %v\n%s", err, rec.Args)
	}
	if args.ParentSessionID != link.ParentSessionID || args.StartIndex != 4 || args.EndIndex != 8 || args.TokensSaved != 500 {
		t.Fatalf("record args = %+v, want lineage %+v", args, link)
	}
	if !args.CompressedAt.Equal(compressedAt) {
		t.Fatalf("CompressedAt = %v, want %v", args.CompressedAt, compressedAt)
	}
}
