package prompttemplates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptTemplateDiscoveryAndExpansion(t *testing.T) {
	root := t.TempDir()
	mustWriteTemplate(t, filepath.Join(root, "review.md"), `---
description: Review staged git changes
argument-hint: "<scope>"
---
Review $1 with all args: $@
Tail: ${@:2}
One: ${@:2:1}
Again: $ARGUMENTS
Missing: $9
`)
	mustWriteTemplate(t, filepath.Join(root, "skills.md"), `---
description: Should not shadow built-in skills command
---
shadow
`)
	mustWriteTemplate(t, filepath.Join(root, "not-markdown.txt"), "ignored")
	mustWriteTemplate(t, filepath.Join(root, "nested", "deep.md"), "ignored")

	catalog, err := Discover(DiscoverOptions{
		Roots:         []string{root, filepath.Join(root, "missing")},
		ReservedNames: []string{"skills"},
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if len(catalog.Templates) != 1 {
		t.Fatalf("templates len = %d, want 1: %+v", len(catalog.Templates), catalog.Templates)
	}
	review, ok := catalog.Lookup("review")
	if !ok {
		t.Fatalf("review template missing from catalog: %+v", catalog.Templates)
	}
	if review.Description != "Review staged git changes" || review.ArgumentHint != "<scope>" {
		t.Fatalf("frontmatter = description %q hint %q", review.Description, review.ArgumentHint)
	}
	if _, ok := catalog.Lookup("deep"); ok {
		t.Fatal("nested prompt templates must not be discovered recursively")
	}

	reasons := fmt.Sprint(catalog.Skipped)
	for _, want := range []string{"reserved_name", "root_missing"} {
		if !strings.Contains(reasons, want) {
			t.Fatalf("skipped evidence %q missing %q", reasons, want)
		}
	}
	if strings.Contains(reasons, root) {
		t.Fatalf("skipped evidence leaked absolute root %q: %s", root, reasons)
	}

	expanded := Expand(review, []string{"staged", "bug fix"})
	for _, want := range []string{
		"Review staged with all args: staged bug fix",
		"Tail: bug fix",
		"One: bug fix",
		"Again: staged bug fix",
		"Missing:",
	} {
		if !strings.Contains(expanded, want) {
			t.Fatalf("expanded template missing %q:\n%s", want, expanded)
		}
	}
}

func mustWriteTemplate(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
