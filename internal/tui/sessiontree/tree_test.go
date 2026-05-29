package sessiontree

import (
	"strings"
	"testing"
)

func TestBuildPageFiltersToolMessages(t *testing.T) {
	page, ok := BuildPage(SessionTreeResult{Filter: "user-only", Entries: []SessionTreeEntry{
		{ID: "root", Title: "Root", LineageKind: "primary", Labels: []string{"pinned"}, UpdatedAt: 100, Messages: []SessionTreeMessage{{ID: 1, Role: "user", Content: "hello"}, {ID: 2, Role: "tool", Content: "noise"}}},
		{ID: "fork", LineageKind: "fork", Depth: 1, Active: true, Messages: []SessionTreeMessage{{ID: 3, Role: "user", Content: "fork prompt"}}},
	}})
	if !ok {
		t.Fatal("BuildPage returned ok=false")
	}
	joined := page.Title + "\n" + page.Body
	for _, want := range []string{"Session Tree — user-only", "Root", "labels: pinned", "*   ↳ fork", "fork prompt"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("page missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "noise") {
		t.Fatalf("page included tool noise:\n%s", joined)
	}
}
