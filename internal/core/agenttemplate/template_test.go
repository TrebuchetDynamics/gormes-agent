package agenttemplate

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
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
	if soul := got["SOUL.md"].Content; !strings.HasPrefix(soul, llm.DefaultSoulMD) {
		t.Fatalf("SOUL.md template must start with llm.DefaultSoulMD\n--- got ---\n%s\n--- want prefix ---\n%s", soul, llm.DefaultSoulMD)
	}
	if soul := got["SOUL.md"].Content; !strings.Contains(soul, "You are Gorm,") || !strings.Contains(soul, "run by gormes") || !strings.Contains(soul, "helpful, knowledgeable, and direct") {
		t.Fatalf("SOUL.md template does not carry the Hermes-derived persona defaults:\n%s", soul)
	}
}

func TestAgentTemplateSoulTemplateKeepsWorkflowOutOfSoul(t *testing.T) {
	var soul string
	for _, file := range DefaultFiles() {
		if filepath.ToSlash(file.Path) == "SOUL.md" {
			soul = file.Content
			break
		}
	}
	if soul == "" {
		t.Fatal("SOUL.md template missing")
	}
	for _, want := range []string{
		"## Personality And Boundaries",
		"Be direct; short answers unless the user asks for detail.",
		"Never send messages, book appointments, spend money, sign up for services, or delete files without showing the plan and getting explicit approval.",
		"If access or evidence is missing, say so; do not guess or pretend to check unavailable systems.",
		"Save durable facts and preferences to memory when the user asks or the fact will matter later; never store secrets.",
		"Keep workflow and project rules in AGENTS.md, IDENTITY.md, or TOOLS.md so SOUL.md stays short.",
	} {
		if !strings.Contains(soul, want) {
			t.Fatalf("SOUL.md missing lean personality/boundary line %q:\n%s", want, soul)
		}
	}
	for _, forbidden := range []string{
		"## Operating Style",
		"Read the local `AGENTS.md`",
		"Use tools when they improve correctness",
		"State assumptions when context is incomplete",
		"When a workspace adds more specific instructions",
	} {
		if strings.Contains(soul, forbidden) {
			t.Fatalf("SOUL.md should keep workflow-specific instruction %q out of the always-loaded persona:\n%s", forbidden, soul)
		}
	}
	if lines := nonBlankLineCount(soul); lines > 8 {
		t.Fatalf("SOUL.md starter template has %d non-blank lines, want <= 8 so fresh installs stay lean:\n%s", lines, soul)
	}
}

func TestAgentTemplateDefaultFilesAreFreshInstallReady(t *testing.T) {
	files := DefaultFiles()
	got := map[string]string{}
	for _, file := range files {
		got[filepath.ToSlash(file.Path)] = file.Content
	}

	for path, wants := range map[string][]string{
		"SOUL.md": {
			"## Personality And Boundaries",
			"short answers unless the user asks for detail",
			"do not guess",
			"never store secrets",
		},
		"AGENTS.md": {
			"agents run by `gormes`",
			"## How To Work Here",
			"## Git And Files",
			"Do not discard user changes",
			"do not create branches or worktrees unless the user asks",
		},
		"IDENTITY.md": {
			"## Agent",
			"Name: Gorm",
			"Runtime: gormes",
			"## Workspace",
			"## Update Rules",
			"Do not store secrets",
		},
		"TOOLS.md": {
			"## Search And Reading",
			"## External Facts",
			"## Verification",
			"web_search",
		},
		"memory/USER.md": {
			"## Stable Preferences",
			"Do not store secrets",
		},
		"memory/MEMORY.md": {
			"## Durable Facts",
			"## Procedures",
			"do not store task progress",
		},
	} {
		body, ok := got[path]
		if !ok {
			t.Fatalf("missing template %s", path)
		}
		for _, want := range wants {
			if !containsFold(body, want) {
				t.Fatalf("%s missing fresh-install marker %q:\n%s", path, want, body)
			}
		}
	}

	combined := strings.Join([]string{got["SOUL.md"], got["AGENTS.md"], got["IDENTITY.md"], got["TOOLS.md"]}, "\n")
	for _, forbidden := range []string{
		"short-lived branch",
		"active Gormes development environment",
		"This workspace is for Gormes development",
		"progress.json contract before broad assumptions",
		"Gormes agents",
		"Name: Gormes Agent",
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("fresh-install templates contain stale project-specific guidance %q:\n%s", forbidden, combined)
		}
	}
}

