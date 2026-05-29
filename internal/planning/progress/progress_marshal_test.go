package progress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveProgressDoesNotHTMLEscapeNestedProgressText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.json")
	prog := &Progress{
		Phases: map[string]Phase{
			"3": {
				Name:        "Memory",
				Deliverable: "SQLite -> graph & recall",
				Subphases: map[string]Subphase{
					"3.F": {
						Name: "Goncho",
						Items: []Item{{
							Name:     "gormes session export <id> --format=markdown",
							Status:   StatusPlanned,
							Contract: "Helper ports keep `A -> B` and `<prompt>` text readable & diff-stable.",
						}},
					},
				},
			},
		},
	}

	if err := SaveProgress(path, prog); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(body)
	for _, escaped := range []string{`\u003c`, `\u003e`, `\u0026`} {
		if strings.Contains(text, escaped) {
			t.Fatalf("SaveProgress output contains HTML escape %s:\n%s", escaped, text)
		}
	}
	for _, want := range []string{"SQLite -> graph & recall", "gormes session export <id> --format=markdown", "`A -> B` and `<prompt>`"} {
		if !strings.Contains(text, want) {
			t.Fatalf("SaveProgress output missing literal %q:\n%s", want, text)
		}
	}
}
