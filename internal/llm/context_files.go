package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles"

const (
	contextFilesDefaultMaxChars = 20000
	contextFilesHeadRatio       = 0.65
	contextFilesTailRatio       = 0.35
)

type ContextFilesOptions = contextfiles.ContextFilesOptions
type ContextFilesReport = contextfiles.ContextFilesReport
type ContextFileEvidence = contextfiles.ContextFileEvidence

func BuildContextFilesPrompt(opts ContextFilesOptions) (string, ContextFilesReport) {
	return contextfiles.BuildContextFilesPrompt(opts)
}
