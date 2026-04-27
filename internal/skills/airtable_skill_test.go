package skills_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

func TestAirtableSkillBundledDocumentParsesReviewedMetadataAndCookbook(t *testing.T) {
	raw := readBundledAirtableSkill(t)

	skill, err := skills.Parse(raw, 64*1024)
	if err != nil {
		t.Fatalf("Parse(Airtable SKILL.md) error = %v", err)
	}
	if skill.Name != "airtable" {
		t.Fatalf("Name = %q, want airtable", skill.Name)
	}
	if skill.Description != "Airtable REST API via curl. Records CRUD, filters, upserts." {
		t.Fatalf("Description = %q", skill.Description)
	}

	if got := stringSliceField(t, skill, "Triggers"); !containsString(got, "Airtable base/table/record work") {
		t.Fatalf("Triggers = %#v, want Airtable trigger", got)
	}
	if got := stringSliceField(t, skill, "Exclusions"); !containsString(got, "Live OAuth, sync daemons, or workspace-wide base discovery") {
		t.Fatalf("Exclusions = %#v, want prompt-safe exclusion", got)
	}
	if got := stringField(t, skill, "ReviewState"); got != "reviewed" {
		t.Fatalf("ReviewState = %q, want reviewed", got)
	}
	if got := credentialGroupsString(t, skill); !strings.Contains(got, "AIRTABLE_API_KEY") || !strings.Contains(got, "AIRTABLE_PAT") {
		t.Fatalf("CredentialGroups = %s, want AIRTABLE_API_KEY/AIRTABLE_PAT any-of group", got)
	}

	for _, want := range []string{
		"# Airtable - Bases, Tables & Records",
		"## Cookbook",
		"### List bases the token can see",
		"### Upsert by a merge field",
		"## Safety Boundaries",
	} {
		if !strings.Contains(skill.Body, want) {
			t.Fatalf("Airtable body missing %q", want)
		}
	}
}

func TestAirtableSkillMissingCredentialsVisibleAsUnavailableCatalogRowAndExcludedPrompt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", root)
	t.Setenv("AIRTABLE_API_KEY", "")
	t.Setenv("AIRTABLE_PAT", "")
	installAirtableSkillFixture(t, root)

	rows := skills.ListInstalledSkills(skills.ListOptions{Source: "builtin"}, nil)
	row, ok := findSkillRow(rows, "airtable")
	if !ok {
		t.Fatalf("Airtable catalog row missing from rows: %#v", rows)
	}
	if row.Status != skills.SkillStatusMissingPrerequisite {
		t.Fatalf("Airtable catalog Status = %q, want %q", row.Status, skills.SkillStatusMissingPrerequisite)
	}

	runtime := skills.NewRuntime(root, 64*1024, 3, "")
	block, names, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "airtable records upsert", skills.RuntimeOptions{
		Env: map[string]string{},
	})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() error = %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("names = %#v, want no prompt-eligible Airtable skill", names)
	}
	if strings.Contains(block, "Airtable") || strings.Contains(block, "curl -s") {
		t.Fatalf("block injected unavailable Airtable instructions:\n%s", block)
	}

	status := findSkillStatus(t, statuses, "airtable")
	if status.Status != skills.SkillStatusMissingPrerequisite {
		t.Fatalf("status = %q, want %q", status.Status, skills.SkillStatusMissingPrerequisite)
	}
	if !strings.Contains(status.Reason, "AIRTABLE_API_KEY or AIRTABLE_PAT") {
		t.Fatalf("missing credential reason = %q, want redacted AIRTABLE_API_KEY/AIRTABLE_PAT evidence", status.Reason)
	}
	if strings.Contains(status.Reason, "pat_secret") || strings.Contains(status.Reason, "key_secret") {
		t.Fatalf("credential reason leaked a secret: %q", status.Reason)
	}
}

func TestAirtableSkillBundledCatalogRowVisibleWhenCredentialsMissing(t *testing.T) {
	activeRoot := t.TempDir()
	bundledRoot, err := filepath.Abs(filepath.Join("..", "..", "skills"))
	if err != nil {
		t.Fatalf("Abs(bundled skills): %v", err)
	}
	t.Setenv("GORMES_SKILLS_ROOT", activeRoot)
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", bundledRoot)
	t.Setenv("AIRTABLE_API_KEY", "")
	t.Setenv("AIRTABLE_PAT", "")

	rows := skills.ListInstalledSkills(skills.ListOptions{Source: "builtin"}, nil)
	row, ok := findSkillRow(rows, "airtable")
	if !ok {
		t.Fatalf("bundled Airtable catalog row missing from rows: %#v", rows)
	}
	if row.Category != "productivity" || row.Source != "builtin" || row.Trust != "system" {
		t.Fatalf("bundled row metadata = category=%q source=%q trust=%q", row.Category, row.Source, row.Trust)
	}
	if row.Status != skills.SkillStatusMissingPrerequisite {
		t.Fatalf("bundled Airtable Status = %q, want %q", row.Status, skills.SkillStatusMissingPrerequisite)
	}
	if !strings.HasSuffix(filepath.ToSlash(row.Path), "skills/productivity/airtable/SKILL.md") {
		t.Fatalf("bundled Airtable Path = %q, want shipped SKILL.md path", row.Path)
	}
}

