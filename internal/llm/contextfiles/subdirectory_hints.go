package contextfiles

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles/subdirhints"

const (
	SubdirectoryHintEvidenceLoaded    = subdirhints.SubdirectoryHintEvidenceLoaded
	SubdirectoryHintEvidenceStatError = subdirhints.SubdirectoryHintEvidenceStatError
	SubdirectoryHintEvidenceReadError = subdirhints.SubdirectoryHintEvidenceReadError
	SubdirectoryHintEvidenceEmpty     = subdirhints.SubdirectoryHintEvidenceEmpty
)

type SubdirectoryHintOptions = subdirhints.SubdirectoryHintOptions

type SubdirectoryHintTracker = subdirhints.SubdirectoryHintTracker

type SubdirectoryHintResult = subdirhints.SubdirectoryHintResult

type SubdirectoryHint = subdirhints.SubdirectoryHint

type SubdirectoryHintEvidence = subdirhints.SubdirectoryHintEvidence

func NewSubdirectoryHintTracker(opts SubdirectoryHintOptions) *SubdirectoryHintTracker {
	return subdirhints.NewSubdirectoryHintTracker(opts)
}
