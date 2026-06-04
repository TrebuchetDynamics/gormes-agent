package prompttemplates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestCatalogDiscoversHomeProjectAndExplicitPromptTemplates(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	writePromptTemplateFixture(t, filepath.Join(home, "prompts", "review.md"), "---\ndescription: Review from home\nargument-hint: '<scope>'\n---\nReview $1\n")
	writePromptTemplateFixture(t, filepath.Join(cwd, ".gormes", "prompts", "component.md"), "---\ndescription: Build component\n---\nBuild $1\n")
	writePromptTemplateFixture(t, filepath.Join(home, "prompts", "skills.md"), "shadow built-in\n")
	dir := t.TempDir()
	file := filepath.Join(t.TempDir(), "adhoc.md")
	writePromptTemplateFixture(t, filepath.Join(dir, "from-dir.md"), "---\ndescription: From dir\n---\nDir $1\n")
	writePromptTemplateFixture(t, file, "---\ndescription: From file\n---\nFile $1\n")

	catalog := Catalog(config.Config{}, cwd, CatalogOptions{Paths: []string{dir, file}})
	for _, name := range []string{"review", "component", "from-dir", "adhoc"} {
		if _, ok := catalog.Lookup(name); !ok {
			t.Fatalf("template %q missing: %+v", name, catalog.Templates)
		}
	}
	if _, ok := catalog.Lookup("skills"); ok {
		t.Fatalf("built-in /skills collision must be skipped: %+v", catalog.Templates)
	}
}

func TestCatalogDisabledReturnsEmptyCatalog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	writePromptTemplateFixture(t, filepath.Join(home, "prompts", "review.md"), "---\ndescription: Review\n---\nReview $1\n")

	catalog := Catalog(config.Config{}, t.TempDir(), CatalogOptions{Disabled: true})
	if len(catalog.Templates) != 0 {
		t.Fatalf("disabled catalog templates = %+v, want none", catalog.Templates)
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
