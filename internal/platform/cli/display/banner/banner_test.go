package banner

import (
	"strings"
	"testing"
)

func TestContextLengthFormattingMatchesUpstreamGolden(t *testing.T) {
	tests := []struct {
		tokens int
		want   string
	}{
		{tokens: 999, want: "999"},
		{tokens: 1000, want: "1K"},
		{tokens: 1049, want: "1K"},
		{tokens: 1050, want: "1.1K"},
		{tokens: 128000, want: "128K"},
		{tokens: 1048576, want: "1M"},
		{tokens: 1099999, want: "1.1M"},
		{tokens: 1999000, want: "2M"},
	}

	for _, tt := range tests {
		if got := FormatContextLength(tt.tokens); got != tt.want {
			t.Fatalf("FormatContextLength(%d) = %q, want %q", tt.tokens, got, tt.want)
		}
	}
}

func TestToolsetNameDisplayMatchesUpstreamGolden(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "", want: "unknown"},
		{name: "homeassistant_tools", want: "homeassistant"},
		{name: "honcho_tools", want: "honcho"},
		{name: "web_tools", want: "web"},
		{name: "browser", want: "browser"},
		{name: "file", want: "file"},
		{name: "terminal", want: "terminal"},
	}

	for _, tt := range tests {
		if got := DisplayToolsetName(tt.name); got != tt.want {
			t.Fatalf("DisplayToolsetName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestVersionLabelsMatchUpstreamGolden(t *testing.T) {
	base := Version{
		AgentName:   "Hermes Agent",
		Version:     "0.11.0",
		ReleaseDate: "2026.4.23",
	}
	tests := []struct {
		name  string
		state *GitState
		want  string
	}{
		{
			name: "without git state",
			want: "Hermes Agent v0.11.0 (2026.4.23)",
		},
		{
			name:  "on upstream main",
			state: &GitState{Upstream: "b2f477a3", Local: "b2f477a3"},
			want:  "Hermes Agent v0.11.0 (2026.4.23) · upstream b2f477a3",
		},
		{
			name:  "one carried commit",
			state: &GitState{Upstream: "b2f477a3", Local: "af8aad31", Ahead: 1},
			want:  "Hermes Agent v0.11.0 (2026.4.23) · upstream b2f477a3 · local af8aad31 (+1 carried commit)",
		},
		{
			name:  "multiple carried commits",
			state: &GitState{Upstream: "b2f477a3", Local: "af8aad31", Ahead: 3},
			want:  "Hermes Agent v0.11.0 (2026.4.23) · upstream b2f477a3 · local af8aad31 (+3 carried commits)",
		},
	}

	for _, tt := range tests {
		version := base
		version.GitState = tt.state
		if got := FormatVersionLabel(version); got != tt.want {
			t.Fatalf("%s: FormatVersionLabel() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestVersionDefaultAgentNameIsGormes(t *testing.T) {
	got := FormatVersionLabel(Version{
		Version:     "0.11.0",
		ReleaseDate: "2026.4.23",
	})
	if got != "Gormes Agent v0.11.0 (2026.4.23)" {
		t.Fatalf("FormatVersionLabel default = %q, want Gormes Agent", got)
	}
	if strings.Contains(got, "Hermes Agent") {
		t.Fatalf("FormatVersionLabel default leaked upstream product label: %q", got)
	}
}
