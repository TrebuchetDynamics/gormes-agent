package skills_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

func TestTouchDesignerSkill_ParsesBundledMetadata(t *testing.T) {
	raw := readBundledTouchDesignerSkill(t)

	skill, err := skills.Parse(raw, 64*1024)
	if err != nil {
		t.Fatalf("Parse(TouchDesigner SKILL.md) error = %v", err)
	}
	if skill.Name != "touchdesigner-mcp" {
		t.Fatalf("Name = %q, want touchdesigner-mcp", skill.Name)
	}
	wantDesc := "Control a running TouchDesigner instance via twozero MCP — create operators, set parameters, wire connections, execute Python, build real-time visuals. 36 native tools."
	if skill.Description != wantDesc {
		t.Fatalf("Description = %q\nwant: %q", skill.Description, wantDesc)
	}
	if got := stringField(t, skill, "ReviewState"); got != "reviewed" {
		t.Fatalf("ReviewState = %q, want reviewed", got)
	}

	tags := stringSliceField(t, skill, "HermesTags")
	for _, want := range []string{"TouchDesigner", "MCP", "twozero"} {
		if !containsString(tags, want) {
			t.Fatalf("HermesTags = %#v, missing %q", tags, want)
		}
	}

	creds := credentialGroupsString(t, skill)
	for _, want := range []string{"TOUCHDESIGNER_MCP_URL", "TWOZERO_MCP_URL"} {
		if !strings.Contains(creds, want) {
			t.Fatalf("CredentialGroups = %s, want %s any-of group", creds, want)
		}
	}

	for _, want := range []string{
		"# TouchDesigner Integration",
		"twozero MCP",
		"36 native tools",
	} {
		if !strings.Contains(skill.Body, want) {
			t.Fatalf("TouchDesigner body missing %q", want)
		}
	}
}

func TestTouchDesignerSkill_MovedFromOptionalCatalog(t *testing.T) {
	activeRoot := t.TempDir()
	bundledRoot, err := filepath.Abs(filepath.Join("..", "..", "skills"))
	if err != nil {
		t.Fatalf("Abs(bundled skills): %v", err)
	}
	t.Setenv("GORMES_SKILLS_ROOT", activeRoot)
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", bundledRoot)
	clearEnvVars(t, "TOUCHDESIGNER_MCP_URL", "TWOZERO_MCP_URL")

	rows := skills.ListInstalledSkills(skills.ListOptions{Source: "builtin"}, nil)
	row, ok := findSkillRow(rows, "touchdesigner-mcp")
	if !ok {
		t.Fatalf("bundled TouchDesigner catalog row missing from rows: %#v", rows)
	}
	if row.Category != "creative" {
		t.Fatalf("Category = %q, want creative", row.Category)
	}
	if row.Source != "builtin" {
		t.Fatalf("Source = %q, want builtin (moved from optional/community)", row.Source)
	}
	if row.Trust != "system" {
		t.Fatalf("Trust = %q, want system", row.Trust)
	}
	if !strings.HasSuffix(filepath.ToSlash(row.Path), "skills/creative/touchdesigner-mcp/SKILL.md") {
		t.Fatalf("Path = %q, want shipped bundled SKILL.md path", row.Path)
	}

	hubRows := skills.ListInstalledSkills(skills.ListOptions{Source: "hub"}, nil)
	if _, found := findSkillRow(hubRows, "touchdesigner-mcp"); found {
		t.Fatalf("touchdesigner-mcp must not appear under hub/community source: %#v", hubRows)
	}
	localRows := skills.ListInstalledSkills(skills.ListOptions{Source: "local"}, nil)
	if _, found := findSkillRow(localRows, "touchdesigner-mcp"); found {
		t.Fatalf("touchdesigner-mcp must not appear under local source: %#v", localRows)
	}
}

