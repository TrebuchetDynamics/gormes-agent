package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

func TestTUIWiresPromptTemplateDiscovery(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	writePromptTemplateFixture(t, filepath.Join(home, "prompts", "review.md"), "---\ndescription: Review from home\nargument-hint: '<scope>'\n---\nReview $1\n")
	writePromptTemplateFixture(t, filepath.Join(cwd, ".gormes", "prompts", "component.md"), "---\ndescription: Build component\n---\nBuild $1\n")
	writePromptTemplateFixture(t, filepath.Join(home, "prompts", "skills.md"), "shadow built-in\n")

	catalog := tuiPromptTemplateCatalog(config.Config{}, cwd, promptTemplateCatalogOptions{})
	if _, ok := catalog.Lookup("review"); !ok {
		t.Fatalf("home prompt template missing: %+v", catalog.Templates)
	}
	if _, ok := catalog.Lookup("component"); !ok {
		t.Fatalf("project prompt template missing: %+v", catalog.Templates)
	}
	if _, ok := catalog.Lookup("skills"); ok {
		t.Fatalf("built-in /skills collision must be skipped: %+v", catalog.Templates)
	}
}

func TestTUIWiresPromptTemplateFlags(t *testing.T) {
	setupTUIModelOverrideTestEnv(t)
	dir := t.TempDir()
	file := filepath.Join(t.TempDir(), "adhoc.md")
	writePromptTemplateFixture(t, filepath.Join(dir, "from-dir.md"), "---\ndescription: From dir\n---\nDir $1\n")
	writePromptTemplateFixture(t, file, "---\ndescription: From file\n---\nFile $1\n")

	var got tuiInvocation
	cmd := newRootCommandWithRuntime(rootRuntime{
		runResolvedTUI: func(_ *cobra.Command, invocation tuiInvocation) error {
			got = invocation
			return nil
		},
	})
	stdout, stderr, err := executeTUIModelOverrideCommand(cmd, "--offline", "--prompt-template", dir, "--prompt-template", file)
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if len(got.PromptTemplatePaths) != 2 || got.PromptTemplatePaths[0] != dir || got.PromptTemplatePaths[1] != file {
		t.Fatalf("PromptTemplatePaths = %#v, want dir+file", got.PromptTemplatePaths)
	}
	catalog := tuiPromptTemplateCatalog(got.Config, t.TempDir(), promptTemplateCatalogOptions{Paths: got.PromptTemplatePaths})
	if _, ok := catalog.Lookup("from-dir"); !ok {
		t.Fatalf("--prompt-template directory not discovered: %+v", catalog.Templates)
	}
	if _, ok := catalog.Lookup("adhoc"); !ok {
		t.Fatalf("--prompt-template file not discovered: %+v", catalog.Templates)
	}

	cmd = newRootCommandWithRuntime(rootRuntime{
		runResolvedTUI: func(_ *cobra.Command, invocation tuiInvocation) error {
			got = invocation
			return nil
		},
	})
	stdout, stderr, err = executeTUIModelOverrideCommand(cmd, "--offline", "--no-prompt-templates", "--prompt-template", file)
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if !got.NoPromptTemplates {
		t.Fatal("NoPromptTemplates = false, want true")
	}
	disabled := tuiPromptTemplateCatalog(got.Config, t.TempDir(), promptTemplateCatalogOptions{Paths: got.PromptTemplatePaths, Disabled: got.NoPromptTemplates})
	if len(disabled.Templates) != 0 {
		t.Fatalf("disabled catalog templates = %+v, want none", disabled.Templates)
	}
}

func writePromptTemplateFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
