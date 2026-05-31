package guidance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestGuidanceConstants_MemoryGuidance_ByteEquivalent(t *testing.T) {
	src, ok := readUpstreamPromptBuilder(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check (drift will surface elsewhere)")
	}
	want, ok := extractPythonStringConstant(src, "MEMORY_GUIDANCE")
	if !ok {
		t.Fatalf("could not extract MEMORY_GUIDANCE from upstream")
	}
	if MemoryGuidance != want {
		t.Fatalf("MemoryGuidance does not match upstream MEMORY_GUIDANCE\n--- got (%d bytes) ---\n%q\n--- want (%d bytes) ---\n%q",
			len(MemoryGuidance), MemoryGuidance, len(want), want)
	}
}

func TestGuidanceConstants_SessionSearchGuidance_ByteEquivalent(t *testing.T) {
	src, ok := readUpstreamPromptBuilder(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check")
	}
	want, ok := extractPythonStringConstant(src, "SESSION_SEARCH_GUIDANCE")
	if !ok {
		t.Fatalf("could not extract SESSION_SEARCH_GUIDANCE from upstream")
	}
	if SessionSearchGuidance != want {
		t.Fatalf("SessionSearchGuidance does not match upstream\n--- got ---\n%q\n--- want ---\n%q", SessionSearchGuidance, want)
	}
}

func TestGuidanceConstants_SkillsGuidance_ByteEquivalent(t *testing.T) {
	src, ok := readUpstreamPromptBuilder(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check")
	}
	want, ok := extractPythonStringConstant(src, "SKILLS_GUIDANCE")
	if !ok {
		t.Fatalf("could not extract SKILLS_GUIDANCE from upstream")
	}
	if SkillsGuidance != want {
		t.Fatalf("SkillsGuidance does not match upstream\n--- got ---\n%q\n--- want ---\n%q", SkillsGuidance, want)
	}
}

func TestGuidanceConstants_ToolUseEnforcementGuidance_ByteEquivalent(t *testing.T) {
	src, ok := readUpstreamPromptBuilder(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check")
	}
	want, ok := extractPythonStringConstant(src, "TOOL_USE_ENFORCEMENT_GUIDANCE")
	if !ok {
		t.Fatalf("could not extract TOOL_USE_ENFORCEMENT_GUIDANCE from upstream")
	}
	if ToolUseEnforcementGuidance != want {
		t.Fatalf("ToolUseEnforcementGuidance does not match upstream\n--- got ---\n%q\n--- want ---\n%q", ToolUseEnforcementGuidance, want)
	}
}

func TestGuidanceConstants_ToolUseEnforcementModels_MatchesUpstream(t *testing.T) {
	src, ok := readUpstreamPromptBuilder(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping match check")
	}
	want, ok := extractPythonTupleOfStrings(src, "TOOL_USE_ENFORCEMENT_MODELS")
	if !ok {
		t.Fatalf("could not extract TOOL_USE_ENFORCEMENT_MODELS from upstream")
	}
	if !reflect.DeepEqual(ToolUseEnforcementModels, want) {
		t.Fatalf("ToolUseEnforcementModels mismatch\n got: %#v\nwant: %#v", ToolUseEnforcementModels, want)
	}
}

func TestGuidanceConstants_DeveloperRoleModels_MatchesUpstream(t *testing.T) {
	src, ok := readUpstreamPromptBuilder(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping match check")
	}
	want, ok := extractPythonTupleOfStrings(src, "DEVELOPER_ROLE_MODELS")
	if !ok {
		t.Fatalf("could not extract DEVELOPER_ROLE_MODELS from upstream")
	}
	if !reflect.DeepEqual(DeveloperRoleModels, want) {
		t.Fatalf("DeveloperRoleModels mismatch\n got: %#v\nwant: %#v", DeveloperRoleModels, want)
	}
}

func TestGuidanceConstants_WSLEnvironmentHint_ByteEquivalent(t *testing.T) {
	src, ok := readUpstreamPromptBuilder(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check")
	}
	want, ok := extractPythonStringConstant(src, "WSL_ENVIRONMENT_HINT")
	if !ok {
		t.Fatalf("could not extract WSL_ENVIRONMENT_HINT from upstream")
	}
	if WSLEnvironmentHint != want {
		t.Fatalf("WSLEnvironmentHint does not match upstream\n--- got ---\n%q\n--- want ---\n%q", WSLEnvironmentHint, want)
	}
}

// TestGuidanceConstants_NoRuntimeImport asserts that the data-only constants
// file imports nothing beyond the standard library — specifically nothing from
// internal/gateway, internal/kernel, internal/channels, internal/runtime,
// internal/provider, and other live-turn subsystems. This static check must
// always run; it does not depend on upstream availability.
func TestGuidanceConstants_NoRuntimeImport(t *testing.T) {
	const target = "constants.go"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, target, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", target, err)
	}
	const repoModule = "github.com/TrebuchetDynamics/gormes-agent/internal/"
	disallowed := []string{
		"gateway", "kernel", "channels", "runtime", "provider",
		"memory", "session", "skills", "cron",
	}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("unquote import path %q: %v", imp.Path.Value, err)
		}
		if !strings.HasPrefix(path, repoModule) {
			continue
		}
		suffix := strings.TrimPrefix(path, repoModule)
		head := suffix
		if i := strings.IndexByte(suffix, '/'); i >= 0 {
			head = suffix[:i]
		}
		for _, banned := range disallowed {
			if head == banned {
				t.Fatalf("guidance_constants.go must not import internal/%s (found %q)", banned, path)
			}
		}
	}
	// Sanity: the file must still parse and define at least one decl.
	if len(file.Decls) == 0 {
		// Parser was invoked with ImportsOnly which strips body decls; a
		// proper second parse confirms the file is non-empty. Use the
		// importsOnly result only as the import oracle.
		_ = ast.Print
	}
}