func TestAgentTemplateDefaultFilesExposeStableRegistryIDs(t *testing.T) {
	files := DefaultFiles()
	byID := map[string]FileTemplate{}
	for _, file := range files {
		if strings.TrimSpace(file.ID) == "" {
			t.Fatalf("template %q is missing a stable registry ID", file.Path)
		}
		if _, exists := byID[file.ID]; exists {
			t.Fatalf("duplicate template ID %q", file.ID)
		}
		byID[file.ID] = file
	}

	for id, wantPath := range map[string]string{
		"soul":          "SOUL.md",
		"agents":        "AGENTS.md",
		"identity":      "IDENTITY.md",
		"tools":         "TOOLS.md",
		"memory-user":   "memory/USER.md",
		"memory-memory": "memory/MEMORY.md",
	} {
		file, ok := byID[id]
		if !ok {
			t.Fatalf("missing template ID %q; got IDs %v", id, sortedTemplateIDs(files))
		}
		if got := filepath.ToSlash(file.Path); got != wantPath {
			t.Fatalf("template ID %q path = %q, want %q", id, got, wantPath)
		}
	}
}

func TestAgentTemplatePairManifestCoversDefaultFiles(t *testing.T) {
	manifest := TemplatePairManifest()
	if len(manifest) == 0 {
		t.Fatal("template pair manifest must not be empty")
	}

	byPath := map[string]TemplatePair{}
	for _, pair := range manifest {
		if strings.TrimSpace(pair.TemplateID) == "" {
			t.Fatalf("%s missing template ID", pair.Path)
		}
		path := filepath.ToSlash(filepath.Clean(pair.Path))
		if pair.Path != path {
			t.Fatalf("manifest path %q must be slash-cleaned as %q", pair.Path, path)
		}
		if _, exists := byPath[pair.Path]; exists {
			t.Fatalf("duplicate manifest path %q", pair.Path)
		}
		if !isExpectedTemplatePairStatus(pair.Status) {
			t.Fatalf("%s has unsupported parity status %q", pair.Path, pair.Status)
		}
		if pair.Status != TemplatePairByteEquivalent {
			if strings.TrimSpace(pair.TransformReason) == "" {
				t.Fatalf("%s status %q requires a transform reason", pair.Path, pair.Status)
			}
		}
		if strings.TrimSpace(pair.OwnerRow) == "" {
			t.Fatalf("%s missing owner row", pair.Path)
		}
		if len(pair.TestGate) == 0 {
			t.Fatalf("%s missing test gate", pair.Path)
		}
		if len(pair.HermesSources) == 0 {
			t.Fatalf("%s missing Hermes source references", pair.Path)
		}
		if len(pair.GormesSources) == 0 {
			t.Fatalf("%s missing Gormes source references", pair.Path)
		}
		if strings.TrimSpace(pair.Contract) == "" {
			t.Fatalf("%s missing parity contract", pair.Path)
		}
		byPath[pair.Path] = pair
	}

	for _, file := range DefaultFiles() {
		path := filepath.ToSlash(file.Path)
		pair, ok := byPath[path]
		if !ok {
			t.Fatalf("default template %q is missing from the template pair manifest", path)
		}
		if pair.TemplateID != file.ID {
			t.Fatalf("%s manifest template ID = %q, want %q", path, pair.TemplateID, file.ID)
		}
		if !slices.Contains(pair.GormesSources, "internal/core/agenttemplate/default_templates.go") {
			t.Fatalf("%s manifest must point back to internal/core/agenttemplate/default_templates.go: %+v", path, pair)
		}
	}

	soul := byPath["SOUL.md"]
	if soul.Status != TemplatePairTransformed {
		t.Fatalf("SOUL.md status = %q, want %q", soul.Status, TemplatePairTransformed)
	}
	if !slices.Contains(soul.HermesSources, "hermes_cli/default_soul.py") {
		t.Fatalf("SOUL.md manifest must cite Hermes DEFAULT_SOUL_MD source: %+v", soul)
	}
	if !slices.Contains(soul.GormesSources, "internal/llm/default_soul.go") {
		t.Fatalf("SOUL.md manifest must cite the Gormes source-paired default: %+v", soul)
	}

	for _, path := range []string{"AGENTS.md", "IDENTITY.md", "TOOLS.md", "memory/USER.md", "memory/MEMORY.md"} {
		pair := byPath[path]
		if pair.Status != TemplatePairGormesOwned {
			t.Fatalf("%s status = %q, want %q because Hermes consumes or inspires this context but does not seed the same file", path, pair.Status, TemplatePairGormesOwned)
		}
		if !strings.Contains(strings.ToLower(pair.Contract), "gormes") {
			t.Fatalf("%s owned-divergence contract must explain the Gormes-owned behavior: %s", path, pair.Contract)
		}
	}
}

