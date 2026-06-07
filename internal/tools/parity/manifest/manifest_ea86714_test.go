package manifest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToolParityManifestRetainsEA86714ComputerUse(t *testing.T) {
	manifest, err := LoadUpstreamToolParityManifest()
	if err != nil {
		t.Fatalf("LoadUpstreamToolParityManifest: %v", err)
	}

	if got, want := manifest.Source.Commit, "524cbabd8"; !strings.HasPrefix(got, want) {
		t.Fatalf("source commit = %q, want prefix %q", got, want)
	}
	for _, input := range []string{
		"toolsets.py",
		"hermes_cli/tools_config.py",
		"tools/computer_use_tool.py",
		"tools/computer_use/schema.py",
		"tools/computer_use/tool.py",
		"tools/kanban_tools.py",
		"tests/tools/test_computer_use.py",
		"tests/tools/test_kanban_tools.py",
	} {
		assertContains(t, manifest.Source.InputFiles, input)
	}

	computerUse := mustTool(t, manifest, "computer_use")
	if got, want := computerUse.Toolset, "computer_use"; got != want {
		t.Fatalf("computer_use toolset = %q, want %q", got, want)
	}
	assertSchemaActionEnumContains(t, computerUse.Schema, []string{
		"capture",
		"click",
		"double_click",
		"right_click",
		"middle_click",
		"drag",
		"scroll",
		"type",
		"key",
		"set_value",
		"wait",
		"list_apps",
		"focus_app",
	})
	assertSchemaPropertyEnum(t, computerUse.Schema, "mode", []string{"som", "vision", "ax"})
	if !computerUse.HasProviderPath("cua-driver") {
		t.Fatalf("computer_use should capture the cua-driver provider path")
	}
	assertContains(t, computerUse.Dependencies, "macos")
	assertContains(t, computerUse.Dependencies, "cua-driver")
	assertContains(t, computerUse.DegradedStatus.Statuses, "unsupported_platform")
	assertContains(t, computerUse.DegradedStatus.Statuses, "cua_driver_missing")

	computerUseToolset := mustToolset(t, manifest, "computer_use")
	assertContains(t, computerUseToolset.DirectTools, "computer_use")
	assertContains(t, computerUseToolset.ResolvedTools, "computer_use")

	cli := mustToolset(t, manifest, "hermes-cli")
	assertContains(t, cli.ResolvedTools, "computer_use")
	assertNotContains(t, cli.ResolvedTools, "discord")
	assertNotContains(t, cli.ResolvedTools, "feishu_doc_read")

	gateway := mustToolset(t, manifest, "hermes-gateway")
	assertContains(t, gateway.ResolvedTools, "computer_use")
}

func TestToolParityManifestHermesEA86714KanbanDescriptors(t *testing.T) {
	manifest, err := LoadUpstreamToolParityManifest()
	if err != nil {
		t.Fatalf("LoadUpstreamToolParityManifest: %v", err)
	}

	kanbanToolset := mustToolset(t, manifest, "kanban")
	wantTools := []string{
		"kanban_show",
		"kanban_complete",
		"kanban_block",
		"kanban_heartbeat",
		"kanban_comment",
		"kanban_create",
		"kanban_link",
	}
	for _, name := range wantTools {
		row := mustTool(t, manifest, name)
		if got, want := row.Toolset, "kanban"; got != want {
			t.Fatalf("%s toolset = %q, want %q", name, got, want)
		}
		assertContains(t, row.RequiredEnv, "HERMES_KANBAN_TASK")
		assertContains(t, row.DegradedStatus.Statuses, "kanban_context_missing")
		assertContains(t, kanbanToolset.ResolvedTools, name)
	}

	cli := mustToolset(t, manifest, "hermes-cli")
	for _, name := range wantTools {
		assertContains(t, cli.ResolvedTools, name)
	}
}

func assertSchemaActionEnumContains(t *testing.T, raw json.RawMessage, want []string) {
	t.Helper()
	action := schemaProperty(t, raw, "action")
	got, ok := action["enum"].([]any)
	if !ok {
		t.Fatalf("action enum = %#v, want array", action["enum"])
	}
	seen := make(map[string]struct{}, len(got))
	for _, value := range got {
		name, ok := value.(string)
		if !ok {
			t.Fatalf("action enum contains non-string value %#v", value)
		}
		seen[name] = struct{}{}
	}
	for _, name := range want {
		if _, ok := seen[name]; !ok {
			t.Fatalf("action enum %v missing %q", got, name)
		}
	}
}

func mustToolset(t *testing.T, manifest UpstreamToolParityManifest, name string) UpstreamToolsetRow {
	t.Helper()
	row, ok := manifest.Toolset(name)
	if !ok {
		t.Fatalf("missing toolset parity row for %s", name)
	}
	return row
}

func assertSchemaPropertyEnum(t *testing.T, raw json.RawMessage, property string, want []string) {
	t.Helper()
	field := schemaProperty(t, raw, property)
	got, ok := field["enum"].([]any)
	if !ok {
		t.Fatalf("%s enum = %#v, want array", property, field["enum"])
	}
	if len(got) != len(want) {
		t.Fatalf("%s enum length = %d (%v), want %d (%v)", property, len(got), got, len(want), want)
	}
	for i, value := range got {
		if value != want[i] {
			t.Fatalf("%s enum[%d] = %v, want %q; full enum=%v", property, i, value, want[i], got)
		}
	}
}
