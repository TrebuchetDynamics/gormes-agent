package renderframe

import "github.com/TrebuchetDynamics/gormes-agent/internal/kernel"

// LastAssistantText extracts the Content of the last assistant-role message in
// the frame History. Returns empty string when no assistant message is found.
func LastAssistantText(f kernel.RenderFrame) string {
	for i := len(f.History) - 1; i >= 0; i-- {
		if f.History[i].Role == "assistant" {
			return f.History[i].Content
		}
	}
	return ""
}
