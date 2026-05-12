package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/budget"

const (
	ToolResultEvidenceUnderBudget       = budget.ToolResultEvidenceUnderBudget
	ToolResultEvidenceTruncated         = budget.ToolResultEvidenceTruncated
	ToolResultEvidencePersisted         = budget.ToolResultEvidencePersisted
	ToolResultEvidencePersistenceFailed = budget.ToolResultEvidencePersistenceFailed
)

type ToolResultBudgetConfig = budget.ToolResultBudgetConfig
type ToolResultEvidence = budget.ToolResultEvidence

func FormatToolResult(cfg ToolResultBudgetConfig, raw []byte, mediaType string) (string, ToolResultEvidence, error) {
	return budget.FormatToolResult(cfg, raw, mediaType)
}
