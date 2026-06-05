package contextfiles

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles/references"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript/contextrefs"
)

const (
	ContextReferenceFile   = references.ContextReferenceFile
	ContextReferenceFolder = references.ContextReferenceFolder
	ContextReferenceGit    = references.ContextReferenceGit
	ContextReferenceURL    = references.ContextReferenceURL
	ContextReferenceDiff   = references.ContextReferenceDiff
	ContextReferenceStaged = references.ContextReferenceStaged
)

type ContextReference = references.ContextReference

type ContextReferenceHandleResult = references.ContextReferenceHandleResult

func ParseContextReferences(message string) []ContextReference {
	return references.ParseContextReferences(message)
}

func AttachContextReferenceHandles(message string, store *contextrefs.Store) ContextReferenceHandleResult {
	return references.AttachContextReferenceHandles(message, store)
}

func RemoveContextReferenceTokens(message string, refs []ContextReference) string {
	return references.RemoveContextReferenceTokens(message, refs)
}