func TestTouchDesignerSkill_MissingPrerequisitesUnavailable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", root)
	clearEnvVars(t, "TOUCHDESIGNER_MCP_URL", "TWOZERO_MCP_URL")
	installTouchDesignerSkillFixture(t, root)

	rows := skills.ListInstalledSkills(skills.ListOptions{Source: "builtin"}, nil)
	row, ok := findSkillRow(rows, "touchdesigner-mcp")
	if !ok {
		t.Fatalf("TouchDesigner catalog row missing from rows: %#v", rows)
	}
	if row.Status != skills.SkillStatusMissingPrerequisite {
		t.Fatalf("Status = %q, want %q (catalog row stays visible)", row.Status, skills.SkillStatusMissingPrerequisite)
	}

	runtime := skills.NewRuntime(root, 64*1024, 3, "")
	_, _, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "touchdesigner mcp build network", skills.RuntimeOptions{
		Env: map[string]string{},
	})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() error = %v", err)
	}

	status := findSkillStatus(t, statuses, "touchdesigner-mcp")
	if status.Status != skills.SkillStatusMissingPrerequisite {
		t.Fatalf("status = %q, want %q", status.Status, skills.SkillStatusMissingPrerequisite)
	}
	if !strings.Contains(status.Reason, "TOUCHDESIGNER_MCP_URL") || !strings.Contains(status.Reason, "TWOZERO_MCP_URL") {
		t.Fatalf("missing prerequisite reason = %q, want both TOUCHDESIGNER_MCP_URL and TWOZERO_MCP_URL evidence", status.Reason)
	}
}

func TestTouchDesignerSkill_PromptExcludedWhenUnavailable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", root)
	clearEnvVars(t, "TOUCHDESIGNER_MCP_URL", "TWOZERO_MCP_URL")
	installTouchDesignerSkillFixture(t, root)

	runtime := skills.NewRuntime(root, 64*1024, 3, "")
	block, names, _, err := runtime.BuildSkillBlockWithOptions(context.Background(), "touchdesigner mcp twozero generative visuals", skills.RuntimeOptions{
		Env: map[string]string{},
	})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() error = %v", err)
	}
	for _, name := range names {
		if name == "touchdesigner-mcp" {
			t.Fatalf("names = %#v, must not include touchdesigner-mcp while unavailable", names)
		}
	}
	if strings.Contains(block, "## touchdesigner-mcp") || strings.Contains(block, "td_execute_python") {
		t.Fatalf("block injected unavailable TouchDesigner instructions:\n%s", block)
	}

	available := map[string]string{"TOUCHDESIGNER_MCP_URL": "http://localhost:40404/mcp"}
	block2, names2, _, err := runtime.BuildSkillBlockWithOptions(context.Background(), "touchdesigner mcp twozero generative visuals", skills.RuntimeOptions{
		Env: available,
	})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions(available) error = %v", err)
	}
	if !containsString(names2, "touchdesigner-mcp") {
		t.Fatalf("names = %#v, want touchdesigner-mcp once prerequisite present", names2)
	}
	if !strings.Contains(block2, "## touchdesigner-mcp") {
		t.Fatalf("block missing TouchDesigner section once available:\n%s", block2)
	}
}

func TestTouchDesignerSkill_ReferenceFilesAreInertSkillAssets(t *testing.T) {
	bundledRoot, err := filepath.Abs(filepath.Join("..", "..", "skills"))
	if err != nil {
		t.Fatalf("Abs(bundled skills): %v", err)
	}
	skillDir := filepath.Join(bundledRoot, "creative", "touchdesigner-mcp")
	refsPath := filepath.Join(skillDir, "references", "mcp-tools.md")
	raw, err := os.ReadFile(refsPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", refsPath, err)
	}
	if !strings.Contains(string(raw), "twozero MCP") {
		t.Fatalf("references/mcp-tools.md missing inert MCP descriptor content")
	}

	if _, err := skills.Parse(raw, 64*1024); err == nil {
		t.Fatalf("references/mcp-tools.md must not parse as a SKILL.md descriptor; got success")
	}

	for _, forbidden := range []string{"setup.sh", ".tox", "scripts/setup"} {
		if _, err := os.Stat(filepath.Join(skillDir, forbidden)); !os.IsNotExist(err) {
			t.Fatalf("bundled TouchDesigner skill must not ship %q (live MCP/TouchDesigner setup): err=%v", forbidden, err)
		}
	}

	t.Setenv("GORMES_SKILLS_ROOT", t.TempDir())
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", bundledRoot)
	rows := skills.ListInstalledSkills(skills.ListOptions{Source: "builtin"}, nil)
	for _, row := range rows {
		if strings.HasSuffix(row.Path, "mcp-tools.md") {
			t.Fatalf("references/mcp-tools.md must not be promoted as its own skill row: %#v", row)
		}
	}
}

func readBundledTouchDesignerSkill(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "skills", "creative", "touchdesigner-mcp", "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return raw
}

func installTouchDesignerSkillFixture(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "active", "creative", "touchdesigner-mcp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), readBundledTouchDesignerSkill(t), 0o644); err != nil {
		t.Fatalf("WriteFile(TouchDesigner SKILL.md): %v", err)
	}
	meta := `{"category":"creative","source":"builtin","trust":"system"}`
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatalf("WriteFile(meta.json): %v", err)
	}
}
