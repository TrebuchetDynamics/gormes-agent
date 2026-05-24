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
	if got := stringSliceField(t, skill, "RequiredEnvVars"); len(got) != 0 {
		t.Fatalf("RequiredEnvVars = %#v, want no environment prerequisite for exact Hermes skill", got)
	}

	tags := stringSliceField(t, skill, "HermesTags")
	for _, want := range []string{"TouchDesigner", "MCP", "twozero"} {
		if !containsString(tags, want) {
			t.Fatalf("HermesTags = %#v, missing %q", tags, want)
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

func TestTouchDesignerSkill_NoCredentialPrerequisitesEnableCatalogAndPrompt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", root)
	clearEnvVars(t, "TOUCHDESIGNER_MCP_URL", "TWOZERO_MCP_URL")
	installTouchDesignerSkillFixture(t, root)

	rows := skills.ListInstalledSkills(skills.ListOptions{Source: "builtin"}, nil)
	row, ok := findSkillRow(rows, "touchdesigner-mcp")
	if !ok {
		t.Fatalf("TouchDesigner catalog row missing from rows: %#v", rows)
	}
	if row.Status != skills.SkillStatusEnabled {
		t.Fatalf("Status = %q, want %q for exact Hermes skill with no env prerequisite", row.Status, skills.SkillStatusEnabled)
	}

	runtime := skills.NewRuntime(root, 64*1024, 3, "")
	block, names, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "touchdesigner mcp twozero generative visuals", skills.RuntimeOptions{
		Env: map[string]string{},
	})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() error = %v", err)
	}
	if !containsString(names, "touchdesigner-mcp") {
		t.Fatalf("names = %#v, want touchdesigner-mcp without local metadata patches", names)
	}
	status := findSkillStatus(t, statuses, "touchdesigner-mcp")
	if status.Status != skills.SkillStatusAvailable {
		t.Fatalf("status = %q, want %q", status.Status, skills.SkillStatusAvailable)
	}
	if !strings.Contains(block, "## touchdesigner-mcp") || !strings.Contains(block, "td_execute_python") {
		t.Fatalf("block missing exact Hermes TouchDesigner instructions:\n%s", block)
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

	setupPath := filepath.Join(skillDir, "scripts", "setup.sh")
	setupRaw, err := os.ReadFile(setupPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", setupPath, err)
	}
	if !strings.Contains(string(setupRaw), "twozero.tox") {
		t.Fatalf("setup.sh should remain an inert exact-Hermes asset with twozero.tox setup guidance")
	}
	if _, err := os.Stat(filepath.Join(skillDir, ".tox")); !os.IsNotExist(err) {
		t.Fatalf("bundled TouchDesigner skill must not ship a live .tox binary: err=%v", err)
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
