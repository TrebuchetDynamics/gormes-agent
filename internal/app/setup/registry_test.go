package setup

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/contractruntime"
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

	contracts, err := contractruntime.NewRegistry([]contractruntime.ModuleContract{
		{Module: progress.ModuleProviders, SetupSections: []contractruntime.SetupSectionSpec{{Name: "provider"}}},
		{Module: progress.ModuleProfiles, SetupSections: []contractruntime.SetupSectionSpec{{Name: "profiles"}}},
	})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if err := setupRegistry.ValidateContracts(contracts); err != nil {
		t.Fatalf("ValidateContracts: %v", err)
	}
}

func TestSetupRegistryCanonicalizesAliasesAndLabels(t *testing.T) {
	setupRegistry := MustSetupRegistry([]SetupSection{
		{Name: "provider", Label: "Provider", Module: progress.ModuleProviders},
		{Name: "gateway", Label: "Messaging Gateway", Module: progress.ModuleGateway},
	})
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: "providers", want: "provider"},
		{input: "messaging platforms", want: "gateway"},
		{input: "slack", want: "gateway"},
		{input: "unknown-section", want: "unknown_section"},
	} {
		if got := setupRegistry.CanonicalSection(tc.input); got != tc.want {
			t.Fatalf("CanonicalSection(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
	if !setupRegistry.KnownSection("gateway") {
		t.Fatal("KnownSection(gateway) = false, want true")
	}
	if setupRegistry.KnownSection("slack") {
		t.Fatal("KnownSection(slack) = true before canonicalization, want false")
	}
	if got := setupRegistry.SectionLabel("gateway"); got != "Messaging Gateway" {
		t.Fatalf("SectionLabel(gateway) = %q, want Messaging Gateway", got)
	}
	if got := setupRegistry.SectionLabel("missing"); got != "missing" {
		t.Fatalf("SectionLabel(missing) = %q, want missing", got)
	}
}

func TestSetupRegistryRendersSectionListsAndOwnership(t *testing.T) {
	setupRegistry := MustSetupRegistry([]SetupSection{
		{Name: "provider", Label: "Provider", Module: progress.ModuleProviders},
		{Name: "model", Label: "Model", Module: progress.ModuleProviders},
		{Name: "gateway", Label: "Messaging Gateway", Module: progress.ModuleGateway},
	})
	if got := setupRegistry.SectionPipeList(0, 2); got != "provider|model" {
		t.Fatalf("SectionPipeList(0,2) = %q, want provider|model", got)
	}
	if got := setupRegistry.SectionPipeList(-2, 99); got != "provider|model|gateway" {
		t.Fatalf("SectionPipeList(-2,99) = %q, want provider|model|gateway", got)
	}
	if got := setupRegistry.SectionPipeList(3, 1); got != "" {
		t.Fatalf("SectionPipeList(3,1) = %q, want empty", got)
	}
	if got := setupRegistry.SectionList(); got != "provider|model|gateway" {
		t.Fatalf("SectionList() = %q, want provider|model|gateway", got)
	}
	if got := SectionOwnership("model"); got != "hermes_owned" {
		t.Fatalf("SectionOwnership(model) = %q, want hermes_owned", got)
	}
	if got := SectionOwnership("workspace"); got != "gormes_owned_extension" {
		t.Fatalf("SectionOwnership(workspace) = %q, want gormes_owned_extension", got)
	}
	if got := SectionOwnership("missing"); got != "unknown" {
		t.Fatalf("SectionOwnership(missing) = %q, want unknown", got)
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
	contracts, err := contractruntime.NewRegistry([]contractruntime.ModuleContract{{Module: progress.ModuleGateway, SetupSections: []contractruntime.SetupSectionSpec{{Name: "profiles"}}}})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if err := setupRegistry.ValidateContracts(contracts); err == nil || !strings.Contains(err.Error(), `does not match contract owner`) {
		t.Fatalf("ValidateContracts mismatch error = %v", err)
	}
}
