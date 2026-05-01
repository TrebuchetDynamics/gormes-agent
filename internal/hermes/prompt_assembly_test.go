package hermes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptAssembly_BlockOrder(t *testing.T) {
	tmp := t.TempDir()
	soul := filepath.Join(tmp, "SOUL.md")
	os.WriteFile(soul, []byte("Test identity."), 0644)

	opts := PromptAssemblyOptions{
		IdentityOpts:     IdentityLoaderOptions{ProfileDir: tmp},
		ContextFilesOpts: ContextFilesOptions{CWD: tmp, SkipSoul: true},
		HasMemoryTool:    true,
		HasSessionSearch: true,
		SkillsOpts:       SkillsPromptOptions{LocalDir: tmp},
		ModelGuidanceOpts: ModelPromptGuidanceOptions{
			Model:                  "test-model",
			ValidToolNames:         []string{"memory", "session_search"},
			ToolUseEnforcementMode: true,
		},
	}

	result := BuildSystemPrompt(opts)
	if result.Prompt == "" {
		t.Fatal("expected non-empty prompt")
	}

	blocks := result.Blocks
	if len(blocks) == 0 {
		t.Fatal("expected block evidence")
	}

	if blocks[0].Block != "identity" {
		t.Fatalf("expected first block=identity, got %s", blocks[0].Block)
	}

	foundMemory := false
	foundSessionSearch := false
	for _, b := range blocks {
		switch b.Block {
		case "memory_guidance":
			foundMemory = true
		case "session_search_guidance":
			foundSessionSearch = true
		}
	}
	if !foundMemory {
		t.Fatal("expected memory_guidance block")
	}
	if !foundSessionSearch {
		t.Fatal("expected session_search_guidance block")
	}
}

func TestPromptAssembly_MissingBlockEvidence(t *testing.T) {
	tmp := t.TempDir()
	opts := PromptAssemblyOptions{
		IdentityOpts:     IdentityLoaderOptions{ProfileDir: tmp},
		ContextFilesOpts: ContextFilesOptions{CWD: tmp, SkipSoul: true},
		HasMemoryTool:    false,
		HasSessionSearch: false,
		SkillsOpts:       SkillsPromptOptions{LocalDir: tmp},
		ModelGuidanceOpts: ModelPromptGuidanceOptions{
			Model: "test-model",
		},
	}

	result := BuildSystemPrompt(opts)
	if result.Prompt == "" {
		t.Fatal("expected non-empty prompt (at least identity)")
	}

	for _, b := range result.Blocks {
		if b.Block == "memory_guidance" && b.Included {
			t.Fatal("expected memory_guidance to be excluded when HasMemoryTool=false")
		}
		if b.Block == "session_search_guidance" && b.Included {
			t.Fatal("expected session_search_guidance to be excluded when HasSessionSearch=false")
		}
	}
}

func TestPromptAssembly_Deterministic(t *testing.T) {
	tmp := t.TempDir()
	soul := filepath.Join(tmp, "SOUL.md")
	os.WriteFile(soul, []byte("Test identity."), 0644)

	opts := PromptAssemblyOptions{
		IdentityOpts:     IdentityLoaderOptions{ProfileDir: tmp},
		ContextFilesOpts: ContextFilesOptions{CWD: tmp, SkipSoul: true},
		HasMemoryTool:    true,
		HasSessionSearch: true,
		SkillsOpts:       SkillsPromptOptions{LocalDir: tmp},
		ModelGuidanceOpts: ModelPromptGuidanceOptions{
			Model:                  "test-model",
			ValidToolNames:         []string{"memory"},
			ToolUseEnforcementMode: true,
		},
	}

	r1 := BuildSystemPrompt(opts)
	r2 := BuildSystemPrompt(opts)
	if r1.Prompt != r2.Prompt {
		t.Fatal("BuildSystemPrompt must be deterministic")
	}
}

func TestPromptAssembly_Pure(t *testing.T) {
	tmp := t.TempDir()
	opts := PromptAssemblyOptions{
		IdentityOpts:     IdentityLoaderOptions{ProfileDir: tmp},
		ContextFilesOpts: ContextFilesOptions{CWD: tmp, SkipSoul: true},
		HasMemoryTool:    false,
		HasSessionSearch: false,
		SkillsOpts:       SkillsPromptOptions{LocalDir: tmp},
		ModelGuidanceOpts: ModelPromptGuidanceOptions{
			Model: "test-model",
		},
	}

	result := BuildSystemPrompt(opts)
	if result.Prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if strings.Contains(result.Prompt, "python") {
		t.Fatal("prompt must not contain python references")
	}
}
