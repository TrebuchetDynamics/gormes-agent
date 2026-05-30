package sessionspage

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestSlashNameAndArg(t *testing.T) {
	if got := SlashName(" /resume sess-123 "); got != "resume" {
		t.Fatalf("SlashName = %q, want resume", got)
	}
	if got := SlashArg("/resume   sess-123 with spaces"); got != "sess-123 with spaces" {
		t.Fatalf("SlashArg = %q, want full query", got)
	}
}

func TestLimitClamps(t *testing.T) {
	cases := map[string]int{"/sessions": 20, "/sessions -1": 1, "/sessions 5": 5, "/sessions 500": 100, "/sessions nope": 20}
	for input, want := range cases {
		if got := Limit(input); got != want {
			t.Fatalf("Limit(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestResumeStatusAndHistoryClone(t *testing.T) {
	if got := ResumeSuccessStatus("sess-1", 2); got != "resumed sess-1 (2 messages)" {
		t.Fatalf("ResumeSuccessStatus = %q", got)
	}
	history := []llm.Message{{Role: "user", ContentParts: []llm.MessageContentPart{{Type: "text", Text: "hello"}}}}
	clone := CloneResumeHistory(history)
	clone[0].ContentParts[0].Text = "mutated"
	if history[0].ContentParts[0].Text != "hello" {
		t.Fatalf("CloneResumeHistory shared ContentParts backing slice")
	}
}

func TestBuildFormatsEntries(t *testing.T) {
	page, ok := Build([]Entry{{ID: "sess-1", Title: "Demo", Preview: "hello", Source: "tui", LastActiveAt: 1700000000, MessageCount: 2}})
	if !ok {
		t.Fatal("Build ok = false, want page")
	}
	for _, want := range []string{"Demo", "ID: sess-1", "Preview: hello", "2 messages", "source: tui", "last active: 2023-11-14 22:13 UTC"} {
		if !strings.Contains(page.Body, want) {
			t.Fatalf("page missing %q:\n%s", want, page.Body)
		}
	}
}

func TestBuildRejectsEmptyEntries(t *testing.T) {
	if _, ok := Build(nil); ok {
		t.Fatal("Build ok = true, want false")
	}
}
