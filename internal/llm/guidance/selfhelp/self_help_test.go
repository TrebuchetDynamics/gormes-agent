package selfhelp

import (
	"strings"
	"testing"
)

func TestGuidanceMentionsGormesDocs(t *testing.T) {
	guidance, ok := GuidanceForPrompt("How do I configure Gormes Agent?")
	if !ok {
		t.Fatal("GuidanceForPrompt() ok = false, want true")
	}

	for _, want := range []string{
		"Gormes",
		"https://docs.gormes.ai/",
		"self-help-unavailable",
	} {
		if !strings.Contains(guidance, want) {
			t.Fatalf("guidance missing %q:\n%s", want, guidance)
		}
	}
	if strings.Contains(guidance, "Gormes Agent") {
		t.Fatalf("guidance must not emit deprecated Gormes Agent wording:\n%s", guidance)
	}
}

func TestGuidanceDoesNotMentionHermesAgent(t *testing.T) {
	guidance, ok := GuidanceForPrompt("Help me troubleshoot Gormes startup.")
	if !ok {
		t.Fatal("GuidanceForPrompt() ok = false, want true")
	}

	if strings.Contains(guidance, "Hermes Agent") {
		t.Fatalf("guidance contains Hermes Agent product wording:\n%s", guidance)
	}
}

func TestGuidanceGateMatchesSetupQuestions(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		wantOK bool
	}{
		{name: "configure", prompt: "How do I configure Gormes for Telegram?", wantOK: true},
		{name: "legacy agent alias setup", prompt: "Can you help me set up Gormes Agent?", wantOK: true},
		{name: "troubleshoot", prompt: "Troubleshoot Gormes doctor failing before launch.", wantOK: true},
		{name: "use", prompt: "How do I use Gormes in TUI mode?", wantOK: true},
		{name: "unrelated", prompt: "Write a Go unit test for JSON parsing.", wantOK: false},
		{name: "unrelated gormes development", prompt: "Use Go to add a Gormes unit test.", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guidance, ok := GuidanceForPrompt(tt.prompt)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v; guidance = %q", ok, tt.wantOK, guidance)
			}
			if !tt.wantOK && guidance != "" {
				t.Fatalf("guidance = %q, want empty for unrelated prompt", guidance)
			}
		})
	}
}

func TestGuidanceIsDeterministic(t *testing.T) {
	first, firstOK := GuidanceForPrompt("How do I configure Gormes?")
	second, secondOK := GuidanceForPrompt("How do I configure Gormes?")

	if firstOK != secondOK || first != second {
		t.Fatalf("guidance changed between calls:\nfirst ok=%v\n%s\nsecond ok=%v\n%s", firstOK, first, secondOK, second)
	}
}
