package repoctl

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/fidelity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func TestHermesContractInventoryClassifiesSkillCatalogSurface(t *testing.T) {
	root := t.TempDir()
	hermes := filepath.Join(root, "hermes-agent")
	writeSkillCatalogHermesFixture(t, hermes)
	writeSkillCatalogSourcePairs(t, root)
	writeSkillCatalogProgress(t, root)

	result, err := WriteHermesContractInventory(HermesContractInventoryOptions{
		Root:             root,
		CurrentHermesSHA: "abc123",
	})
	if err != nil {
		t.Fatalf("WriteHermesContractInventory: %v", err)
	}

	families := result.Report.SkillCatalog
	if got, want := skillCatalogFamilyIDs(families), []string{
		"bundled_catalog_metadata",
		"category_descriptions",
		"optional_catalog_metadata",
		"prerequisites_readiness_metadata",
		"python_script_examples",
		"support_assets",
		"sync_reset_boundaries",
		"triggers_tags_related_skills",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("skill catalog family ids = %v, want %v", got, want)
	}

	bundled := skillCatalogFamilyByID(t, families, "bundled_catalog_metadata")
	if bundled.Status != fidelity.StatusCovered || bundled.Count == 0 {
		t.Fatalf("bundled catalog family = %+v, want covered with examples", bundled)
	}
	if !containsSkillCatalogString(bundled.Examples, "skills/creative/popular-web-designs/SKILL.md") {
		t.Fatalf("bundled examples = %v, want popular-web-designs SKILL.md", bundled.Examples)
	}
	if !containsSkillCatalogProgressRow(bundled.ProgressRows, "Portable SKILL.md format") {
		t.Fatalf("bundled progress rows = %+v, want Portable SKILL.md format", bundled.ProgressRows)
	}
	if !containsSkillCatalogSourcePair(bundled.SourcePairs, "skills/creative/popular-web-designs/SKILL.md") {
		t.Fatalf("bundled source pairs = %+v, want popular-web-designs SKILL.md", bundled.SourcePairs)
	}

	triggers := skillCatalogFamilyByID(t, families, "triggers_tags_related_skills")
	for _, want := range []string{"skills/yuanbao/SKILL.md", "optional-skills/devops/pinggy-tunnel/SKILL.md"} {
		if !containsSkillCatalogString(triggers.Examples, want) {
			t.Fatalf("triggers/tags examples = %v, want %s", triggers.Examples, want)
		}
	}

	support := skillCatalogFamilyByID(t, families, "support_assets")
	if support.Status != fidelity.StatusPlanned {
		t.Fatalf("support assets status = %q, want planned", support.Status)
	}
	for _, want := range []string{
		"optional-skills/creative/concept-diagrams/references/dashboard-patterns.md",
		"optional-skills/creative/concept-diagrams/templates/template.html",
	} {
		if !containsSkillCatalogString(support.Examples, want) {
			t.Fatalf("support asset examples = %v, want %s", support.Examples, want)
		}
	}

	sync := skillCatalogFamilyByID(t, families, "sync_reset_boundaries")
	if sync.Status != fidelity.StatusOwnedDivergence {
		t.Fatalf("sync/reset status = %q, want owned_divergence", sync.Status)
	}
	if !containsSkillCatalogString(sync.Examples, "skills/index-cache/openai_skills_skills_.json") {
		t.Fatalf("sync/reset examples = %v, want index-cache JSON", sync.Examples)
	}

	scripts := skillCatalogFamilyByID(t, families, "python_script_examples")
	if scripts.Status != fidelity.StatusExcluded {
		t.Fatalf("python/script status = %q, want excluded", scripts.Status)
	}
	if !containsSkillCatalogString(scripts.Examples, "optional-skills/research/darwinian-evolver/scripts/evolve.py") {
		t.Fatalf("python/script examples = %v, want evolve.py", scripts.Examples)
	}
	if containsSkillCatalogSourcePair(scripts.SourcePairs, "tools/transcription_tools.py") {
		t.Fatalf("python/script source pairs = %+v, want unrelated tool script source pair ignored", scripts.SourcePairs)
	}
	if containsSkillCatalogProgressRow(scripts.ProgressRows, "Voice transcription helper scripts") {
		t.Fatalf("python/script progress rows = %+v, want unrelated non-skill progress row ignored", scripts.ProgressRows)
	}

	md, err := os.ReadFile(result.MarkdownPath)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}
	for _, want := range []string{
		"## Skill Catalog Classification",
		"| `bundled_catalog_metadata` | `covered` |",
		"`skills/creative/popular-web-designs/SKILL.md`",
		"| `support_assets` | `planned` |",
		"| `sync_reset_boundaries` | `owned_divergence` |",
		"| `python_script_examples` | `excluded` |",
	} {
		if !strings.Contains(string(md), want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func writeSkillCatalogHermesFixture(t *testing.T, hermes string) {
	t.Helper()
	files := map[string]string{
		"hermes_cli/main.py":                                                         "# fixture\n",
		"skills/creative/popular-web-designs/SKILL.md":                               "---\nname: popular-web-designs\ntriggers:\n  - landing page\nmetadata:\n  hermes:\n    tags: [creative, web]\n---\nBody\n",
		"skills/productivity/notion/SKILL.md":                                        "---\nname: notion\nprerequisites:\n  env_vars:\n    - NOTION_API_KEY\n---\nBody\n",
		"skills/yuanbao/SKILL.md":                                                    "---\nname: yuanbao\nmetadata:\n  hermes:\n    tags: [chat, china]\n---\nBody\n",
		"skills/creative/DESCRIPTION.md":                                             "# Creative skills\n",
		"skills/index-cache/openai_skills_skills_.json":                              "{\"source\":\"openai\"}\n",
		"optional-skills/DESCRIPTION.md":                                             "# Optional skills\n",
		"optional-skills/research/osint-investigation/SKILL.md":                      "---\nname: osint-investigation\nmetadata:\n  hermes:\n    category: research\n---\nBody\n",
		"optional-skills/devops/pinggy-tunnel/SKILL.md":                              "---\nname: pinggy-tunnel\nrelated_skills:\n  - fastmcp\n---\nBody\n",
		"optional-skills/creative/concept-diagrams/references/dashboard-patterns.md": "# dashboard patterns\n",
		"optional-skills/creative/concept-diagrams/templates/template.html":          "<html></html>\n",
		"optional-skills/research/darwinian-evolver/scripts/evolve.py":               "print('fixture')\n",
		"RELEASE_v0.14.0.md":                                                         "# release\n",
	}
	for rel, body := range files {
		path := filepath.Join(hermes, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

func writeSkillCatalogSourcePairs(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "hermes-source-pairs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir source-pairs: %v", err)
	}
	body := `{
  "schema_version": "1.0",
  "hermes_sha": "abc123",
  "pairs": [
    {
      "hermes_file": "skills/creative/popular-web-designs/SKILL.md",
      "gormes_targets": ["internal/skills/parser.go"],
      "status": "covered",
      "contract": "Portable bundled SKILL.md metadata parsing and prompt exposure.",
      "tests": ["go test ./internal/skills -run TestParseSkill -count=1"],
      "progress_rows": ["Portable SKILL.md format"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "skills/index-cache/openai_skills_skills_.json",
      "gormes_targets": ["internal/skills/update_sync.go"],
      "status": "owned",
      "contract": "Gormes owns source lockfile and reset/sync boundaries while reporting Hermes index-cache evidence.",
      "tests": ["go test ./internal/skills -run TestSyncBundledSkillsFromManifestUsesDigestThreeWayAndConflictCopies -count=1"],
      "progress_rows": ["Skill hub source lockfile"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "optional-skills/research/darwinian-evolver/scripts/evolve.py",
      "status": "excluded",
      "contract": "Report Python-only helper scripts as explicit non-runtime evidence.",
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "tools/transcription_tools.py",
      "gormes_targets": ["internal/tools/transcription.go"],
      "status": "covered",
      "contract": "Python helper script execution for voice transcription tools.",
      "tests": ["go test ./internal/tools -run TestTranscriptionTool -count=1"],
      "progress_rows": ["Voice transcription helper scripts"],
      "last_checked_hermes_sha": "abc123"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write source-pairs: %v", err)
	}
}

func writeSkillCatalogProgress(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir progress: %v", err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "test"},
		Phases: map[string]progress.Phase{
			"1": {
				Name:        "Skill catalog fixture",
				Deliverable: "Fixture",
				Subphases: map[string]progress.Subphase{
					"1.A": {
						Name: "Skill catalog",
						Items: []progress.Item{
							skillCatalogProgressItem("Portable SKILL.md format", progress.StatusComplete, progress.ContractStatusValidated, "skills", "hermes-agent/skills/creative/popular-web-designs/SKILL.md", "go test ./internal/skills -run TestParseSkill -count=1"),
							skillCatalogProgressItem("SKILL.md metadata parser fixtures", progress.StatusComplete, progress.ContractStatusValidated, "skills", "hermes-agent/skills/productivity/notion/SKILL.md", "go test ./internal/skills -run TestOptionalSkillCatalogHermesV014Metadata -count=1"),
							skillCatalogProgressItem("Optional skill catalog metadata", progress.StatusComplete, progress.ContractStatusValidated, "skills", "hermes-agent/optional-skills/research/osint-investigation/SKILL.md", "go test ./internal/skills -run TestOptionalSkillCatalogHermesV014Metadata -count=1"),
							skillCatalogProgressItem("Skill hub source lockfile", progress.StatusComplete, progress.ContractStatusValidated, "skills", "hermes-agent/skills/index-cache/openai_skills_skills_.json", "go test ./internal/skills -run TestSyncBundledSkillsFromManifestUsesDigestThreeWayAndConflictCopies -count=1"),
							skillCatalogProgressItem("Skill category description catalog", progress.StatusPlanned, progress.ContractStatusFixtureReady, "skills", "hermes-agent/skills/creative/DESCRIPTION.md", "go test ./internal/skills -run TestListInstalledSkills -count=1"),
							skillCatalogProgressItem("Skill support asset inventory", progress.StatusPlanned, progress.ContractStatusFixtureReady, "skills", "hermes-agent/optional-skills/creative/concept-diagrams/references/dashboard-patterns.md", "go test ./internal/skills -run TestPreprocessSkillContent -count=1"),
							skillCatalogProgressItem("Voice transcription helper scripts", progress.StatusComplete, progress.ContractStatusValidated, "tools", "hermes-agent/tools/transcription_tools.py", "go test ./internal/tools -run TestTranscriptionTool -count=1"),
						},
					},
				},
			},
		},
	}
	if err := progress.SaveProgress(path, p); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
}

func skillCatalogProgressItem(name string, status progress.Status, contractStatus progress.ContractStatus, module, sourceRef, testCommand string) progress.Item {
	return progress.Item{
		Name:           name,
		Priority:       "P1",
		Status:         status,
		ContractStatus: contractStatus,
		Module:         module,
		Contract:       name + " contract.",
		SourceRefs:     []string{sourceRef},
		TestCommands:   []string{testCommand},
	}
}

func skillCatalogFamilyIDs(families []fidelity.CatalogFamilyReport) []string {
	ids := make([]string, 0, len(families))
	for _, family := range families {
		ids = append(ids, family.ID)
	}
	return ids
}

func skillCatalogFamilyByID(t *testing.T, families []fidelity.CatalogFamilyReport, id string) fidelity.CatalogFamilyReport {
	t.Helper()
	for _, family := range families {
		if family.ID == id {
			return family
		}
	}
	t.Fatalf("skill catalog family %q missing from %+v", id, families)
	return fidelity.CatalogFamilyReport{}
}

func containsSkillCatalogString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsSkillCatalogProgressRow(rows []fidelity.ProgressRowEvidence, want string) bool {
	for _, row := range rows {
		if row.Name == want {
			return true
		}
	}
	return false
}

func containsSkillCatalogSourcePair(pairs []fidelity.SourcePairEvidence, want string) bool {
	for _, pair := range pairs {
		if pair.HermesFile == want {
			return true
		}
	}
	return false
}
