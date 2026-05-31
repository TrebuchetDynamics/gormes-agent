package guidance

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

func TestMemoryGuidance_SwitchContract(t *testing.T) {
	assertGuidanceSwitch(t, guidanceSwitchCase{
		name:               "memory",
		guidance:           MemoryGuidance,
		build:              BuildMemoryGuidance,
		injectedEvidence:   "memory_guidance_injected",
		suppressedEvidence: "memory_guidance_suppressed",
	})
}
