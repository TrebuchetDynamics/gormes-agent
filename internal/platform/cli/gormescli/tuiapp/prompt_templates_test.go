package tuiapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestTUIWiresPromptTemplateFlags(t *testing.T) {
	setupTUIModelOverrideTestEnv(t)
	dir := t.TempDir()
	file := filepath.Join(t.TempDir(), "adhoc.md")
	writePromptTemplateFixture(t, filepath.Join(dir, "from-dir.md"), "---\ndescription: From dir\n---\nDir $1\n")
	writePromptTemplateFixture(t, file, "---\ndescription: From file\n---\nFile $1\n")

	var got Invocation
	cmd := newRootCommandWithRuntime(Runtime{
		RunResolvedTUI: func(_ *cobra.Command, invocation Invocation) error {
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
	cmd = newRootCommandWithRuntime(Runtime{
		RunResolvedTUI: func(_ *cobra.Command, invocation Invocation) error {
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
