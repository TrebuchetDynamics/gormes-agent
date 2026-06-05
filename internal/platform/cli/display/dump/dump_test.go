package dump

import (
	"strings"
	"testing"
)

func TestRenderSummary_StableOrder(t *testing.T) {
	in := Input{
		Version:     "0.1.0",
		OS:          "linux",
		Arch:        "amd64",
		ProfileName: "main",
		Toolsets:    []string{"core", "web"},
	}
	got := RenderSummary(in)
	want := "version: 0.1.0\nos: linux\narch: amd64\nprofile: main\ntoolsets: core, web\n"
	if got != want {
		t.Fatalf("RenderSummary stable order:\n got=%q\nwant=%q", got, want)
	}
}

func TestRenderSummary_RedactsSecrets(t *testing.T) {
	in := Input{
		Version:         "v1.0-sk-abcdef",
		OS:              "linux",
		Arch:            "amd64",
		ProfileName:     "main",
		Toolsets:        []string{"core"},
		SecretsLikeKeys: []string{"sk-abcdef"},
	}
	got := RenderSummary(in)
	if strings.Contains(got, "sk-abcdef") {
		t.Fatalf("expected secret 'sk-abcdef' to be redacted, got: %q", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("expected '[redacted]' marker in output, got: %q", got)
	}
	if !strings.Contains(got, "version: v1.0-[redacted]") {
		t.Fatalf("expected version line to keep prefix and redact secret, got: %q", got)
	}
}

func TestRenderSummary_HandlesMissingFields(t *testing.T) {
	got := RenderSummary(Input{})
	want := "version: unknown\nos: unknown\narch: unknown\nprofile: unknown\ntoolsets: (none)\n"
	if got != want {
		t.Fatalf("RenderSummary missing fields:\n got=%q\nwant=%q", got, want)
	}
}

func TestRenderSummary_DeterministicAcrossCalls(t *testing.T) {
	in := Input{
		Version:         "0.2.0",
		OS:              "darwin",
		Arch:            "arm64",
		ProfileName:     "alpha",
		Toolsets:        []string{"core", "web"},
		SecretsLikeKeys: []string{"sk-zzzz"},
	}
	first := RenderSummary(in)
	second := RenderSummary(in)
	if first != second {
		t.Fatalf("RenderSummary not deterministic across calls:\n first=%q\nsecond=%q", first, second)
	}
}

func TestRenderSummary_NoTrailingWhitespace(t *testing.T) {
	in := Input{
		Version:     "0.1.0",
		OS:          "linux",
		Arch:        "amd64",
		ProfileName: "main",
		Toolsets:    []string{"core", "web"},
	}
	got := RenderSummary(in)
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("expected output to end in single '\\n', got: %q", got)
	}
	if strings.HasSuffix(got, "\n\n") {
		t.Fatalf("expected single trailing newline, got: %q", got)
	}
	body := strings.TrimSuffix(got, "\n")
	for i, line := range strings.Split(body, "\n") {
		if trimmed := strings.TrimRight(line, " \t"); trimmed != line {
			t.Fatalf("line %d has trailing whitespace: %q", i, line)
		}
	}
}
