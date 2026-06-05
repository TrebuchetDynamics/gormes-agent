package llm

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript/contextrefs"
)

const (
	ContextReferenceFile   = contextfiles.ContextReferenceFile
	ContextReferenceFolder = contextfiles.ContextReferenceFolder
	ContextReferenceGit    = contextfiles.ContextReferenceGit
	ContextReferenceURL    = contextfiles.ContextReferenceURL
	ContextReferenceDiff   = contextfiles.ContextReferenceDiff
	ContextReferenceStaged = contextfiles.ContextReferenceStaged
)

type ContextReference = contextfiles.ContextReference

type ContextReferenceHandleResult = contextfiles.ContextReferenceHandleResult

func ParseContextReferences(message string) []ContextReference {
	return contextfiles.ParseContextReferences(message)
}

func AttachContextReferenceHandles(message string, store *contextrefs.Store) ContextReferenceHandleResult {
	return contextfiles.AttachContextReferenceHandles(message, store)
}

func RemoveContextReferenceTokens(message string, refs []ContextReference) string {
	return contextfiles.RemoveContextReferenceTokens(message, refs)
}
