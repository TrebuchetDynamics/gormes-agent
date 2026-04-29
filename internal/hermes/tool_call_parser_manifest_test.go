package hermes

import (
	"reflect"
	"testing"
)

func TestToolCallParserManifest_ClassifiesUpstreamFamilies(t *testing.T) {
	manifest := DefaultToolCallParserManifest()

	hermesParser, ok := manifest.Lookup("hermes_parser.py")
	if !ok {
		t.Fatal("hermes_parser.py missing from manifest")
	}
	if hermesParser.Family != "hermes" {
		t.Fatalf("hermes family = %q", hermesParser.Family)
	}
	if hermesParser.Status != ToolCallParserStatusRowBacked {
		t.Fatalf("hermes status = %q", hermesParser.Status)
	}
	if hermesParser.ProgressRow != "5.M Raw tool-call parser fixture matrix" {
		t.Fatalf("hermes progress row = %q", hermesParser.ProgressRow)
	}

	deepseek, ok := manifest.Lookup("deepseek_v3_1_parser.py")
	if !ok {
		t.Fatal("deepseek_v3_1_parser.py missing from manifest")
	}
	if deepseek.Family != "deepseek_v31" {
		t.Fatalf("deepseek family = %q", deepseek.Family)
	}
	if deepseek.InputStyle != "deepseek-v3.1" {
		t.Fatalf("deepseek input style = %q", deepseek.InputStyle)
	}
	if deepseek.Status != ToolCallParserStatusRowBacked {
		t.Fatalf("deepseek status = %q", deepseek.Status)
	}

	upstream := []string{
		"__init__.py",
		"deepseek_v3_1_parser.py",
		"deepseek_v3_parser.py",
		"glm45_parser.py",
		"glm47_parser.py",
		"hermes_parser.py",
		"kimi_k2_parser.py",
		"llama_parser.py",
		"longcat_parser.py",
		"mistral_parser.py",
		"qwen3_coder_parser.py",
		"qwen_parser.py",
	}
	if missing := manifest.MissingUpstreamFiles(upstream); len(missing) != 0 {
		t.Fatalf("missing upstream parser files = %v", missing)
	}
}

func TestToolCallParserManifest_UnknownFamiliesStayExplicit(t *testing.T) {
	manifest := DefaultToolCallParserManifest()
	got := manifest.UnknownFamilies([]string{
		"hermes_parser.py",
		"new_model_parser.py",
		"notes.txt",
	})
	want := []string{"new_model_parser.py"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UnknownFamilies() = %v, want %v", got, want)
	}
}
