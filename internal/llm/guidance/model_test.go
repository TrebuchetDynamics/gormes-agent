package guidance

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

func TestModelPromptRoleForDeveloperModels(t *testing.T) {
	tests := []struct {
		model string
		want  PromptRole
	}{
		{"gpt-5.1-codex", PromptRoleDeveloper},
		{"codex-mini-latest", PromptRoleDeveloper},
		{"openai/gpt-5.2", PromptRoleDeveloper},
		{"gemini-2.5-pro", PromptRoleSystem},
		{"claude-sonnet-4-5", PromptRoleSystem},
		{"openrouter/auto", PromptRoleSystem},
		{"", PromptRoleSystem},
	}
	for _, tt := range tests {
		if got := ModelPromptRole(tt.model); got != tt.want {
			t.Fatalf("ModelPromptRole(%q)=%q want %q", tt.model, got, tt.want)
		}
	}
}

func TestToolUseEnforcementConfigModes(t *testing.T) {
	validTools := []string{"read_file", "terminal"}
	tests := []struct {
		name        string
		model       string
		mode        any
		want        bool
		wantDefault bool
	}{
		{"always bool", "claude-sonnet", true, true, false},
		{"always string", "claude-sonnet", "always", true, false},
		{"never bool", "gpt-5", false, false, false},
		{"never off", "gpt-5", "off", false, false},
		{"auto matches", "gpt-5", "auto", true, false},
		{"auto misses", "claude-sonnet", "auto", false, false},
		{"family string matches", "claude-sonnet", "claude", true, false},
		{"family string misses", "gemini-2.5-pro", "claude", false, false},
		{"explicit list matches", "qwen-coder", []string{"claude", "qwen"}, true, false},
		{"explicit list misses", "grok", []string{"claude", "qwen"}, false, false},
		{"malformed defaults to auto", "gpt-5", 123, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildModelPromptGuidance(ModelPromptGuidanceOptions{
				Model:                  tt.model,
				ValidToolNames:         validTools,
				ToolUseEnforcementMode: tt.mode,
			})
			has := strings.Contains(got.Guidance, "# Tool-use enforcement")
			if has != tt.want {
				t.Fatalf("tool guidance presence=%v want %v\nguidance=%s", has, tt.want, got.Guidance)
			}
			if got.ToolUseEnforcementDefaulted != tt.wantDefault {
				t.Fatalf("defaulted=%v want %v", got.ToolUseEnforcementDefaulted, tt.wantDefault)
			}
			if tt.wantDefault && !containsModelGuidanceEvidence(got.Evidence, "tool_use_enforcement_defaulted") {
				t.Fatalf("expected tool_use_enforcement_defaulted evidence, got %#v", got.Evidence)
			}
		})
	}
}

func TestToolUseEnforcementRequiresTools(t *testing.T) {
	got := BuildModelPromptGuidance(ModelPromptGuidanceOptions{
		Model:                  "gpt-5",
		ValidToolNames:         nil,
		ToolUseEnforcementMode: "always",
	})
	if strings.Contains(got.Guidance, "# Tool-use enforcement") {
		t.Fatalf("tool guidance emitted without valid tools: %q", got.Guidance)
	}
	if !containsModelGuidanceEvidence(got.Evidence, "tool_use_enforcement_suppressed_no_tools") {
		t.Fatalf("missing no-tools evidence: %#v", got.Evidence)
	}
}

func TestModelOperationalGuidanceByFamily(t *testing.T) {
	tests := []struct {
		model      string
		wantGoogle bool
		wantOpenAI bool
	}{
		{"gemini-2.5-pro", true, false},
		{"gemma-3", true, false},
		{"gpt-5", false, true},
		{"codex-mini", false, true},
		{"claude-sonnet", false, false},
	}
	for _, tt := range tests {
		got := BuildModelPromptGuidance(ModelPromptGuidanceOptions{Model: tt.model, ValidToolNames: []string{"read_file"}})
		hasGoogle := strings.Contains(got.Guidance, "# Google model operational directives")
		hasOpenAI := strings.Contains(got.Guidance, "# Execution discipline")
		if hasGoogle != tt.wantGoogle || hasOpenAI != tt.wantOpenAI {
			t.Fatalf("%s google=%v want %v openai=%v want %v\n%s", tt.model, hasGoogle, tt.wantGoogle, hasOpenAI, tt.wantOpenAI, got.Guidance)
		}
	}
}

func TestResearchQualityGuidanceRequiresWebSearchTool(t *testing.T) {
	got := BuildModelPromptGuidance(ModelPromptGuidanceOptions{
		Model:          "gpt-5",
		ValidToolNames: []string{"read_file", "web_search", "web_extract"},
	})
	for _, want := range []string{
		"# Research quality",
		"open-source projects",
		"maturity",
		"license",
		"project-specific fit",
		"migration workflow",
	} {
		if !strings.Contains(got.Guidance, want) {
			t.Fatalf("research guidance missing %q:\n%s", want, got.Guidance)
		}
	}
	if !containsModelGuidanceEvidence(got.Evidence, "research_quality_guidance_injected") {
		t.Fatalf("missing injected evidence: %#v", got.Evidence)
	}

	withoutSearch := BuildModelPromptGuidance(ModelPromptGuidanceOptions{
		Model:          "gpt-5",
		ValidToolNames: []string{"read_file", "web_extract"},
	})
	if strings.Contains(withoutSearch.Guidance, "# Research quality") {
		t.Fatalf("research guidance emitted without web_search:\n%s", withoutSearch.Guidance)
	}
	if !containsModelGuidanceEvidence(withoutSearch.Evidence, "research_quality_guidance_suppressed_no_web_search") {
		t.Fatalf("missing suppressed evidence: %#v", withoutSearch.Evidence)
	}
}

func TestPromptGuidanceIsPure(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "model.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse model.go: %v", err)
	}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("unquote import: %v", err)
		}
		if strings.Contains(path, "net/http") || strings.Contains(path, "os") || strings.Contains(path, "internal/config") || strings.Contains(path, "internal/tools") {
			t.Fatalf("model guidance helper must stay pure; unexpected import %q", path)
		}
	}
}

func containsModelGuidanceEvidence(evidence []string, want string) bool {
	for _, got := range evidence {
		if got == want {
			return true
		}
	}
	return false
}
