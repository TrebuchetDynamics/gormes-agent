package parsers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const upstreamParsersRoot = "../../../../hermes-agent/environments/tool_call_parsers"

func TestToolCallParserManifestCoversEveryUpstreamFile(t *testing.T) {
	manifest := ToolCallParserManifest()
	if len(manifest) == 0 {
		t.Fatal("ToolCallParserManifest() returned no entries")
	}

	if _, err := os.Stat(upstreamParsersRoot); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("upstream Hermes checkout not present at %s", upstreamParsersRoot)
		}
		t.Fatalf("stat upstream parsers root: %v", err)
	}

	entries, err := os.ReadDir(upstreamParsersRoot)
	if err != nil {
		t.Fatalf("read upstream parsers root: %v", err)
	}

	upstreamFiles := map[string]struct{}{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(name, ".py") {
			continue
		}
		if name == "__init__.py" {
			continue
		}
		upstreamFiles[name] = struct{}{}
	}
	if len(upstreamFiles) == 0 {
		t.Fatal("no upstream parser .py files discovered")
	}

	manifestFiles := map[string]ToolCallParserEntry{}
	for _, entry := range manifest {
		if _, dup := manifestFiles[entry.UpstreamFile]; dup {
			t.Fatalf("manifest contains duplicate upstream_file=%q", entry.UpstreamFile)
		}
		manifestFiles[entry.UpstreamFile] = entry
	}

	for upstream := range upstreamFiles {
		entry, ok := manifestFiles[upstream]
		if !ok {
			t.Fatalf("manifest missing upstream parser file %q (acceptance: docs test fails when a new upstream parser appears without a classification)", upstream)
		}
		if entry.ModelFamily == "" {
			t.Fatalf("manifest entry %q missing model_family", upstream)
		}
		if entry.ExpectedInputStyle == "" {
			t.Fatalf("manifest entry %q missing expected_input_style", upstream)
		}
		if entry.TargetGoPackage == "" {
			t.Fatalf("manifest entry %q missing target_go_package", upstream)
		}
		switch entry.Status {
		case ToolCallParserStatusMapped, ToolCallParserStatusRowBacked:
		default:
			t.Fatalf("manifest entry %q has invalid status %q (want mapped|row_backed)", upstream, entry.Status)
		}
	}

	for upstream := range manifestFiles {
		if _, ok := upstreamFiles[upstream]; !ok {
			t.Fatalf("manifest references upstream parser file %q that does not exist on disk", upstream)
		}
	}
}

func TestToolCallParserManifestExposesAllRequiredFields(t *testing.T) {
	manifest := ToolCallParserManifest()
	required := []string{
		"hermes_parser.py",
		"deepseek_v3_1_parser.py",
	}
	idx := map[string]ToolCallParserEntry{}
	for _, e := range manifest {
		idx[e.UpstreamFile] = e
	}
	for _, name := range required {
		entry, ok := idx[name]
		if !ok {
			t.Fatalf("manifest missing %s", name)
		}
		if entry.ModelFamily == "" || entry.ExpectedInputStyle == "" || entry.TargetGoPackage == "" {
			t.Fatalf("entry %q missing required fields: %+v", name, entry)
		}
	}
}

func TestToolCallParserManifestStatusReflectsFixtures(t *testing.T) {
	manifest := ToolCallParserManifest()
	mappedExpected := map[string]bool{
		"hermes_parser.py":        true,
		"deepseek_v3_1_parser.py": true,
		"qwen_parser.py":          true,
		"mistral_parser.py":       true,
		"llama_parser.py":         true,
		"glm45_parser.py":         true,
		"glm47_parser.py":         true,
	}
	for _, entry := range manifest {
		if mappedExpected[entry.UpstreamFile] {
			if entry.Status != ToolCallParserStatusMapped {
				t.Fatalf("entry %q expected mapped, got %q", entry.UpstreamFile, entry.Status)
			}
			if len(entry.GoldenFixtures) == 0 {
				t.Fatalf("entry %q is mapped but has no golden fixtures", entry.UpstreamFile)
			}
		} else {
			if entry.Status != ToolCallParserStatusRowBacked {
				t.Fatalf("entry %q should remain row_backed until fixtures land, got %q", entry.UpstreamFile, entry.Status)
			}
			if len(entry.GoldenFixtures) != 0 {
				t.Fatalf("entry %q is row_backed but has unexpected fixtures %v", entry.UpstreamFile, entry.GoldenFixtures)
			}
		}
	}
}

func TestToolCallParserManifestGoldenFixturesExistAndCoverDegraded(t *testing.T) {
	manifest := ToolCallParserManifest()
	for _, entry := range manifest {
		if entry.Status != ToolCallParserStatusMapped {
			continue
		}
		if len(entry.GoldenFixtures) == 0 {
			t.Fatalf("mapped entry %q has no golden fixtures", entry.UpstreamFile)
		}
		hasDegraded := false
		for _, fx := range entry.GoldenFixtures {
			path := filepath.Join("..", "..", "testdata", "tool_call_parsers", fx)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture %s for %s: %v", path, entry.UpstreamFile, err)
			}
			var parsed struct {
				Parser         string `json:"parser"`
				Label          string `json:"label"`
				RawOutput      string `json:"raw_output"`
				Degraded       bool   `json:"degraded"`
				DegradedReason string `json:"degraded_reason"`
			}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("decode fixture %s: %v", path, err)
			}
			if parsed.Parser == "" || parsed.Label == "" || parsed.RawOutput == "" {
				t.Fatalf("fixture %s has empty parser/label/raw_output", path)
			}
			if parsed.Degraded && parsed.DegradedReason == "" {
				t.Fatalf("fixture %s marked degraded but degraded_reason is empty", path)
			}
			if parsed.Degraded {
				hasDegraded = true
			}
		}
		if !hasDegraded {
			t.Fatalf("entry %q golden fixtures lack a malformed/degraded case", entry.UpstreamFile)
		}
	}
}

func TestRawToolCallParserManifestDoesNotInvokeRuntime(t *testing.T) {
	manifest := ToolCallParserManifest()
	for _, entry := range manifest {
		if strings.Contains(strings.ToLower(entry.TargetGoPackage), "python") {
			t.Fatalf("entry %q target_go_package %q must not reference python runtime", entry.UpstreamFile, entry.TargetGoPackage)
		}
		if strings.Contains(strings.ToLower(entry.ExpectedInputStyle), "live") {
			t.Fatalf("entry %q expected_input_style %q must not reference live model runs", entry.UpstreamFile, entry.ExpectedInputStyle)
		}
	}
}

func TestRawToolCallParserManifestStableOrder(t *testing.T) {
	first := ToolCallParserManifest()
	second := ToolCallParserManifest()
	if len(first) != len(second) {
		t.Fatalf("manifest length unstable: %d vs %d", len(first), len(second))
	}
	names := make([]string, 0, len(first))
	for _, entry := range first {
		names = append(names, entry.UpstreamFile)
	}
	if !sort.StringsAreSorted(names) {
		t.Fatalf("manifest must be sorted by upstream_file; got %v", names)
	}
	for i := range first {
		if !reflect.DeepEqual(first[i], second[i]) {
			t.Fatalf("manifest entry %d differs between calls: %+v vs %+v", i, first[i], second[i])
		}
	}
}
