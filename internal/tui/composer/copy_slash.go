package composer

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer/copyslash"
)

type CopySlashResult = copyslash.CopySlashResult

func HandleCopySlash(input string, history []llm.Message, clipboardAvailable bool) CopySlashResult {
	return copyslash.HandleCopySlash(input, history, clipboardAvailable)
}
