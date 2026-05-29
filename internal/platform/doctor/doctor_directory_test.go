package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Parity with hermes_cli/doctor.py@55c9f3206:812 ◆ Directory Structure, but
// over the Gormes-OWNED ~/.gormes layout: a populated home reports PASS items
// for the home dir, the Gormes subdirs, the SOUL.md persona, and the
// memory/MEMORY.md + memory/USER.md starters with char counts.
func TestCheckDirectoryStructurePopulatedHome(t *testing.T) {
	home := t.TempDir()
	for _, d := range []string{"sessions", "memory", "skills", "cron", "subagents", "tools", "hooks"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	if err := os.WriteFile(filepath.Join(home, "SOUL.md"), []byte("You are Gormes, a helpful agent with a distinct voice.\n"), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "memory", "MEMORY.md"), []byte("durable memory line one\n"), 0o644); err != nil {
		t.Fatalf("write MEMORY.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "memory", "USER.md"), []byte("user prefers terse answers\n"), 0o644); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}

	r := CheckDirectoryStructure(home)

	if r.Name != "Directory Structure" {
		t.Fatalf("CheckResult.Name = %q, want %q", r.Name, "Directory Structure")
	}
	if r.Status != StatusPass {
		t.Fatalf("fully-populated home should be PASS, got %v summary=%q", r.Status, r.Summary)
	}
	itemByName := map[string]ItemInfo{}
	for _, it := range r.Items {
		itemByName[it.Name] = it
	}
	for _, d := range []string{"sessions/", "memory/", "skills/", "cron/", "subagents/", "tools/", "hooks/"} {
		it, ok := itemByName[d]
		if !ok {
			t.Fatalf("missing subdir item %q in %+v", d, r.Items)
		}
		if it.Status != StatusPass {
			t.Fatalf("subdir %q present but item not PASS: %+v", d, it)
		}
	}
	soul := itemByName["SOUL.md"]
	if soul.Status != StatusPass || !strings.Contains(strings.ToLower(soul.Note), "persona") {
		t.Fatalf("SOUL.md with real content should PASS 'persona configured', got %+v", soul)
	}
	for _, f := range []string{"memory/MEMORY.md", "memory/USER.md"} {
		it := itemByName[f]
		if it.Status != StatusPass || !strings.Contains(it.Note, "chars") {
			t.Fatalf("%s present should PASS with a char count, got %+v", f, it)
		}
	}
	// Owned divergence: never Hermes paths/wording.
	all := r.Summary
	for _, it := range r.Items {
		all += " " + it.Name + " " + it.Note
	}
	for _, forbidden := range []string{"~/.hermes", "hermes setup", "memories/"} {
		if strings.Contains(all, forbidden) {
			t.Fatalf("Directory Structure leaked Hermes-owned wording %q: %s", forbidden, all)
		}
	}
}

func TestCheckDirectoryStructureDefaultSoulIsTemplateOnly(t *testing.T) {
	home := t.TempDir()
	soulContent, ok := defaultTemplateContent("SOUL.md")
	if !ok {
		t.Fatal("default SOUL.md template not found")
	}
	if err := os.WriteFile(filepath.Join(home, "SOUL.md"), []byte(soulContent), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}

	r := CheckDirectoryStructure(home)

	var soul ItemInfo
	for _, it := range r.Items {
		if it.Name == "SOUL.md" {
			soul = it
			break
		}
	}
	if soul.Name == "" {
		t.Fatalf("missing SOUL.md item in %+v", r.Items)
	}
	if soul.Status != StatusPass {
		t.Fatalf("default SOUL.md should be a non-actionable PASS item, got %+v", soul)
	}
	if !strings.Contains(soul.Note, "edit it to customize personality") || strings.Contains(soul.Note, "persona configured") {
		t.Fatalf("default SOUL.md should be treated as template-only, got %+v", soul)
	}
}

// A fresh/empty home: subdirs and SOUL.md are WARN with create/setup
// guidance, but the final Found-N summary follows Hermes' explicit issue
// funnel. These local-first warnings are visible in the section yet do not
// inflate the action summary.
func TestCheckDirectoryStructureFreshHomeMemoryStartersAreNonActionable(t *testing.T) {
	home := filepath.Join(t.TempDir(), "does-not-exist-yet")

	r := CheckDirectoryStructure(home)

	if r.Status != StatusWarn {
		t.Fatalf("fresh home with missing subdirs should be WARN overall, got %v", r.Status)
	}
	itemByName := map[string]ItemInfo{}
	for _, it := range r.Items {
		itemByName[it.Name] = it
	}
	if it := itemByName["sessions/"]; it.Status != StatusWarn || !strings.Contains(it.Note, "will be created") {
		t.Fatalf("missing subdir should WARN 'will be created on first use', got %+v", it)
	}
	if it := itemByName["SOUL.md"]; it.Status != StatusWarn || !strings.Contains(it.Note, "gormes setup") {
		t.Fatalf("missing SOUL.md should WARN with `gormes setup`, got %+v", it)
	}
	for _, f := range []string{"memory/MEMORY.md", "memory/USER.md"} {
		it := itemByName[f]
		if it.Status != StatusPass {
			t.Fatalf("%s not-yet-created MUST be non-actionable PASS (not WARN/FAIL), got %+v", f, it)
		}
		if !strings.Contains(it.Note, "not yet created") {
			t.Fatalf("%s should note 'not yet created', got %+v", f, it)
		}
	}
	if issues := CollectDoctorIssues([]CheckResult{r}); len(issues) != 0 {
		t.Fatalf("fresh-home directory warnings must not inflate Found-N issues: %+v", issues)
	}
}