func TestAirtableSkillDotenvCredentialEnablesPromptWithoutSecretLeak(t *testing.T) {
	root := t.TempDir()
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("HERMES_HOME", "")
	clearEnvVars(t, "AIRTABLE_API_KEY", "AIRTABLE_PAT")
	installAirtableSkillFixture(t, root)

	secret := "pat_secret_from_dotenv"
	cfgDir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", cfgDir, err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ".env"), []byte("AIRTABLE_PAT="+secret+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.env): %v", err)
	}

	evidence := config.CheckOptionalEnvAny("AIRTABLE_API_KEY", "AIRTABLE_PAT")
	if !evidence.Available {
		t.Fatalf("optional env evidence = %+v, want available", evidence)
	}
	if evidence.PresentName != "AIRTABLE_PAT" {
		t.Fatalf("PresentName = %q, want AIRTABLE_PAT", evidence.PresentName)
	}
	if !strings.Contains(evidence.Evidence, "AIRTABLE_PAT=[redacted]") {
		t.Fatalf("Evidence = %q, want redacted present credential", evidence.Evidence)
	}
	if strings.Contains(evidence.Evidence, secret) {
		t.Fatalf("Evidence leaked secret: %q", evidence.Evidence)
	}

	runtime := skills.NewRuntime(root, 64*1024, 3, "")
	block, names, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "airtable records upsert", skills.RuntimeOptions{})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() error = %v", err)
	}
	if !reflect.DeepEqual(names, []string{"airtable"}) {
		t.Fatalf("names = %#v, want airtable", names)
	}
	status := findSkillStatus(t, statuses, "airtable")
	if status.Status != skills.SkillStatusAvailable {
		t.Fatalf("status = %q, want %q", status.Status, skills.SkillStatusAvailable)
	}
	if !strings.Contains(block, "## airtable") || !strings.Contains(block, "### Upsert by a merge field") {
		t.Fatalf("block did not include Airtable cookbook:\n%s", block)
	}
	if strings.Contains(block, secret) {
		t.Fatalf("block leaked Airtable credential:\n%s", block)
	}
}

func TestAirtableSkillDisabledExcludesPromptEvenWhenCredentialsPresent(t *testing.T) {
	root := t.TempDir()
	installAirtableSkillFixture(t, root)

	runtime := skills.NewRuntime(root, 64*1024, 3, "")
	block, names, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "airtable records", skills.RuntimeOptions{
		DisabledSkillNames: map[string]bool{"airtable": true},
		Env:                map[string]string{"AIRTABLE_PAT": "pat_secret"},
	})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() error = %v", err)
	}
	if len(names) != 0 || block != "" {
		t.Fatalf("disabled Airtable prompt rendered names=%#v block=%q", names, block)
	}
	status := findSkillStatus(t, statuses, "airtable")
	if status.Status != skills.SkillStatusDisabled {
		t.Fatalf("status = %q, want %q", status.Status, skills.SkillStatusDisabled)
	}
}

func readBundledAirtableSkill(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "skills", "productivity", "airtable", "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return raw
}

func installAirtableSkillFixture(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "active", "productivity", "airtable")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), readBundledAirtableSkill(t), 0o644); err != nil {
		t.Fatalf("WriteFile(Airtable SKILL.md): %v", err)
	}
	meta := `{"category":"productivity","source":"builtin","trust":"system"}`
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatalf("WriteFile(meta.json): %v", err)
	}
}

func stringSliceField(t *testing.T, value any, fieldName string) []string {
	t.Helper()
	field := reflect.ValueOf(value).FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("Skill.%s field missing", fieldName)
	}
	if field.Kind() != reflect.Slice {
		t.Fatalf("Skill.%s kind = %s, want slice", fieldName, field.Kind())
	}
	out := make([]string, 0, field.Len())
	for i := 0; i < field.Len(); i++ {
		out = append(out, field.Index(i).String())
	}
	return out
}

func stringField(t *testing.T, value any, fieldName string) string {
	t.Helper()
	field := reflect.ValueOf(value).FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("Skill.%s field missing", fieldName)
	}
	if field.Kind() != reflect.String {
		t.Fatalf("Skill.%s kind = %s, want string", fieldName, field.Kind())
	}
	return field.String()
}

func credentialGroupsString(t *testing.T, value any) string {
	t.Helper()
	field := reflect.ValueOf(value).FieldByName("CredentialGroups")
	if !field.IsValid() {
		t.Fatalf("Skill.CredentialGroups field missing")
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf("%#v", field.Interface()), "\n", " "), "\t", " "))
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func findSkillRow(rows []skills.SkillRow, name string) (skills.SkillRow, bool) {
	for _, row := range rows {
		if row.Name == name {
			return row, true
		}
	}
	return skills.SkillRow{}, false
}

func findSkillStatus(t *testing.T, statuses []skills.SkillStatus, name string) skills.SkillStatus {
	t.Helper()
	for _, status := range statuses {
		if status.Name == name {
			return status
		}
	}
	t.Fatalf("status for %q missing from %#v", name, statuses)
	return skills.SkillStatus{}
}

func clearEnvVars(t *testing.T, names ...string) {
	t.Helper()
	type oldEnv struct {
		name  string
		value string
		ok    bool
	}
	old := make([]oldEnv, 0, len(names))
	for _, name := range names {
		value, ok := os.LookupEnv(name)
		old = append(old, oldEnv{name: name, value: value, ok: ok})
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("Unsetenv(%q): %v", name, err)
		}
	}
	t.Cleanup(func() {
		for _, entry := range old {
			if entry.ok {
				_ = os.Setenv(entry.name, entry.value)
			} else {
				_ = os.Unsetenv(entry.name)
			}
		}
	})
}
