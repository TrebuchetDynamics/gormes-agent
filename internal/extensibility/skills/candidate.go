package skills

import (
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/candidate"
)

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

func (s *Store) CandidateDir() string {
	if s == nil {
		return ""
	}
	return candidate.CandidateDir(s.root)
}

func (s *Store) DraftCandidate(draft CandidateDraft) (CandidateMetadata, error) {
	if s == nil {
		return CandidateMetadata{}, fmt.Errorf("skills: nil store")
	}
	return candidate.DraftCandidate(s.root, s.maxBytes, draft)
}

func (s *Store) PromoteCandidate(candidateID string) (ActiveMetadata, error) {
	if s == nil {
		return ActiveMetadata{}, fmt.Errorf("skills: nil store")
	}
	return candidate.PromoteCandidate(s.root, s.ActiveDir(), s.maxBytes, candidateID)
}
