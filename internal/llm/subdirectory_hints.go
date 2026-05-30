package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles"

const (
	subdirectoryHintsDefaultMaxChars     = 8000
	subdirectoryHintsDefaultAncestorWalk = 5
	SubdirectoryHintEvidenceLoaded       = contextfiles.SubdirectoryHintEvidenceLoaded
	SubdirectoryHintEvidenceStatError    = contextfiles.SubdirectoryHintEvidenceStatError
	SubdirectoryHintEvidenceReadError    = contextfiles.SubdirectoryHintEvidenceReadError
	SubdirectoryHintEvidenceEmpty        = contextfiles.SubdirectoryHintEvidenceEmpty
)

type SubdirectoryHintOptions = contextfiles.SubdirectoryHintOptions
type SubdirectoryHintTracker = contextfiles.SubdirectoryHintTracker
type SubdirectoryHintResult = contextfiles.SubdirectoryHintResult
type SubdirectoryHint = contextfiles.SubdirectoryHint
type SubdirectoryHintEvidence = contextfiles.SubdirectoryHintEvidence

func NewSubdirectoryHintTracker(opts SubdirectoryHintOptions) *SubdirectoryHintTracker {
	return contextfiles.NewSubdirectoryHintTracker(opts)
}
