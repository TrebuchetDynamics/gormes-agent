package agenttemplate

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAgentTemplateDefaultFilesMatchLiveTurnLookup(t *testing.T) {
	files := DefaultFiles()
	got := map[string]FileTemplate{}
	for _, file := range files {
		if filepath.IsAbs(file.Path) {
			t.Fatalf("template path %q must be relative", file.Path)
		}
		if strings.Contains(filepath.ToSlash(file.Path), "../") {
			t.Fatalf("template path %q must not traverse", file.Path)
		}
		got[file.Path] = file
	}

	for _, want := range []string{
		"SOUL.md",
		"AGENTS.md",
		"IDENTITY.md",
		"TOOLS.md",
		filepath.Join("memory", "USER.md"),
		filepath.Join("memory", "MEMORY.md"),
	} {
		if _, ok := got[want]; !ok {
			t.Fatalf("missing default template %q; got paths %v", want, sortedTemplatePaths(files))
		}
	}
	if soul := got["SOUL.md"].Content; !strings.Contains(soul, "Gormes Agent") || !strings.Contains(soul, "helpful, knowledgeable, and direct") {
		t.Fatalf("SOUL.md template does not carry the Hermes-derived persona defaults:\n%s", soul)
	}
}

func TestAgentTemplateApplyCreatesMissingFiles(t *testing.T) {
	target := t.TempDir()

	result, err := ApplyDefaultTemplates(WriteOptions{TargetDir: target})
	if err != nil {
		t.Fatalf("ApplyDefaultTemplates: %v", err)
	}
	if got := actionsByPath(result); got["SOUL.md"] != ActionCreate || got[filepath.Join("memory", "USER.md")] != ActionCreate {
		t.Fatalf("actions = %v, want create for SOUL.md and memory/USER.md", got)
	}
	for _, file := range DefaultFiles() {
		path := filepath.Join(target, file.Path)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", file.Path, err)
		}
		if string(body) != file.Content {
			t.Fatalf("%s content mismatch", file.Path)
		}
	}
}

func TestAgentTemplateApplySkipsExistingWithoutForce(t *testing.T) {
	target := t.TempDir()
	path := filepath.Join(target, "SOUL.md")
	if err := os.WriteFile(path, []byte("custom persona\n"), 0o644); err != nil {
		t.Fatalf("seed SOUL.md: %v", err)
	}

	result, err := ApplyDefaultTemplates(WriteOptions{TargetDir: target})
	if err != nil {
		t.Fatalf("ApplyDefaultTemplates: %v", err)
	}
	if got := actionsByPath(result)["SOUL.md"]; got != ActionSkip {
		t.Fatalf("SOUL.md action = %q, want %q", got, ActionSkip)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SOUL.md: %v", err)
	}
	if string(body) != "custom persona\n" {
		t.Fatalf("SOUL.md was overwritten without force:\n%s", body)
	}
}

func TestAgentTemplateApplyForceOverwritesExisting(t *testing.T) {
	target := t.TempDir()
	path := filepath.Join(target, "SOUL.md")
	if err := os.WriteFile(path, []byte("custom persona\n"), 0o644); err != nil {
		t.Fatalf("seed SOUL.md: %v", err)
	}

	result, err := ApplyDefaultTemplates(WriteOptions{TargetDir: target, Force: true})
	if err != nil {
		t.Fatalf("ApplyDefaultTemplates: %v", err)
	}
	if got := actionsByPath(result)["SOUL.md"]; got != ActionOverwrite {
		t.Fatalf("SOUL.md action = %q, want %q", got, ActionOverwrite)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SOUL.md: %v", err)
	}
	if strings.Contains(string(body), "custom persona") || !strings.Contains(string(body), "Gormes Agent") {
		t.Fatalf("SOUL.md was not overwritten with the default template:\n%s", body)
	}
}

func TestAgentTemplateApplyDryRunDoesNotWrite(t *testing.T) {
	target := t.TempDir()
	path := filepath.Join(target, "SOUL.md")
	if err := os.WriteFile(path, []byte("custom persona\n"), 0o644); err != nil {
		t.Fatalf("seed SOUL.md: %v", err)
	}

	result, err := ApplyDefaultTemplates(WriteOptions{TargetDir: target, DryRun: true})
	if err != nil {
		t.Fatalf("ApplyDefaultTemplates dry-run: %v", err)
	}
	actions := actionsByPath(result)
	if got := actions["SOUL.md"]; got != ActionWouldSkip {
		t.Fatalf("SOUL.md dry-run action = %q, want %q", got, ActionWouldSkip)
	}
	if got := actions[filepath.Join("memory", "USER.md")]; got != ActionWouldCreate {
		t.Fatalf("memory/USER.md dry-run action = %q, want %q", got, ActionWouldCreate)
	}
	if _, err := os.Stat(filepath.Join(target, "memory")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created memory dir or returned unexpected error: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read SOUL.md: %v", err)
	}
	if string(body) != "custom persona\n" {
		t.Fatalf("dry-run changed SOUL.md:\n%s", body)
	}

	forced, err := ApplyDefaultTemplates(WriteOptions{TargetDir: target, DryRun: true, Force: true})
	if err != nil {
		t.Fatalf("ApplyDefaultTemplates dry-run force: %v", err)
	}
	if got := actionsByPath(forced)["SOUL.md"]; got != ActionWouldOverwrite {
		t.Fatalf("SOUL.md force dry-run action = %q, want %q", got, ActionWouldOverwrite)
	}
}

func sortedTemplatePaths(files []FileTemplate) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	slices.Sort(paths)
	return paths
}

func actionsByPath(result WriteResult) map[string]Action {
	actions := make(map[string]Action, len(result.Files))
	for _, file := range result.Files {
		actions[file.Path] = file.Action
	}
	return actions
}
