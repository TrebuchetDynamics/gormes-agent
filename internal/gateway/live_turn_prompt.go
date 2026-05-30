package gateway

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/liveprompt"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type liveTurnPromptSeams = liveprompt.Seams

func defaultLiveTurnPromptSeams() liveTurnPromptSeams { return liveprompt.DefaultSeams() }

func defaultLiveTurnCWD() string { return liveprompt.DefaultCWD() }

func defaultLiveTurnProfileDir(cwd string) string { return liveprompt.DefaultProfileDir(cwd) }

func defaultLiveTurnMemoryDir(cwd string) string { return liveprompt.DefaultMemoryDir(cwd) }

func assembleLiveTurnPrompt(seams liveTurnPromptSeams, submitText, activeSessionID, sessionBlock string) (string, llm.ContextFilesReport, llm.DurableUserContextReport) {
	return liveprompt.Assemble(seams, submitText, activeSessionID, sessionBlock)
}
