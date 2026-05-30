package composer

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// CopySlashResult is the pure slash-command decision behind /copy. When
// WriteClipboard is true, root package tui writes Text through its injected
// clipboard seam and maps any host error to a visible status.
type CopySlashResult struct {
	Handled        bool
	WriteClipboard bool
	Text           string
	Status         string
}

// HandleCopySlash parses /copy [number], selects visible assistant text, and
// returns the status text to show when no clipboard write should be attempted.
func HandleCopySlash(input string, history []llm.Message, clipboardAvailable bool) CopySlashResult {
	if !clipboardAvailable {
		return CopySlashResult{Handled: true, Status: "copy: clipboard unavailable"}
	}
	fields := strings.Fields(input)
	arg := ""
	if len(fields) > 1 {
		arg = fields[1]
	}
	result := SelectComposerCopyText(history, arg)
	if !result.OK {
		return CopySlashResult{Handled: true, Status: copyStatusForEvidence(result)}
	}
	return CopySlashResult{
		Handled:        true,
		WriteClipboard: true,
		Text:           result.Text,
		Status:         fmt.Sprintf("Copied assistant response #%d to clipboard", result.ResponseNumber),
	}
}

func copyStatusForEvidence(result ComposerCopyResult) string {
	switch result.Evidence {
	case "tui_ingress_copy_invalid_index":
		return "copy: invalid response number"
	case "tui_ingress_copy_empty_response":
		return fmt.Sprintf("copy: assistant response #%d has no visible text", result.ResponseNumber)
	default:
		return "copy: nothing to copy"
	}
}
