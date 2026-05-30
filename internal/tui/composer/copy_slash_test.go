package composer

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestHandleCopySlash(t *testing.T) {
	history := []llm.Message{
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "one"},
		{Role: "assistant", Content: "<think>hidden</think>Visible answer"},
	}

	latest := HandleCopySlash("/copy", history, true)
	if !latest.Handled || !latest.WriteClipboard || latest.Text != "Visible answer" || !strings.Contains(latest.Status, "#2") {
		t.Fatalf("HandleCopySlash latest = %+v, want clipboard write for visible response #2", latest)
	}

	first := HandleCopySlash("/copy 1", history, true)
	if !first.WriteClipboard || first.Text != "one" || !strings.Contains(first.Status, "#1") {
		t.Fatalf("HandleCopySlash first = %+v, want clipboard write for response #1", first)
	}

	invalid := HandleCopySlash("/copy 99", history, true)
	if invalid.WriteClipboard || invalid.Status != "copy: invalid response number" {
		t.Fatalf("HandleCopySlash invalid = %+v, want invalid-index status", invalid)
	}

	unavailable := HandleCopySlash("/copy", history, false)
	if unavailable.WriteClipboard || unavailable.Status != "copy: clipboard unavailable" {
		t.Fatalf("HandleCopySlash unavailable = %+v, want unavailable status", unavailable)
	}
}
