package builderloop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeValidProgressJSON(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(completeProgressFixture(t, content)), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func completeProgressFixture(t *testing.T, content string) string {
	t.Helper()

	var doc map[string]any
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("fixture progress JSON must be valid: %v", err)
	}
	if _, ok := doc["meta"]; !ok {
		doc["meta"] = map[string]any{
			"version":      "2.0",
			"last_updated": "2026-04-27",
			"links": map[string]any{
				"github_readme": "https://example.test/readme",
				"landing_page":  "https://example.test",
				"docs_site":     "https://example.test/docs",
				"source_code":   "https://example.test/source",
			},
		}
	}

	phases, _ := doc["phases"].(map[string]any)
	for _, phaseValue := range phases {
		phase, _ := phaseValue.(map[string]any)
		subphases, _ := phase["subphases"].(map[string]any)
		if len(subphases) == 0 {
			subphases, _ = phase["sub_phases"].(map[string]any)
		}
		for _, subphaseValue := range subphases {
			subphase, _ := subphaseValue.(map[string]any)
			items, _ := subphase["items"].([]any)
			for _, itemValue := range items {
				item, _ := itemValue.(map[string]any)
				if _, ok := item["contract"].(string); !ok {
					continue
				}
				fillContractFixtureDefaults(item)
			}
		}
	}

	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal completed progress fixture: %v", err)
	}
	return string(body)
}

func fillContractFixtureDefaults(item map[string]any) {
	if _, ok := item["contract_status"]; !ok {
		item["contract_status"] = "draft"
	}
	if _, ok := item["slice_size"]; !ok {
		item["slice_size"] = "small"
	}
	if _, ok := item["execution_owner"]; !ok {
		item["execution_owner"] = "orchestrator"
	}
	if _, ok := item["ready_when"]; !ok {
		item["ready_when"] = []any{"test fixture is ready"}
	}
	if _, ok := item["write_scope"]; !ok {
		item["write_scope"] = []any{"."}
	}
	if _, hasCommands := item["test_commands"]; !hasCommands {
		if _, hasNoTest := item["no_test_required"]; !hasNoTest {
			item["no_test_required"] = "test fixture uses non-command evidence"
		}
	}
	if _, ok := item["done_signal"]; !ok {
		item["done_signal"] = []any{"test fixture done signal"}
	}

	status, _ := item["status"].(string)
	priority, _ := item["priority"].(string)
	if status == "in_progress" || priority == "P0" {
		if _, ok := item["trust_class"]; !ok {
			item["trust_class"] = []any{"system"}
		}
		if _, ok := item["degraded_mode"]; !ok {
			item["degraded_mode"] = "test fixture degraded mode"
		}
		if _, ok := item["fixture"]; !ok {
			item["fixture"] = "internal/progress/builderloop/test_fixture.go"
		}
		if _, ok := item["source_refs"]; !ok {
			item["source_refs"] = []any{"internal/progress/builderloop/run_test.go"}
		}
		if _, ok := item["acceptance"]; !ok {
			item["acceptance"] = []any{"test fixture acceptance"}
		}
	}
}
