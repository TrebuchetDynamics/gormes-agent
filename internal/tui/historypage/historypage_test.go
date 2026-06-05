package historypage

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestPreviewLimit(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  int
	}{
		{input: "/history", want: 400},
		{input: "/history 120", want: 120},
		{input: "/history 10", want: 80},
		{input: "/history nope", want: 400},
	} {
		if got := PreviewLimit(tt.input); got != tt.want {
			t.Fatalf("PreviewLimit(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestBuildHistoryPage(t *testing.T) {
	page, ok := Build([]llm.Message{
		{Role: "system", Content: "hidden"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", ContentParts: []llm.MessageContentPart{{Type: "text", Text: "hi"}}},
	}, 80)
	if !ok {
		t.Fatal("Build returned ok=false")
	}
	for _, want := range []string{"[You #1]", "hello", "[Gormes #2]", "hi"} {
		if !strings.Contains(page.Body, want) {
			t.Fatalf("body missing %q:\n%s", want, page.Body)
		}
	}
	if strings.Contains(page.Body, "hidden") {
		t.Fatalf("system message leaked into history page:\n%s", page.Body)
	}
}

func TestBuildHistoryPageRejectsEmptyConversation(t *testing.T) {
	if _, ok := Build([]llm.Message{{Role: "system", Content: "hidden"}}, 80); ok {
		t.Fatal("Build system-only history ok=true, want false")
	}
}

func TestHandleSlashOpensHistoryOrReportsEmptyConversation(t *testing.T) {
	closed := HandleSlash(nil, "/history")
	if closed.Open || closed.StatusMessage != "no conversation yet" {
		t.Fatalf("closed /history result = %+v, want no conversation yet", closed)
	}

	opened := HandleSlash([]llm.Message{{Role: "user", Content: "hello"}}, "/history 80")
	if !opened.Open || opened.StatusMessage != "history opened" || opened.Page.Title != "History" {
		t.Fatalf("opened /history result = %+v", opened)
	}
	if !strings.Contains(opened.Page.Body, "hello") {
		t.Fatalf("opened history missing user message:\n%s", opened.Page.Body)
	}
}
