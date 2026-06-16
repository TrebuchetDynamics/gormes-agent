package renderframe

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestLastAssistantTextReturnsLastAssistantMessage(t *testing.T) {
	frame := kernel.RenderFrame{History: []llm.Message{
		{Role: "assistant", Content: "first"},
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "second"},
	}}
	if got := LastAssistantText(frame); got != "second" {
		t.Fatalf("LastAssistantText = %q, want second", got)
	}
}

func TestLastAssistantTextMissingAssistant(t *testing.T) {
	frame := kernel.RenderFrame{History: []llm.Message{{Role: "user", Content: "question"}}}
	if got := LastAssistantText(frame); got != "" {
		t.Fatalf("LastAssistantText = %q, want empty", got)
	}
}
