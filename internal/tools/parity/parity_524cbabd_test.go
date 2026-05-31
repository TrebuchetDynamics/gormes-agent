package parity

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestToolParityManifestHermes524CBABDPatchSchema(t *testing.T) {
	manifest, err := LoadUpstreamToolParityManifest()
	if err != nil {
		t.Fatalf("LoadUpstreamToolParityManifest: %v", err)
	}

	if got, want := manifest.Source.Commit, "524cbabd8"; !strings.HasPrefix(got, want) {
		t.Fatalf("source commit = %q, want prefix %q", got, want)
	}
	for _, input := range []string{
		"tools/file_tools.py",
		"tests/tools/test_file_tools.py",
	} {
		assertContains(t, manifest.Source.InputFiles, input)
	}

	patch := mustTool(t, manifest, "patch")
	var schema struct {
		Description string `json:"description"`
		Parameters  struct {
			Required   []string                   `json:"required"`
			Properties map[string]map[string]any  `json:"properties"`
			AnyOf      []map[string]any           `json:"anyOf"`
			OneOf      []map[string]any           `json:"oneOf"`
			Extra      map[string]json.RawMessage `json:"-"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(patch.Schema, &schema); err != nil {
		t.Fatalf("unmarshal patch schema: %v", err)
	}

	assertContainsString(t, schema.Description, "REQUIRED PARAMETERS: mode, path, old_string, new_string")
	assertContainsString(t, schema.Description, "REQUIRED PARAMETERS: mode, patch")
	if !reflect.DeepEqual(schema.Parameters.Required, []string{"mode"}) {
		t.Fatalf("patch schema required = %#v, want [mode]", schema.Parameters.Required)
	}
	if len(schema.Parameters.AnyOf) != 0 {
		t.Fatalf("patch schema must not use anyOf: %#v", schema.Parameters.AnyOf)
	}
	if len(schema.Parameters.OneOf) != 0 {
		t.Fatalf("patch schema must not use oneOf: %#v", schema.Parameters.OneOf)
	}

	for property, want := range map[string]string{
		"path":       "REQUIRED when mode='replace'",
		"old_string": "REQUIRED when mode='replace'",
		"new_string": "REQUIRED when mode='replace'",
		"patch":      "REQUIRED when mode='patch'",
	} {
		got, _ := schema.Parameters.Properties[property]["description"].(string)
		assertContainsString(t, got, want)
	}
}

func assertContainsString(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("string missing %q in:\n%s", want, got)
	}
}
