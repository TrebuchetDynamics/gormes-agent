package tools

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type UXDecision string

const (
	UXDecisionYes     UXDecision = "yes"
	UXDecisionNo      UXDecision = "no"
	UXDecisionAlways  UXDecision = "always"
	UXDecisionUnknown UXDecision = "unknown"
)

type ApprovalPrompt struct {
	Command       string
	Category      BlocklistCategory
	AffectedPaths []string
	RiskLevel     string
}

type ApprovalUXResult struct {
	Decision    UXDecision
	Persisted   bool
	Evidence    string
	AuditRecord ApprovalAuditRecord
}

type ApprovalAuditRecord struct {
	Timestamp time.Time
	Command   string
	Category  string
	Decision  string
	Context   string
}

type ApprovalSession struct {
	mu          sync.RWMutex
	preferences map[string]UXDecision
	interactive bool
	auditLog    []ApprovalAuditRecord
}

func NewApprovalSession(interactive bool) *ApprovalSession {
	return &ApprovalSession{
		preferences: make(map[string]UXDecision),
		interactive: interactive,
		auditLog:    make([]ApprovalAuditRecord, 0),
	}
}

func (s *ApprovalSession) RequestApproval(prompt ApprovalPrompt, onPrompt func(ApprovalPrompt) UXDecision) ApprovalUXResult {
	if !s.interactive {
		return ApprovalUXResult{
			Decision: UXDecisionNo,
			Evidence: "approval_ui_unavailable: non-interactive context",
			AuditRecord: ApprovalAuditRecord{
				Timestamp: time.Now(),
				Command:   prompt.Command,
				Category:  string(prompt.Category),
				Decision:  "denied",
				Context:   "non-interactive",
			},
		}
	}

	s.mu.RLock()
	pref, hasPref := s.preferences[string(prompt.Category)]
	s.mu.RUnlock()

	if hasPref {
		return ApprovalUXResult{
			Decision:  pref,
			Persisted: true,
			Evidence:  fmt.Sprintf("approval_preference_applied: %s for category %s", pref, prompt.Category),
			AuditRecord: ApprovalAuditRecord{
				Timestamp: time.Now(),
				Command:   prompt.Command,
				Category:  string(prompt.Category),
				Decision:  string(pref),
				Context:   "preference",
			},
		}
	}

	decision := onPrompt(prompt)

	result := ApprovalUXResult{
		Decision: decision,
		Evidence: fmt.Sprintf("approval_prompt_decision: %s", decision),
		AuditRecord: ApprovalAuditRecord{
			Timestamp: time.Now(),
			Command:   prompt.Command,
			Category:  string(prompt.Category),
			Decision:  string(decision),
			Context:   "interactive",
		},
	}

	if decision == UXDecisionAlways {
		s.mu.Lock()
		s.preferences[string(prompt.Category)] = UXDecisionYes
		s.mu.Unlock()
		result.Persisted = true
		result.Decision = UXDecisionYes
		result.Evidence = fmt.Sprintf("approval_always_persisted: category %s", prompt.Category)
	}

	s.mu.Lock()
	s.auditLog = append(s.auditLog, result.AuditRecord)
	s.mu.Unlock()

	return result
}

func (s *ApprovalSession) GetPreferenceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.preferences)
}

func (s *ApprovalSession) GetAuditLog() []ApprovalAuditRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log := make([]ApprovalAuditRecord, len(s.auditLog))
	copy(log, s.auditLog)
	return log
}

func (s *ApprovalSession) IsInteractive() bool {
	return s.interactive
}

func BuildApprovalPrompt(command string, category BlocklistCategory) ApprovalPrompt {
	riskLevel := "medium"
	switch category {
	case BlocklistDestructive:
		riskLevel = "high"
	case BlocklistPrivilege:
		riskLevel = "high"
	case BlocklistDataExfil:
		riskLevel = "high"
	case BlocklistNetwork:
		riskLevel = "medium"
	case BlocklistCryptoMine:
		riskLevel = "medium"
	}

	return ApprovalPrompt{
		Command:       command,
		Category:      category,
		AffectedPaths: extractPaths(command),
		RiskLevel:     riskLevel,
	}
}

func extractPaths(command string) []string {
	var paths []string
	words := strings.Fields(command)
	for _, w := range words {
		if strings.HasPrefix(w, "/") || strings.HasPrefix(w, "./") || strings.HasPrefix(w, "../") {
			paths = append(paths, w)
		}
	}
	return paths
}
