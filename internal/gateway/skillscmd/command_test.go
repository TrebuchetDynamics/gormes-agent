package skillscmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

func TestSkillsCommandNeedsInstalledRootsUnderstandsCommandTokenVariants(t *testing.T) {
	for _, body := range []string{"/skills list", "／skills list", "/skills@GormesBot inspect planner"} {
		if !skillsCommandNeedsInstalledRoots(body) {
			t.Fatalf("skillsCommandNeedsInstalledRoots(%q) = false, want true", body)
		}
	}
}

func TestHandleSkillsCommandAcceptsFullwidthSlashPrefix(t *testing.T) {
	root := t.TempDir()
	writeSkillCommandSkill(t, root, "active/review-skill", "review-skill", "Review files", "Review.")

	out := HandleSkillsCommandWithOptions(context.Background(), "／skills list", SkillsCommandOptions{SkillsRoot: root})
	if !strings.Contains(out, "Installed Skills") || !strings.Contains(out, "review-skill") {
		t.Fatalf("fullwidth /skills prefix did not dispatch list:\n%s", out)
	}
	if strings.Contains(out, "Unknown /skills subcommand") {
		t.Fatalf("fullwidth /skills prefix treated as unknown:\n%s", out)
	}
}

func TestHandleSkillsCommandWithOptionsAllowsNilContext(t *testing.T) {
	out := HandleSkillsCommandWithOptions(nil, "/skills search planner", SkillsCommandOptions{
		HubProviders: []skills.HubRegistryProvider{skillCommandContextCheckingProvider{}},
	})
	if !strings.Contains(out, "Skill Hub Search") || !strings.Contains(out, "planner") {
		t.Fatalf("nil-context skills search output missing expected content:\n%s", out)
	}
}

func TestHandleSkillsUnknownSubcommandSanitizesToken(t *testing.T) {
	out := HandleSkillsCommandWithOptions(context.Background(), "/skills bad`**cmd**", SkillsCommandOptions{})
	for _, forbidden := range []string{"bad`**cmd**", "**cmd**"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("unknown subcommand leaked unsafe token %q:\n%s", forbidden, out)
		}
	}
	if !strings.Contains(out, "Unknown /skills subcommand: \"bad'''cmd''\"") {
		t.Fatalf("unknown subcommand missing sanitized token:\n%s", out)
	}
}

func TestHandleSkillsInspectSanitizesMetadataFields(t *testing.T) {
	root := t.TempDir()
	writeSkillCommandSkill(t, root, "active/unsafe-skill", "unsafe-skill", "**Injected:** `token`", "Body remains visible.")

	out := HandleSkillsCommandWithOptions(context.Background(), "/skills inspect unsafe-skill", SkillsCommandOptions{SkillsRoot: root})
	for _, forbidden := range []string{"**Injected:**", "`token`"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("inspect output leaked unsafe metadata %q:\n%s", forbidden, out)
		}
	}
	if !strings.Contains(out, "Description: ''Injected:'' 'token'") {
		t.Fatalf("inspect output missing sanitized description:\n%s", out)
	}
}

func TestHandleSkillsInspectSanitizesBodyPreview(t *testing.T) {
	root := t.TempDir()
	writeSkillCommandSkill(t, root, "active/body-skill", "body-skill", "Body skill", "# Injected\napi_key=plain-secret-token\nUse `danger`.")

	out := HandleSkillsCommandWithOptions(context.Background(), "/skills inspect body-skill", SkillsCommandOptions{SkillsRoot: root})
	for _, forbidden := range []string{"# Injected", "api_key=plain-secret-token", "plain-secret-token", "`danger`"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("inspect output leaked unsafe body preview %q:\n%s", forbidden, out)
		}
	}
	if !strings.Contains(out, "[redacted]") || !strings.Contains(out, "Use 'danger'.") {
		t.Fatalf("inspect output missing sanitized/redacted body preview:\n%s", out)
	}
}

func TestHandleSkillsInspectBodyPreviewKeepsUTF8Boundary(t *testing.T) {
	root := t.TempDir()
	body := strings.Repeat("a", 1999) + "🙂tail"
	writeSkillCommandSkill(t, root, "active/utf8-skill", "utf8-skill", "UTF-8 skill", body)

	out := HandleSkillsCommandWithOptions(context.Background(), "/skills inspect utf8-skill", SkillsCommandOptions{SkillsRoot: root})

	if !utf8.ValidString(out) {
		t.Fatalf("inspect output is not valid UTF-8 near body boundary: %q", out)
	}
	if !strings.Contains(out, strings.Repeat("a", 1999)+"🙂") {
		t.Fatalf("inspect output dropped complete boundary rune; output suffix=%q", out[len(out)-min(len(out), 120):])
	}
	if strings.Contains(out, string(utf8.RuneError)+"tail") {
		t.Fatalf("inspect output contains replacement rune from split UTF-8 body: %q", out)
	}
}

type skillCommandContextCheckingProvider struct{}

func (skillCommandContextCheckingProvider) Snapshot(ctx context.Context) ([]skills.HubSearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []skills.HubSearchResult{{Name: "planner", Description: "Plan safely", InstallID: "planner", Score: 1}}, nil
}

func writeSkillCommandSkill(t *testing.T, root, rel, name, description, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel), "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	raw := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}
