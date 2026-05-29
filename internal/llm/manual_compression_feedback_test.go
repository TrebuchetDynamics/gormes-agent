package llm

import (
	"strings"
	"testing"
)

func TestManualCompressionFeedbackNoop(t *testing.T) {
	history := []Message{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
		{Role: "assistant", Content: "four"},
	}

	got := SummarizeManualCompression(history, append([]Message(nil), history...), 1200, 1200)

	if !got.Noop {
		t.Fatalf("Noop = false, want true")
	}
	if got.Headline != "No changes from compression: 4 messages" {
		t.Fatalf("Headline = %q", got.Headline)
	}
	if got.TokenLine != "Approx request size: ~1,200 tokens (unchanged)" {
		t.Fatalf("TokenLine = %q", got.TokenLine)
	}
	if got.Note != "" {
		t.Fatalf("Note = %q, want empty", got.Note)
	}
	if strings.Contains(got.Headline, "Compressed:") {
		t.Fatalf("noop headline has compressed success marker: %q", got.Headline)
	}
}

func TestManualCompressionFeedbackCompressedTokenDelta(t *testing.T) {
	before := []Message{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
		{Role: "assistant", Content: "four"},
	}
	after := []Message{
		before[0],
		{Role: "assistant", Content: "summary"},
		before[3],
	}

	got := SummarizeManualCompression(before, after, 1234, 678)

	if got.Noop {
		t.Fatalf("Noop = true, want false")
	}
	if got.Headline != "Compressed: 4 -> 3 messages" {
		t.Fatalf("Headline = %q", got.Headline)
	}
	if got.TokenLine != "Approx request size: ~1,234 -> ~678 tokens" {
		t.Fatalf("TokenLine = %q", got.TokenLine)
	}
}

func TestManualCompressionFeedbackTokenEstimateRises(t *testing.T) {
	before := []Message{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
		{Role: "assistant", Content: "four"},
	}
	after := []Message{
		before[0],
		{Role: "assistant", Content: "Dense summary that still estimates higher."},
		before[3],
	}

	got := SummarizeManualCompression(before, after, 100, 120)

	if got.Note == "" || !strings.Contains(got.Note, "denser summaries") {
		t.Fatalf("Note = %q, want denser-summary explanation", got.Note)
	}
}

func TestParseManualCompressionFocus(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "topic", raw: "/compress database schema", want: "database schema"},
		{name: "bare", raw: "/compress", want: ""},
		{name: "space", raw: "/compress   ", want: ""},
		{name: "without slash", raw: "compress API endpoints", want: "API endpoints"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseManualCompressionFocus(tt.raw); got != tt.want {
				t.Fatalf("ParseManualCompressionFocus(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestManualCompressionSessionSplitEvidence(t *testing.T) {
	got := ManualCompressionSessionSplit("sess-parent", "sess-child")

	if got.Code != "manual_compression_session_split" {
		t.Fatalf("Code = %q", got.Code)
	}
	if got.OldSessionID != "sess-parent" || got.NewSessionID != "sess-child" {
		t.Fatalf("session ids = %q/%q", got.OldSessionID, got.NewSessionID)
	}
	if !got.PendingTitleReset {
		t.Fatalf("PendingTitleReset = false, want true")
	}
	if got.Message == "" || strings.Contains(got.Message, "one") || strings.Contains(got.Message, "summary") {
		t.Fatalf("Message = %q, want redacted evidence without prompt content", got.Message)
	}
}
