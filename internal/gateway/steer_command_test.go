package gateway

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseSteerCommand_MissingArgs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty command", raw: "/steer"},
		{name: "space only args", raw: "/steer     "},
		{name: "newline only args", raw: "/steer\n\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSteerCommand(tt.raw, SteerPayloadMetadata{})
			if got.Evidence != SteerEvidenceUsage {
				t.Fatalf("Evidence = %q, want %q", got.Evidence, SteerEvidenceUsage)
			}
			if got.Guidance != "" {
				t.Fatalf("Guidance = %q, want empty", got.Guidance)
			}
			if got.Preview != "" {
				t.Fatalf("Preview = %q, want empty", got.Preview)
			}
		})
	}
}

func TestParseSteerCommand_RejectsImageBearingPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload SteerPayloadMetadata
	}{
		{name: "image metadata", payload: SteerPayloadMetadata{ImageCount: 1}},
		{name: "attachment metadata", payload: SteerPayloadMetadata{AttachmentCount: 1}},
		{name: "both metadata types", payload: SteerPayloadMetadata{ImageCount: 1, AttachmentCount: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSteerCommand("/steer keep investigating", tt.payload)
			if got.Evidence != SteerEvidencePayloadUnsupported {
				t.Fatalf("Evidence = %q, want %q", got.Evidence, SteerEvidencePayloadUnsupported)
			}
			if got.Guidance != "" {
				t.Fatalf("Guidance = %q, want empty", got.Guidance)
			}
			if got.Preview != "" {
				t.Fatalf("Preview = %q, want empty", got.Preview)
			}
		})
	}
}

func TestParseSteerCommand_TrimsText(t *testing.T) {
	got := ParseSteerCommand("/steer    keep   the internal\n  whitespace   ", SteerPayloadMetadata{})

	if got.Evidence != "" {
		t.Fatalf("Evidence = %q, want empty", got.Evidence)
	}
	want := "keep   the internal\n  whitespace"
	if got.Guidance != want {
		t.Fatalf("Guidance = %q, want %q", got.Guidance, want)
	}
	if got.Preview != want {
		t.Fatalf("Preview = %q, want %q", got.Preview, want)
	}
}

func TestSteerPreview_TruncatesLongGuidance(t *testing.T) {
	longGuidance := strings.Repeat("0123456789", 9)

	first := SteerPreview(longGuidance)
	second := SteerPreview(longGuidance)

	if first != second {
		t.Fatalf("SteerPreview is not deterministic: %q != %q", first, second)
	}
	if got := utf8.RuneCountInString(first); got > SteerPreviewMaxRunes {
		t.Fatalf("preview rune count = %d, want <= %d", got, SteerPreviewMaxRunes)
	}
	if !strings.HasSuffix(first, "...") {
		t.Fatalf("preview = %q, want truncation marker suffix", first)
	}
	if first == longGuidance {
		t.Fatalf("preview was not truncated")
	}
}
