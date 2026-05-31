package contextfiles

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles/project"

type ContextFilesOptions = project.ContextFilesOptions

type ContextFilesReport = project.ContextFilesReport

// ContextFileEvidence describes one context source considered for prompt input.
type ContextFileEvidence = project.ContextFileEvidence

func BuildContextFilesPrompt(opts ContextFilesOptions) (string, ContextFilesReport) {
	return project.BuildContextFilesPrompt(opts)
}
