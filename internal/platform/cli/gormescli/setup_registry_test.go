package gormescli

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

func TestSetupRegistryPreservesOrderLabelsAndContractOwners(t *testing.T) {
	setupRegistry, err := NewSetupRegistry([]SetupSection{
		{Name: "provider", Label: "Provider", Module: progress.ModuleProviders},
		{Name: "profiles", Label: "Profiles", Module: progress.ModuleProfiles},
	})
	if err != nil {
		t.Fatalf("NewSetupRegistry: %v", err)
	}
	if got := strings.Join(setupRegistry.Names(), "|"); got != "provider|profiles" {
		t.Fatalf("Names = %q, want provider|profiles", got)
	}
	labels := setupRegistry.Labels()
	if labels["profiles"] != "Profiles" {
		t.Fatalf("profiles label = %q, want Profiles", labels["profiles"])
	}
	labels["profiles"] = "mutated"
	if setupRegistry.Labels()["profiles"] != "Profiles" {
		t.Fatal("Labels must return a defensive copy")
	}

	contracts, err := NewRegistry([]ModuleContract{
		{Module: progress.ModuleProviders, SetupSections: []SetupSectionSpec{{Name: "provider"}}},
		{Module: progress.ModuleProfiles, SetupSections: []SetupSectionSpec{{Name: "profiles"}}},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if err := setupRegistry.ValidateContracts(contracts); err != nil {
		t.Fatalf("ValidateContracts: %v", err)
	}
}

func TestSetupRegistryRejectsInvalidModuleAndContractMismatch(t *testing.T) {
	if _, err := NewSetupRegistry([]SetupSection{{Name: "profiles", Label: "Profiles", Module: "cmd"}}); err == nil || !strings.Contains(err.Error(), `invalid module "cmd"`) {
		t.Fatalf("NewSetupRegistry invalid module error = %v", err)
	}
	setupRegistry, err := NewSetupRegistry([]SetupSection{{Name: "profiles", Label: "Profiles", Module: progress.ModuleProfiles}})
	if err != nil {
		t.Fatalf("NewSetupRegistry: %v", err)
	}
	contracts, err := NewRegistry([]ModuleContract{{Module: progress.ModuleGateway, SetupSections: []SetupSectionSpec{{Name: "profiles"}}}})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if err := setupRegistry.ValidateContracts(contracts); err == nil || !strings.Contains(err.Error(), `does not match contract owner`) {
		t.Fatalf("ValidateContracts mismatch error = %v", err)
	}
}
