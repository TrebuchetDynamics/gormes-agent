package prompttemplates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestCatalogDiscoversHomeProjectAndExplicitPaths(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	extra := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	writePromptTemplateFixture(t, filepath.Join(home, "prompts", "review.md"), "---\ndescription: Review\n---\nReview $1\n")
	writePromptTemplateFixture(t, filepath.Join(cwd, ".gormes", "prompts", "component.md"), "---\ndescription: Component\n---\nComponent $1\n")
	writePromptTemplateFixture(t, filepath.Join(extra, "adhoc.md"), "---\ndescription: Adhoc\n---\nAdhoc $1\n")
	writePromptTemplateFixture(t, filepath.Join(home, "prompts", "skills.md"), "shadow built-in\n")

	catalog := Catalog(config.Config{}, cwd, CatalogOptions{Paths: []string{extra}})
	for _, name := range []string{"review", "component", "adhoc"} {
		if _, ok := catalog.Lookup(name); !ok {
			t.Fatalf("template %q missing: %+v", name, catalog.Templates)
		}
	}
	if _, ok := catalog.Lookup("skills"); ok {
		t.Fatalf("built-in collision should be skipped: %+v", catalog.Templates)
	}
}

func TestCatalogDisabled(t *testing.T) {
	catalog := Catalog(config.Config{}, t.TempDir(), CatalogOptions{Disabled: true, Paths: []string{t.TempDir()}})
	if len(catalog.Templates) != 0 {
		t.Fatalf("disabled catalog templates = %+v", catalog.Templates)
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
