package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListInstalledSkills_StatusColumnPopulated(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", root)
	writeListSkillDoc(t, root, "x", "hub-skill", "hub", "community")
	writeListSkillDoc(t, root, "x", "builtin-skill", "builtin", "builtin")
	writeListSkillDoc(t, root, "x", "local-skill", "local", "local")

	rows := ListInstalledSkills(ListOptions{}, nil)

	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3: %#v", len(rows), rows)
	}
	for _, row := range rows {
		if row.Status != "enabled" {
			t.Fatalf("row %q Status = %q, want enabled", row.Name, row.Status)
		}
	}
}

func TestListInstalledSkills_DisabledRowsCarryDisabledStatus(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", root)
	writeListSkillDoc(t, root, "x", "hub-skill", "hub", "community")
	writeListSkillDoc(t, root, "x", "local-skill", "local", "local")

	rows := ListInstalledSkills(ListOptions{}, map[string]struct{}{"hub-skill": {}})

	got := rowStatuses(rows)
	want := map[string]string{
		"hub-skill":   "disabled",
		"local-skill": "enabled",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("statuses = %#v, want %#v", got, want)
	}
}

func TestListInstalledSkills_EnabledOnlyFilter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", root)
	writeListSkillDoc(t, root, "x", "hub-skill", "hub", "community")
	writeListSkillDoc(t, root, "x", "builtin-skill", "builtin", "builtin")
	writeListSkillDoc(t, root, "x", "local-skill", "local", "local")

	rows := ListInstalledSkills(ListOptions{EnabledOnly: true}, map[string]struct{}{"hub-skill": {}})

	got := rowNames(rows)
	want := []string{"builtin-skill", "local-skill"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %#v, want %#v", got, want)
	}
}

func TestListInstalledSkills_SourceFilterRespected(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", root)
	writeListSkillDoc(t, root, "x", "hub-skill", "hub", "community")
	writeListSkillDoc(t, root, "x", "builtin-skill", "builtin", "builtin")
	writeListSkillDoc(t, root, "x", "local-skill", "local", "local")

	tests := map[string][]string{
		"hub":     {"hub-skill"},
		"builtin": {"builtin-skill"},
		"local":   {"local-skill"},
	}
	for source, want := range tests {
		t.Run(source, func(t *testing.T) {
			rows := ListInstalledSkills(ListOptions{Source: source}, nil)
			if got := rowNames(rows); !reflect.DeepEqual(got, want) {
				t.Fatalf("names = %#v, want %#v", got, want)
			}
		})
	}
}

func TestListInstalledSkillsExternalDirs(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	writeListSkillDoc(t, root, "ops", "shared", "local", "local")
	writeExternalListSkillDoc(t, external, "research", "external-only")
	writeExternalListSkillDoc(t, external, "research", "shared")

	rows := ListInstalledSkillsFromRoots(root, "", ListOptions{ExternalRoots: []string{external}}, nil)
	got := rowNames(rows)
	want := []string{"shared", "external-only"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %#v, want %#v; rows=%#v", got, want, rows)
	}
	local, ok := findListRow(rows, "shared")
	if !ok || local.Source != "local" || local.Trust != "local" {
		t.Fatalf("shared row = %#v, want local precedence", local)
	}
	externalRow, ok := findListRow(rows, "external-only")
	if !ok {
		t.Fatalf("external-only missing from rows: %#v", rows)
	}
	if externalRow.Category != "research" || externalRow.Source != "external" || externalRow.Trust != "operator" || externalRow.Status != SkillStatusEnabled {
		t.Fatalf("external row metadata = %#v", externalRow)
	}

	externalRows := ListInstalledSkillsFromRoots(root, "", ListOptions{Source: "external", ExternalRoots: []string{external}}, map[string]struct{}{"external-only": {}})
	if got := rowNames(externalRows); !reflect.DeepEqual(got, []string{"external-only"}) {
		t.Fatalf("external source names = %#v, want external-only", got)
	}
	if externalRows[0].Status != SkillStatusDisabled {
		t.Fatalf("disabled external status = %q, want disabled", externalRows[0].Status)
	}
}

func TestListInstalledSkills_BundledRootSymlinkTracksHermesSkills(t *testing.T) {
	activeRoot := t.TempDir()
	bundledReal := t.TempDir()
	writeBundledListSkillDoc(t, bundledReal, "productivity", "hermes-skill")

	bundledLink := filepath.Join(t.TempDir(), "skills")
	if err := os.Symlink(bundledReal, bundledLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("GORMES_SKILLS_ROOT", activeRoot)
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", bundledLink)

	rows := ListInstalledSkills(ListOptions{Source: "builtin"}, nil)
	row, ok := findListRow(rows, "hermes-skill")
	if !ok {
		t.Fatalf("bundled symlink skill missing from rows: %#v", rows)
	}
	if row.Category != "productivity" || row.Source != "builtin" || row.Trust != "system" {
		t.Fatalf("row metadata = category=%q source=%q trust=%q", row.Category, row.Source, row.Trust)
	}
	if !strings.HasPrefix(filepath.ToSlash(row.Path), filepath.ToSlash(bundledLink)+"/") {
		t.Fatalf("row.Path = %q, want logical symlink root %q", row.Path, bundledLink)
	}
}

func writeBundledListSkillDoc(t *testing.T, root, category, name string) {
	t.Helper()
	writeListSkillDocAt(t, filepath.Join(root, category, name, "SKILL.md"), name)
}

func writeExternalListSkillDoc(t *testing.T, root, category, name string) {
	t.Helper()
	writeListSkillDocAt(t, filepath.Join(root, category, name, "SKILL.md"), name)
}

func writeListSkillDoc(t *testing.T, root, category, name, source, trust string) {
	t.Helper()
	dir := filepath.Join(root, "active", category, name)
	writeListSkillDocAt(t, filepath.Join(dir, "SKILL.md"), name)
	meta := `{"source":"` + source + `","trust":"` + trust + `"}`
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatalf("WriteFile(meta.json): %v", err)
	}
}

func writeListSkillDocAt(t *testing.T, path, name string) {
	t.Helper()
	writeSkillDoc(t, path, name, name+" description", "Use "+name+".")
}

func findListRow(rows []SkillRow, name string) (SkillRow, bool) {
	for _, row := range rows {
		if row.Name == name {
			return row, true
		}
	}
	return SkillRow{}, false
}

func rowNames(rows []SkillRow) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}
	return out
}

func rowStatuses(rows []SkillRow) map[string]string {
	out := make(map[string]string)
	for _, row := range rows {
		out[row.Name] = string(row.Status)
	}
	return out
}
