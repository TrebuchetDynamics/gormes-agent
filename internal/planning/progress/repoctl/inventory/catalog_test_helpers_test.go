package inventory

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/fidelity"
)

func catalogProgressItem(name string, status progress.Status, contractStatus progress.ContractStatus, module, sourceRef, testCommand string) progress.Item {
	return progress.Item{
		Name:           name,
		Priority:       "P1",
		Status:         status,
		ContractStatus: contractStatus,
		Module:         module,
		Contract:       name + " contract.",
		SourceRefs:     []string{sourceRef},
		TestCommands:   []string{testCommand},
	}
}

func catalogFamilyIDs(families []fidelity.CatalogFamilyReport) []string {
	ids := make([]string, 0, len(families))
	for _, family := range families {
		ids = append(ids, family.ID)
	}
	return ids
}

func catalogFamilyByID(t *testing.T, families []fidelity.CatalogFamilyReport, id, label string) fidelity.CatalogFamilyReport {
	t.Helper()
	for _, family := range families {
		if family.ID == id {
			return family
		}
	}
	t.Fatalf("%s family %q missing from %+v", label, id, families)
	return fidelity.CatalogFamilyReport{}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsProgressRow(rows []fidelity.ProgressRowEvidence, want string) bool {
	for _, row := range rows {
		if row.Name == want {
			return true
		}
	}
	return false
}

func containsSourcePair(pairs []fidelity.SourcePairEvidence, want string) bool {
	for _, pair := range pairs {
		if pair.HermesFile == want {
			return true
		}
	}
	return false
}
