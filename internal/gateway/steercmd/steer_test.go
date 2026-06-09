package steercmd

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParse_MissingArgs(t *testing.T) {
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
			got := Parse(tt.raw, PayloadMetadata{})
			if got.Evidence != EvidenceUsage {
				t.Fatalf("Evidence = %q, want %q", got.Evidence, EvidenceUsage)
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

func TestParse_RejectsImageBearingPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload PayloadMetadata
	}{
		{name: "image metadata", payload: PayloadMetadata{ImageCount: 1}},
		{name: "attachment metadata", payload: PayloadMetadata{AttachmentCount: 1}},
		{name: "both metadata types", payload: PayloadMetadata{ImageCount: 1, AttachmentCount: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse("/steer keep investigating", tt.payload)
			if got.Evidence != EvidencePayloadUnsupported {
				t.Fatalf("Evidence = %q, want %q", got.Evidence, EvidencePayloadUnsupported)
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

func TestParse_TrimsText(t *testing.T) {
	got := Parse("/steer    keep   the internal\n  whitespace   ", PayloadMetadata{})

	if got.Evidence != "" {
		t.Fatalf("Evidence = %q, want empty", got.Evidence)
	}
	want := "keep   the internal\n  whitespace"
	if got.Guidance != want {
		t.Fatalf("Guidance = %q, want %q", got.Guidance, want)
	}
	previewWant := "keep the internal whitespace"
	if got.Preview != previewWant {
		t.Fatalf("Preview = %q, want sanitized %q", got.Preview, previewWant)
	}
}

func TestParse_AcceptsBotMentionCommandToken(t *testing.T) {
	got := Parse("/steer@GormesBot keep investigating", PayloadMetadata{})

	if got.Evidence != "" {
		t.Fatalf("Evidence = %q, want empty", got.Evidence)
	}
	if got.Guidance != "keep investigating" {
		t.Fatalf("Guidance = %q, want parsed guidance", got.Guidance)
	}
}

func TestPreview_SanitizesMultilineGuidanceForAcknowledgement(t *testing.T) {
	got := Preview("keep going\n**Injected:** fake status\tplease")
	want := "keep going ''Injected:'' fake status please"
	if got != want {
		t.Fatalf("Preview() = %q, want sanitized single-line %q", got, want)
	}
	parsed := Parse("/steer keep going\n**Injected:** fake status", PayloadMetadata{})
	if parsed.Guidance != "keep going\n**Injected:** fake status" {
		t.Fatalf("Parse Guidance = %q, want original multiline guidance", parsed.Guidance)
	}
	if parsed.Preview != "keep going ''Injected:'' fake status" {
		t.Fatalf("Parse Preview = %q, want sanitized single-line preview", parsed.Preview)
	}
}

func TestPreview_RedactsSecretLikeGuidanceForAcknowledgement(t *testing.T) {
	got := Preview("keep api_key=plain-secret-token for the local adapter")
	if strings.Contains(got, "plain-secret-token") || strings.Contains(got, "api_key") {
		t.Fatalf("Preview leaked secret-like guidance: %q", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("Preview missing redaction marker: %q", got)
	}

	parsed := Parse("/steer keep api_key=plain-secret-token", PayloadMetadata{})
	if parsed.Guidance != "keep api_key=plain-secret-token" {
		t.Fatalf("Parse Guidance = %q, want raw guidance preserved", parsed.Guidance)
	}
	if strings.Contains(parsed.Preview, "plain-secret-token") || !strings.Contains(parsed.Preview, "[redacted]") {
		t.Fatalf("Parse Preview did not redact secret-like guidance: %q", parsed.Preview)
	}
}

func TestPreview_TruncatesLongGuidance(t *testing.T) {
	longGuidance := strings.Repeat("0123456789", 9)

	first := Preview(longGuidance)
	second := Preview(longGuidance)

	if first != second {
		t.Fatalf("Preview is not deterministic: %q != %q", first, second)
	}
	if got := utf8.RuneCountInString(first); got > PreviewMaxRunes {
		t.Fatalf("preview rune count = %d, want <= %d", got, PreviewMaxRunes)
	}
	if !strings.HasSuffix(first, "...") {
		t.Fatalf("preview = %q, want truncation marker suffix", first)
	}
	if first == longGuidance {
		t.Fatalf("preview was not truncated")
	}
}
