package tools

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/budget"
)

func TestBudgetFacadeContractsStayAliased(t *testing.T) {
	cfg := ToolResultBudgetConfig{
		OutputDir:       t.TempDir(),
		TextBudgetBytes: 8,
		PreviewBytes:    4,
	}
	var budgetConfig budget.ToolResultBudgetConfig = cfg

	text, evidence, err := FormatToolResult(budgetConfig, []byte(strings.Repeat("x", 16)), "text/plain")
	if err != nil {
		t.Fatalf("FormatToolResult: %v", err)
	}
	if text == "" {
		t.Fatal("expected bounded pointer text")
	}

	var budgetEvidence budget.ToolResultEvidence = evidence
	if budgetEvidence.Code != ToolResultEvidenceTruncated {
		t.Fatalf("evidence code = %q, want %q", budgetEvidence.Code, ToolResultEvidenceTruncated)
	}
}
