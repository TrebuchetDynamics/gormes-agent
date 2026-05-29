package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/spf13/cobra"
)

// TestSkillsListCommand_JSONEmitsStructuredInventory proves
// `gormes skills list --json` emits a parseable
// `{build, source, enabled_only, counts: {enabled, disabled}, skills:
// [{name, category, source, trust, status, path}]}` document so fleet
// automation inventorying installed skills across machines can ingest
// the list with binary attribution. Build provenance leads — same
// convention as the rest of the `--json` arc. The default
// human-readable text output remains unchanged.
func TestSkillsListCommand_JSONEmitsStructuredInventory(t *testing.T) {
	rows := []skills.SkillRow{
		{Name: "hub-skill", Category: "x", Source: "hub", Trust: "community", Status: "disabled", Path: "/skills/hub/hub-skill"},
		{Name: "builtin-skill", Category: "y", Source: "builtin", Trust: "builtin", Status: "enabled", Path: "/skills/builtin/builtin-skill"},
		{Name: "local-skill", Category: "z", Source: "local", Trust: "local", Status: "enabled", Path: "/skills/local/local-skill"},
	}
	cmd := NewSkillsCommand(SkillsCommandDeps{
		ListInstalledSkills: func(opts skills.ListOptions, disabled map[string]struct{}) []skills.SkillRow {
			return rows
		},
		DisabledSkills: func(string) map[string]struct{} {
			return map[string]struct{}{"hub-skill": {}}
		},
		BuildProvenance: func() any {
			return map[string]string{"version": "test-version-1.0"}
		},
	})

	stdout, err := executeSkillsCommand(cmd, "list", "--json")
	if err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	var got struct {
		Build       map[string]string `json:"build"`
		Source      string            `json:"source"`
		EnabledOnly bool              `json:"enabled_only"`
		Counts      struct {
			Enabled  int `json:"enabled"`
			Disabled int `json:"disabled"`
		} `json:"counts"`
		Skills []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Source   string `json:"source"`
			Trust    string `json:"trust"`
			Status   string `json:"status"`
			Path     string `json:"path"`
		} `json:"skills"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build["version"] != "test-version-1.0" {
		t.Errorf("build.version = %q, want test-version-1.0", got.Build["version"])
	}
	if got.Source != "all" {
		t.Errorf("source = %q, want all", got.Source)
	}
	if got.Counts.Enabled != 2 || got.Counts.Disabled != 1 {
		t.Errorf("counts = %+v, want enabled=2 disabled=1", got.Counts)
	}
	if len(got.Skills) != 3 {
		t.Fatalf("skills len = %d, want 3", len(got.Skills))
	}
	if got.Skills[0].Name != "hub-skill" || got.Skills[0].Status != "disabled" {
		t.Errorf("skills[0] = %+v, want hub-skill disabled", got.Skills[0])
	}
}

func TestSkillsListCommand_RendersStatusColumnAndSummary(t *testing.T) {
	rows := []skills.SkillRow{
		{Name: "hub-skill", Category: "x", Source: "hub", Trust: "community", Status: "disabled"},
		{Name: "builtin-skill", Category: "x", Source: "builtin", Trust: "builtin", Status: "enabled"},
		{Name: "local-skill", Category: "x", Source: "local", Trust: "local", Status: "enabled"},
	}
	cmd := NewSkillsCommand(SkillsCommandDeps{
		ListInstalledSkills: func(opts skills.ListOptions, disabled map[string]struct{}) []skills.SkillRow {
			if opts.Source != "all" || opts.EnabledOnly {
				t.Fatalf("opts = %+v, want source=all enabledOnly=false", opts)
			}
			if _, ok := disabled["hub-skill"]; !ok {
				t.Fatalf("disabled set = %#v, want hub-skill", disabled)
			}
			return rows
		},
		DisabledSkills: func(platform string) map[string]struct{} {
			if platform != "" {
				t.Fatalf("platform = %q, want empty", platform)
			}
			return map[string]struct{}{"hub-skill": {}}
		},
	})

	stdout, err := executeSkillsCommand(cmd, "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, want := range []string{"Name", "Category", "Source", "Trust", "Status", "hub-skill", "disabled", "2 enabled, 1 disabled"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSkillsListCommand_RendersEnabledOnlySummary(t *testing.T) {
	var gotOpts skills.ListOptions
	cmd := NewSkillsCommand(SkillsCommandDeps{
		ListInstalledSkills: func(opts skills.ListOptions, disabled map[string]struct{}) []skills.SkillRow {
			gotOpts = opts
			return []skills.SkillRow{
				{Name: "builtin-skill", Category: "x", Source: "builtin", Trust: "builtin", Status: "enabled"},
				{Name: "local-skill", Category: "x", Source: "local", Trust: "local", Status: "enabled"},
			}
		},
		DisabledSkills: func(platform string) map[string]struct{} {
			return map[string]struct{}{"hub-skill": {}}
		},
	})

	stdout, err := executeSkillsCommand(cmd, "list", "--source", "builtin", "--enabled-only")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	wantOpts := skills.ListOptions{Source: "builtin", EnabledOnly: true}
	if !reflect.DeepEqual(gotOpts, wantOpts) {
		t.Fatalf("opts = %+v, want %+v", gotOpts, wantOpts)
	}
	if strings.Contains(stdout, "disabled") {
		t.Fatalf("enabled-only stdout mentioned disabled rows:\n%s", stdout)
	}
	if !strings.Contains(stdout, "2 enabled shown") {
		t.Fatalf("stdout missing enabled-only summary:\n%s", stdout)
	}
}

func TestSkillsListCommand_AllowsExternalSourceFilter(t *testing.T) {
	var gotOpts skills.ListOptions
	cmd := NewSkillsCommand(SkillsCommandDeps{
		ListInstalledSkills: func(opts skills.ListOptions, disabled map[string]struct{}) []skills.SkillRow {
			gotOpts = opts
			return []skills.SkillRow{{Name: "external-skill", Category: "x", Source: "external", Trust: "operator", Status: "enabled"}}
		},
	})

	stdout, err := executeSkillsCommand(cmd, "list", "--source", "external")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if gotOpts.Source != "external" {
		t.Fatalf("opts.Source = %q, want external", gotOpts.Source)
	}
	for _, want := range []string{"external-skill", "external", "operator"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSkillsListCommand_PlatformArgNotPropagated(t *testing.T) {
	t.Setenv("HERMES_PLATFORM", "telegram")
	var seenPlatform string
	cmd := NewSkillsCommand(SkillsCommandDeps{
		ListInstalledSkills: func(opts skills.ListOptions, disabled map[string]struct{}) []skills.SkillRow {
			return nil
		},
		DisabledSkills: func(platform string) map[string]struct{} {
			seenPlatform = platform
			return nil
		},
	})

	if _, err := executeSkillsCommand(cmd, "list"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if seenPlatform != "" {
		t.Fatalf("disabled skills resolver saw platform %q, want empty", seenPlatform)
	}
}

func executeSkillsCommand(cmd *cobra.Command, args ...string) (string, error) {
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}
