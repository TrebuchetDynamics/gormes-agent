package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/candidate"

type CandidateStatus = candidate.CandidateStatus

const (
	CandidateStatusCandidate = candidate.CandidateStatusCandidate
	CandidateStatusPromoted  = candidate.CandidateStatusPromoted
	CandidateStatusRejected  = candidate.CandidateStatusRejected

	ActiveStatus = candidate.ActiveStatus
)

type CandidateDraft = candidate.CandidateDraft

type CandidateMetadata = candidate.CandidateMetadata

type ActiveMetadata = candidate.ActiveMetadata
