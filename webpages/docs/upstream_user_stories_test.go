package docs_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type upstreamUserStory struct {
	ID       string `json:"id"`
	Author   string `json:"author"`
	Category string `json:"category"`
	Headline string `json:"headline"`
	Quote    string `json:"quote"`
}

func TestUpstreamHermesUserStoriesStaticMirror(t *testing.T) {
	stories := readUpstreamHermesUserStories(t)
	if len(stories) == 0 {
		t.Fatal("upstream user stories source is empty")
	}
	page := readDoc(t, "content/upstream-hermes/user-stories.md")

	if strings.Contains(page, "Upstream interactive component omitted from the static mirror") {
		t.Fatalf("user-stories mirror still hides the upstream story dataset behind an omitted-component placeholder")
	}
	assertContainsAll(t, "content/upstream-hermes/user-stories.md", page, []string{
		"Source data: `hermes-agent/website/src/data/userStories.json`",
		fmt.Sprintf("%d verified upstream user stories", len(stories)),
		"## Category Rollup",
		"## Fresh Representative Entries",
	})

	for _, row := range topUserStoryCategoryRows(stories, 5) {
		if !strings.Contains(page, row) {
			t.Fatalf("user-stories mirror missing category row %q", row)
		}
	}

	byID := make(map[string]upstreamUserStory, len(stories))
	for _, story := range stories {
		byID[story.ID] = story
	}
	for _, id := range []string{
		"reddit-ninjapapi-5-things-hermes",
		"reddit-itsdodobitch-kanban-feature",
		"x-vmiss33-human-guide",
	} {
		story, ok := byID[id]
		if !ok {
			t.Fatalf("upstream userStories.json no longer contains expected fresh entry %q", id)
		}
		for _, want := range []string{story.Headline, story.Author} {
			if !strings.Contains(page, want) {
				t.Fatalf("user-stories mirror missing fresh entry field %q for %s", want, id)
			}
		}
	}

	if strings.Contains(page, stories[0].Quote) {
		t.Fatalf("user-stories mirror should summarize the dataset, not copy every upstream quote body")
	}
}

func readUpstreamHermesUserStories(t *testing.T) []upstreamUserStory {
	t.Helper()
	root := findRepoRoot(t)
	candidates := []string{
		filepath.Join(root, "hermes-agent", "website", "src", "data", "userStories.json"),
		filepath.Join(root, "..", "hermes-agent", "website", "src", "data", "userStories.json"),
		filepath.Join(root, "references", "hermes-agent", "website", "src", "data", "userStories.json"),
	}
	var checked []string
	for _, path := range candidates {
		checked = append(checked, path)
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("read upstream user stories %s: %v", path, err)
		}
		var stories []upstreamUserStory
		if err := json.Unmarshal(raw, &stories); err != nil {
			t.Fatalf("decode upstream user stories %s: %v", path, err)
		}
		return stories
	}
	t.Skipf("upstream Hermes userStories.json not present; checked %s", strings.Join(checked, ", "))
	return nil
}

func topUserStoryCategoryRows(stories []upstreamUserStory, limit int) []string {
	counts := map[string]int{}
	for _, story := range stories {
		category := strings.TrimSpace(story.Category)
		if category == "" {
			category = "uncategorized"
		}
		counts[category]++
	}
	type categoryCount struct {
		category string
		count    int
	}
	rows := make([]categoryCount, 0, len(counts))
	for category, count := range counts {
		rows = append(rows, categoryCount{category: category, count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].count == rows[j].count {
			return rows[i].category < rows[j].category
		}
		return rows[i].count > rows[j].count
	})
	if limit > len(rows) {
		limit = len(rows)
	}
	out := make([]string, 0, limit)
	for _, row := range rows[:limit] {
		out = append(out, fmt.Sprintf("| `%s` | %d |", row.category, row.count))
	}
	return out
}
