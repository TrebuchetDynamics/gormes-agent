package llm

import (
	"strings"
	"testing"
	"time"
)

func TestBuildTurnMetadataBlock_RendersHermesFormat(t *testing.T) {
	clock := time.Date(2026, time.April, 29, 14, 46, 0, 0, time.Local)
	got := BuildTurnMetadataBlock(TurnMetadataOptions{
		Now:       clock,
		SessionID: "sess-1",
		Model:     "claude-opus-4-7",
		Provider:  "anthropic",
	})
	want := "Conversation started: Wednesday, April 29, 2026 02:46 PM\nSession ID: sess-1\nModel: claude-opus-4-7\nProvider: anthropic"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("BuildTurnMetadataBlock() does not start with the Hermes-format block.\nwant prefix:\n%s\ngot:\n%s", want, got)
	}
}

func TestBuildTurnMetadataBlock_OmitsEmptyFields(t *testing.T) {
	clock := time.Date(2026, time.April, 29, 14, 46, 0, 0, time.Local)
	t.Run("only_timestamp", func(t *testing.T) {
		got := BuildTurnMetadataBlock(TurnMetadataOptions{Now: clock})
		want := "Conversation started: Wednesday, April 29, 2026 02:46 PM"
		if got != want {
			t.Fatalf("only-timestamp got %q, want %q", got, want)
		}
		if strings.Contains(got, "Session ID") || strings.Contains(got, "Model") || strings.Contains(got, "Provider") {
			t.Fatalf("only-timestamp must not include Session ID / Model / Provider lines: %q", got)
		}
	})
	t.Run("empty_only_model", func(t *testing.T) {
		got := BuildTurnMetadataBlock(TurnMetadataOptions{
			Now:       clock,
			SessionID: "sess-1",
			Provider:  "anthropic",
		})
		if !strings.Contains(got, "Session ID: sess-1") {
			t.Fatalf("empty-only-model must keep Session ID line: %q", got)
		}
		if !strings.Contains(got, "Provider: anthropic") {
			t.Fatalf("empty-only-model must keep Provider line: %q", got)
		}
		if strings.Contains(got, "Model:") {
			t.Fatalf("empty-only-model must omit Model line: %q", got)
		}
	})
}

func TestBuildTurnMetadataBlock_EmptyClockReturnsEmpty(t *testing.T) {
	got := BuildTurnMetadataBlock(TurnMetadataOptions{Now: time.Time{}})
	if got != "" {
		t.Fatalf("zero clock + empty fields must return empty string, got %q", got)
	}
}

func TestBuildTurnMetadataBlock_TimezoneHonorsLocal(t *testing.T) {
	// One absolute instant rendered in two distinct zones must produce two
	// distinct formatted strings, proving the helper honors the supplied
	// location rather than always reformatting to UTC or another fixed zone.
	hawaii, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("LoadLocation Pacific/Honolulu: %v", err)
	}
	tUTC := time.Date(2026, time.April, 29, 14, 46, 0, 0, time.UTC)
	tHawaii := tUTC.In(hawaii)

	gotUTC := BuildTurnMetadataBlock(TurnMetadataOptions{Now: tUTC})
	gotHI := BuildTurnMetadataBlock(TurnMetadataOptions{Now: tHawaii})

	if gotUTC == "" || gotHI == "" {
		t.Fatalf("expected non-empty rendered blocks; gotUTC=%q gotHI=%q", gotUTC, gotHI)
	}
	if gotUTC == gotHI {
		t.Fatalf("expected zone-distinct rendering (UTC and Pacific/Honolulu views of the same instant), got identical:\n%s", gotUTC)
	}
}
