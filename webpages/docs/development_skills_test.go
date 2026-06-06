package docs_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDevelopmentGoalSkillContractIsRoutable(t *testing.T) {
	goal := readRepoText(t, "development-skills/gormes-goal/SKILL.md")
	for _, want := range []string{
		"DEV_GOAL_REPORT:",
		"DEV_GOAL_VALIDATED:",
		"DEV_GOAL_DECISION:",
		"no prose after them",
		"marker recovery",
		"DEV_GOAL_VALIDATED: no",
		"DEV_GOAL_DECISION: blocked",
	} {
		if !strings.Contains(goal, want) {
			t.Fatalf("gormes-goal missing development-goal contract text %q", want)
		}
	}

	report := strings.Index(goal, "DEV_GOAL_REPORT:")
	validated := strings.Index(goal, "DEV_GOAL_VALIDATED:")
	decision := strings.Index(goal, "DEV_GOAL_DECISION:")
	if !(report >= 0 && report < validated && validated < decision) {
		t.Fatalf("development-goal marker order mismatch: report=%d validated=%d decision=%d", report, validated, decision)
	}

	manager := readRepoText(t, "development-skills/gormes-skill-manager/SKILL.md")
	for _, want := range []string{
		"development-goal iteration",
		"DEV_GOAL markers",
		"development-goal runner prompt",
		"`gormes-goal`",
	} {
		if !strings.Contains(manager, want) {
			t.Fatalf("gormes-skill-manager missing development-goal routing text %q", want)
		}
	}
}

func TestSkillManagerRoutesSkillMaintenanceThroughWriteASkill(t *testing.T) {
	manager := readRepoText(t, "development-skills/gormes-skill-manager/SKILL.md")
	for _, want := range []string{
		"Create or improve skills",
		"`write-a-skill`",
		"`skill-creator`",
		"process guidance rather than a session diary",
	} {
		if !strings.Contains(manager, want) {
			t.Fatalf("gormes-skill-manager missing skill-maintenance routing text %q", want)
		}
	}
}

func TestDevelopmentSkillLoaderViewsResolveToCanonicalSource(t *testing.T) {
	repoRoot := findRepoRoot(t)
	canonicalRoot := filepath.Join(repoRoot, "development-skills")
	docsAlias := filepath.Join(repoRoot, "webpages", "docs", "development-skills")
	if _, err := os.Lstat(docsAlias); err == nil {
		t.Fatalf("docs-site development-skills alias must not exist: %s", docsAlias)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat docs-site development-skills alias: %v", err)
	}
	skills := canonicalDevelopmentSkills(t, canonicalRoot)
	if len(skills) == 0 {
		t.Fatalf("no canonical development skills found under %s", canonicalRoot)
	}

	for _, loaderRoot := range []string{".agents/skills", ".claude/skills", ".codex/skills"} {
		loaderRoot := loaderRoot
		t.Run(loaderRoot, func(t *testing.T) {
			for _, skill := range skills {
				loaderPath := filepath.Join(repoRoot, loaderRoot, skill)
				info, err := os.Lstat(loaderPath)
				if err != nil {
					t.Fatalf("loader view missing %s for skill %s: %v", loaderRoot, skill, err)
				}
				if info.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("loader view %s for skill %s is not a symlink", loaderRoot, skill)
				}
				got, err := filepath.EvalSymlinks(loaderPath)
				if err != nil {
					t.Fatalf("resolve loader view %s for skill %s: %v", loaderRoot, skill, err)
				}
				want, err := filepath.EvalSymlinks(filepath.Join(canonicalRoot, skill))
				if err != nil {
					t.Fatalf("resolve canonical skill %s: %v", skill, err)
				}
				if got != want {
					t.Fatalf("loader view %s for skill %s resolves to %s, want %s", loaderRoot, skill, got, want)
				}
			}
		})
	}
}

func canonicalDevelopmentSkills(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read canonical development skills root: %v", err)
	}
	var skills []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), "SKILL.md")); err == nil {
			skills = append(skills, entry.Name())
		}
	}
	sort.Strings(skills)
	return skills
}
