package llm

import (
	"strings"
	"testing"
)

func TestMemoryGuidance_ByteEquivalent(t *testing.T) {
	if MemoryGuidance == "" {
		t.Fatal("MemoryGuidance constant must not be empty")
	}
	if !strings.Contains(MemoryGuidance, "persistent memory") {
		t.Fatal("MemoryGuidance must contain 'persistent memory'")
	}
	if !strings.Contains(MemoryGuidance, "declarative facts") {
		t.Fatal("MemoryGuidance must contain 'declarative facts'")
	}
}

func TestMemoryGuidance_InjectedWhenToolAvailable(t *testing.T) {
	result := BuildMemoryGuidance(true)
	if !result.Injected {
		t.Fatal("expected injected when memory tool is available")
	}
	if result.Guidance != MemoryGuidance {
		t.Fatal("expected full guidance text")
	}
	if result.Evidence != "memory_guidance_injected" {
		t.Fatalf("expected evidence=memory_guidance_injected, got %s", result.Evidence)
	}
}

func TestMemoryGuidance_SuppressedWhenNoTool(t *testing.T) {
	result := BuildMemoryGuidance(false)
	if result.Injected {
		t.Fatal("expected not injected when memory tool is unavailable")
	}
	if result.Guidance != "" {
		t.Fatal("expected empty guidance when suppressed")
	}
	if !strings.Contains(result.Evidence, "memory_guidance_suppressed") {
		t.Fatalf("expected suppression evidence, got %s", result.Evidence)
	}
}

func TestMemoryGuidance_Pure(t *testing.T) {
	r1 := BuildMemoryGuidance(true)
	r2 := BuildMemoryGuidance(true)
	if r1.Guidance != r2.Guidance {
		t.Fatal("BuildMemoryGuidance must be deterministic")
	}
}
