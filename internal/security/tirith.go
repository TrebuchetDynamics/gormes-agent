// Package security provides security finding ingestion and guard decision
// composition for Gormes. It mirrors the Hermes Tirith subsystem:
// load findings from an external source, classify by severity, and expose an
// allow/deny decision that gateway/cron/CLI callers can query before executing
// dangerous commands.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// TirithSeverity represents the severity of a security finding.
type TirithSeverity string

const (
	SeverityCritical TirithSeverity = "critical"
	SeverityHigh     TirithSeverity = "high"
	SeverityMedium   TirithSeverity = "medium"
	SeverityLow      TirithSeverity = "low"
	SeverityInfo     TirithSeverity = "info"
)

// severityOrder maps severities to numeric values for sorting (lower = more severe).
var severityOrder = map[TirithSeverity]int{
	SeverityCritical: 0,
	SeverityHigh:     1,
	SeverityMedium:   2,
	SeverityLow:      3,
	SeverityInfo:     4,
}

// TirithFinding is a single security finding from an external scanner such as
// semgrep, CodeQL, or a custom policy engine.
type TirithFinding struct {
	RuleID   string         `json:"rule_id"`
	Severity TirithSeverity `json:"severity"`
	Message  string         `json:"message"`
	File     string         `json:"file"`
}

// TirithEvidence is the typed result of a Tirith decision query. Callers
// inspect Allow to decide whether to proceed and Reason/EvidenceType for
// audit logging.
type TirithEvidence struct {
	Allow        bool   `json:"allow"`
	Reason       string `json:"reason"`
	EvidenceType string `json:"evidence_type"`
}

// tirithPayload is the on-disk JSON structure that TirithClient reads.
type tirithPayload struct {
	Findings []TirithFinding `json:"findings"`
}

// TirithClient loads security findings from a JSON source and provides
// allow/deny decisions based on severity thresholds.
type TirithClient struct {
	findings []TirithFinding
	evidence TirithEvidence
}

// NewTirithClient creates a client by reading the file at sourcePath. If the
// file does not exist, the client returns an allow decision with
// tirith_unavailable evidence. If the file is corrupt, the client returns a
// deny decision with tirith_corrupt_evidence. Otherwise findings are parsed,
// sorted by severity (critical → info), and the decision is computed.
func NewTirithClient(sourcePath string) (*TirithClient, error) {
	c := &TirithClient{}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.evidence = TirithEvidence{
				Allow:        true,
				Reason:       fmt.Sprintf("Tirith source not found: %s", sourcePath),
				EvidenceType: "tirith_unavailable",
			}
			return c, nil
		}
		return nil, fmt.Errorf("tirith: reading source %s: %w", sourcePath, err)
	}

	var payload tirithPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		c.evidence = TirithEvidence{
			Allow:        false,
			Reason:       fmt.Sprintf("Tirith source corrupt: %s", err),
			EvidenceType: "tirith_corrupt_evidence",
		}
		return c, nil
	}

	c.findings = payload.Findings
	sort.Slice(c.findings, func(i, j int) bool {
		return severityOrder[c.findings[i].Severity] < severityOrder[c.findings[j].Severity]
	})

	c.evidence = computeDecision(c.findings)
	return c, nil
}

// Findings returns the parsed findings sorted by severity (critical first).
func (c *TirithClient) Findings() []TirithFinding {
	return c.findings
}

// Decision returns the allow/deny evidence for this client's findings.
func (c *TirithClient) Decision() TirithEvidence {
	return c.evidence
}

// computeDecision determines whether findings warrant denial. Findings with
// severity critical or high return deny; medium/low/info return allow.
func computeDecision(findings []TirithFinding) TirithEvidence {
	if len(findings) == 0 {
		return TirithEvidence{
			Allow:        true,
			Reason:       "No Tirith findings",
			EvidenceType: "tirith_no_findings",
		}
	}

	for _, f := range findings {
		if f.Severity == SeverityCritical || f.Severity == SeverityHigh {
			return TirithEvidence{
				Allow:        false,
				Reason:       fmt.Sprintf("Tirith deny: rule %s (%s) - %s", f.RuleID, f.Severity, f.Message),
				EvidenceType: "tirith_deny",
			}
		}
	}

	return TirithEvidence{
		Allow:        true,
		Reason:       "Tirith findings are medium or lower — no block",
		EvidenceType: "tirith_allow",
	}
}