func TestAgentTemplatePairManifestValidatesSourceReferences(t *testing.T) {
	repoRoot, hermesRoot := templatePairSourceRoots(t)
	opts := TemplatePairValidationOptions{
		RepoRoot:   repoRoot,
		HermesRoot: hermesRoot,
	}

	if err := ValidateTemplatePairs(TemplatePairManifest(), opts); err != nil {
		t.Fatalf("template pair manifest source references must resolve: %v", err)
	}

	stale := []TemplatePair{
		{
			Path:            "SOUL.md",
			TemplateID:      "soul",
			Status:          TemplatePairTransformed,
			HermesSources:   []string{"hermes_cli/missing_default_soul.py"},
			GormesSources:   []string{"internal/core/agenttemplate/default_templates.go"},
			TransformReason: "Gormes replaces Hermes identity with the Gorm persona.",
			TestGate:        []string{"go test ./internal/core/agenttemplate -count=1"},
			OwnerRow:        "Gormes agent template reset command",
			Contract:        "Gormes fixture for stale source validation.",
		},
	}
	err := ValidateTemplatePairs(stale, opts)
	if err == nil {
		t.Fatal("ValidateTemplatePairs accepted a stale Hermes source reference")
	}
	if got := err.Error(); !strings.Contains(got, "missing Hermes source") || !strings.Contains(got, "hermes_cli/missing_default_soul.py") {
		t.Fatalf("stale source error = %q, want missing Hermes source path", got)
	}
}

func TestAgentTemplateApplyRejectsUnsafeTemplatePath(t *testing.T) {
	target := t.TempDir()
	_, err := ApplyTemplates(WriteOptions{TargetDir: target}, []FileTemplate{{ID: "bad", Path: "../SOUL.md", Content: "bad"}})
	if err == nil || !strings.Contains(err.Error(), "invalid agent template path") {
		t.Fatalf("ApplyTemplates unsafe path error = %v, want invalid path", err)
	}
	if _, statErr := os.Stat(filepath.Join(target, "..", "SOUL.md")); !os.IsNotExist(statErr) {
		t.Fatalf("unsafe template path wrote outside target or stat failed unexpectedly: %v", statErr)
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
	if strings.Contains(string(body), "custom persona") || !strings.Contains(string(body), "You are Gorm,") {
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

func sortedTemplateIDs(files []FileTemplate) []string {
	ids := make([]string, 0, len(files))
	for _, file := range files {
		ids = append(ids, file.ID)
	}
	slices.Sort(ids)
	return ids
}

func isExpectedTemplatePairStatus(status TemplatePairStatus) bool {
	switch status {
	case TemplatePairByteEquivalent, TemplatePairTransformed, TemplatePairGormesOwned, TemplatePairNotApplicable, TemplatePairBlocked:
		return true
	default:
		return false
	}
}

func actionsByPath(result WriteResult) map[string]Action {
	actions := make(map[string]Action, len(result.Files))
	for _, file := range result.Files {
		actions[file.Path] = file.Action
	}
	return actions
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func nonBlankLineCount(s string) int {
	count := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func templatePairSourceRoots(t *testing.T) (string, string) {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		t.Fatalf("repo root %s missing go.mod: %v", repoRoot, err)
	}
	hermesRoot := filepath.Join(repoRoot, "hermes-agent")
	if _, err := os.Stat(filepath.Join(hermesRoot, "hermes_cli", "default_soul.py")); err != nil {
		t.Skipf("upstream Hermes checkout unavailable for manifest source validation: %v", err)
	}
	return repoRoot, hermesRoot
}
