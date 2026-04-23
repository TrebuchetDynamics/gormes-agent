package skills

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestFilesystemSourceListSkillsBuildsDeterministicCatalog(t *testing.T) {
	root := t.TempDir()
	writeSkillDoc(t, filepath.Join(root, "research", "arxiv", "SKILL.md"), "arxiv", "Search and retrieve arXiv papers", "Use arXiv APIs and summarize relevant papers.")
	writeSkillDoc(t, filepath.Join(root, "github", "github-pr-workflow", "SKILL.md"), "github-pr-workflow", "Manage pull request workflows", "Create, review, and merge pull requests.")

	source := NewFilesystemSource(FilesystemSourceConfig{
		Name:             "official",
		Title:            "Official optional skills",
		Prefix:           "official",
		Root:             root,
		Optional:         true,
		Readiness:        SkillReadinessAvailable,
		MaxDocumentBytes: 8 * 1024,
	})

	got, err := source.ListSkills(context.Background())
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}

	want := []SkillMeta{
		{
			Source:      "official",
			Ref:         "official/github/github-pr-workflow",
			Name:        "github-pr-workflow",
			Description: "Manage pull request workflows",
			Category:    "github",
			Path:        filepath.Join(root, "github", "github-pr-workflow", "SKILL.md"),
			Optional:    true,
			Readiness:   SkillReadinessAvailable,
		},
		{
			Source:      "official",
			Ref:         "official/research/arxiv",
			Name:        "arxiv",
			Description: "Search and retrieve arXiv papers",
			Category:    "research",
			Path:        filepath.Join(root, "research", "arxiv", "SKILL.md"),
			Optional:    true,
			Readiness:   SkillReadinessAvailable,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListSkills() = %#v, want %#v", got, want)
	}
}

func TestSourceRegistryLockBuildsDeterministicSnapshot(t *testing.T) {
	root := t.TempDir()
	writeSkillDoc(t, filepath.Join(root, "creative", "ascii-art", "SKILL.md"), "ascii-art", "Generate ASCII art", "Use local tools first and fall back only when needed.")

	registry, err := NewSourceRegistry(
		NewStaticSource(
			SkillBundle{Name: "clawhub", Title: "ClawHub", Readiness: SkillReadinessAvailable},
			[]SkillMeta{{
				Source:      "clawhub",
				Ref:         "clawhub/devops/docker-management",
				Name:        "docker-management",
				Description: "Manage Docker from the CLI",
				Category:    "devops",
				Readiness:   SkillReadinessAvailable,
			}},
		),
		NewFilesystemSource(FilesystemSourceConfig{
			Name:             "bundled",
			Title:            "Bundled skills",
			Prefix:           "bundled",
			Root:             root,
			Readiness:        SkillReadinessReady,
			MaxDocumentBytes: 8 * 1024,
		}),
	)
	if err != nil {
		t.Fatalf("NewSourceRegistry() error = %v", err)
	}

	lock, err := registry.Lock(context.Background(), time.Date(2026, 4, 23, 16, 17, 14, 0, time.UTC))
	if err != nil {
		t.Fatalf("Lock() error = %v", err)
	}

	if lock.Version != 1 {
		t.Fatalf("Version = %d, want 1", lock.Version)
	}
	if !lock.GeneratedAt.Equal(time.Date(2026, 4, 23, 16, 17, 14, 0, time.UTC)) {
		t.Fatalf("GeneratedAt = %s", lock.GeneratedAt)
	}

	wantBundles := []SkillBundle{
		{Name: "bundled", Title: "Bundled skills", Prefix: "bundled", Root: root, Readiness: SkillReadinessReady},
		{Name: "clawhub", Title: "ClawHub", Readiness: SkillReadinessAvailable},
	}
	if !reflect.DeepEqual(lock.Bundles, wantBundles) {
		t.Fatalf("Bundles = %#v, want %#v", lock.Bundles, wantBundles)
	}

	wantRefs := []string{
		"bundled/creative/ascii-art",
		"clawhub/devops/docker-management",
	}
	var gotRefs []string
	for _, skill := range lock.Skills {
		gotRefs = append(gotRefs, skill.Ref)
	}
	if !reflect.DeepEqual(gotRefs, wantRefs) {
		t.Fatalf("skill refs = %#v, want %#v", gotRefs, wantRefs)
	}
}

func TestBuiltinSkillRegistryBundlesCoverTrackedSources(t *testing.T) {
	got := BuiltinSkillRegistryBundles()
	want := []SkillBundle{
		{Name: "bundled", Title: "Bundled skills", Prefix: "bundled"},
		{Name: "official", Title: "Official optional skills", Prefix: "official", Optional: true},
		{Name: "claude-marketplace", Title: "Claude Marketplace"},
		{Name: "clawhub", Title: "ClawHub"},
		{Name: "github", Title: "GitHub"},
		{Name: "hermes-index", Title: "Hermes Index"},
		{Name: "lobehub", Title: "LobeHub"},
		{Name: "skills.sh", Title: "skills.sh"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuiltinSkillRegistryBundles() = %#v, want %#v", got, want)
	}
}
