package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPromptFragmentsIncludeResolvesThroughPrioritySearchAndVariables(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "profile")
	user := filepath.Join(root, "user")
	plugin := filepath.Join(root, "plugin")
	base := filepath.Join(root, "base")

	writePromptFragment(t, profile, "agent.system.main.md", "Profile {{agent_name}}\n{{include agent.system.main.role.md}}\n{{include \"agent.system.tool.search.md\"}}")
	writePromptFragment(t, user, "agent.system.main.role.md", "User role for {{agent_name}}")
	writePromptFragment(t, plugin, "agent.system.tool.search.md", "Plugin search")
	writePromptFragment(t, base, "agent.system.main.role.md", "Base role")

	result, err := RenderPromptFragment(PromptFragmentRequest{
		Entry: "agent.system.main.md",
		Sources: []PromptFragmentSource{
			{Name: "profile", Dir: profile},
			{Name: "user", Dir: user},
			{Name: "plugin", Dir: plugin},
			{Name: "base", Dir: base},
		},
		Variables: map[string]string{"agent_name": "Gormes"},
	})
	if err != nil {
		t.Fatalf("RenderPromptFragment() error = %v", err)
	}

	want := "Profile Gormes\nUser role for Gormes\nPlugin search"
	if result.Text != want {
		t.Fatalf("Text = %q, want %q", result.Text, want)
	}
	if len(result.Fragments) != 3 {
		t.Fatalf("Fragments = %+v, want 3 resolved fragments", result.Fragments)
	}
}

func TestPromptFragmentsIncludeOriginalChainsToLowerPrioritySource(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "profile")
	user := filepath.Join(root, "user")
	base := filepath.Join(root, "base")

	writePromptFragment(t, profile, "agent.system.main.role.md", "Profile before\n{{include original}}\nProfile after")
	writePromptFragment(t, user, "agent.system.main.role.md", "User before\n{{include original}}\nUser after")
	writePromptFragment(t, base, "agent.system.main.role.md", "Base role")

	result, err := RenderPromptFragment(PromptFragmentRequest{
		Entry: "agent.system.main.role.md",
		Sources: []PromptFragmentSource{
			{Name: "profile", Dir: profile},
			{Name: "user", Dir: user},
			{Name: "base", Dir: base},
		},
	})
	if err != nil {
		t.Fatalf("RenderPromptFragment() error = %v", err)
	}

	want := "Profile before\nUser before\nBase role\nUser after\nProfile after"
	if result.Text != want {
		t.Fatalf("Text = %q, want %q", result.Text, want)
	}
}

func TestPromptFragmentsCircularIncludeReportsChain(t *testing.T) {
	root := t.TempDir()
	writePromptFragment(t, root, "a.md", "A {{include b.md}}")
	writePromptFragment(t, root, "b.md", "B {{include a.md}}")

	_, err := RenderPromptFragment(PromptFragmentRequest{
		Entry:   "a.md",
		Sources: []PromptFragmentSource{{Name: "base", Dir: root}},
	})
	if err == nil {
		t.Fatal("RenderPromptFragment() error = nil, want circular include error")
	}
	if !strings.Contains(err.Error(), "prompt_fragment_error") ||
		!strings.Contains(err.Error(), "a.md -> b.md -> a.md") {
		t.Fatalf("error = %q, want prompt_fragment_error with include chain", err.Error())
	}
}

func TestPromptFragmentsCacheInvalidatesWhenFileMTimeChanges(t *testing.T) {
	root := t.TempDir()
	path := writePromptFragment(t, root, "agent.system.main.md", "first")
	cache := NewPromptFragmentCache()

	result, err := RenderPromptFragment(PromptFragmentRequest{
		Entry:   "agent.system.main.md",
		Sources: []PromptFragmentSource{{Name: "base", Dir: root}},
		Cache:   cache,
	})
	if err != nil {
		t.Fatalf("first RenderPromptFragment() error = %v", err)
	}
	if result.Text != "first" {
		t.Fatalf("first Text = %q, want first", result.Text)
	}

	if err := os.WriteFile(path, []byte("second"), 0o600); err != nil {
		t.Fatalf("rewrite prompt fragment: %v", err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes prompt fragment: %v", err)
	}

	result, err = RenderPromptFragment(PromptFragmentRequest{
		Entry:   "agent.system.main.md",
		Sources: []PromptFragmentSource{{Name: "base", Dir: root}},
		Cache:   cache,
	})
	if err != nil {
		t.Fatalf("second RenderPromptFragment() error = %v", err)
	}
	if result.Text != "second" {
		t.Fatalf("second Text = %q, want cache to invalidate after mtime change", result.Text)
	}
}

func writePromptFragment(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir parent %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
