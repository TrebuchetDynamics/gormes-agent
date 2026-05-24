package gormescli

import (
	"strings"
	"testing"
)

func TestDefaultContractsValidate(t *testing.T) {
	registry, err := NewRegistry(DefaultContracts())
	if err != nil {
		t.Fatalf("NewRegistry(DefaultContracts()) error = %v", err)
	}
	if len(registry.Modules()) == 0 {
		t.Fatal("default CLI contract registry has no modules")
	}
	if _, ok := registry.CommandOwner("profile use"); !ok {
		t.Fatal("profile use must be owned by the profiles module")
	}
	if _, ok := registry.SetupSectionOwner("profiles"); !ok {
		t.Fatal("setup profiles must be owned by the profiles module")
	}
	if _, ok := registry.SlashCommandOwner("profile"); !ok {
		t.Fatal("/profile must be owned by the profiles module")
	}
}

func TestRegistryRejectsInvalidModule(t *testing.T) {
	_, err := NewRegistry([]ModuleContract{{
		Module: "cmd",
		Commands: []CommandSpec{
			{Path: "cmd-only"},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), `invalid module "cmd"`) {
		t.Fatalf("NewRegistry invalid module error = %v", err)
	}
}

func TestRegistryRejectsAmbiguousCommandOwnership(t *testing.T) {
	_, err := NewRegistry([]ModuleContract{
		{
			Module: "cli",
			Commands: []CommandSpec{
				{Path: "profile", IncludeDescendants: true},
			},
		},
		{
			Module: "profiles",
			Commands: []CommandSpec{
				{Path: "profile use"},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), `command ownership overlap`) {
		t.Fatalf("NewRegistry overlap error = %v", err)
	}
}

func TestCommandManifestRejectsUnownedCobraPath(t *testing.T) {
	registry, err := NewRegistry(DefaultContracts())
	if err != nil {
		t.Fatalf("NewRegistry(DefaultContracts()) error = %v", err)
	}
	_, err = registry.CommandManifest([]string{"profile", "profile use", "not-owned"})
	if err == nil || !strings.Contains(err.Error(), `no module owner for command path "not-owned"`) {
		t.Fatalf("CommandManifest unowned error = %v", err)
	}
}

func TestSetupAndSlashValidationRejectMissingOwners(t *testing.T) {
	registry, err := NewRegistry(DefaultContracts())
	if err != nil {
		t.Fatalf("NewRegistry(DefaultContracts()) error = %v", err)
	}
	if err := registry.ValidateSetupSections([]string{"profiles", "missing-section"}); err == nil || !strings.Contains(err.Error(), `no module owner for setup section "missing-section"`) {
		t.Fatalf("ValidateSetupSections missing owner error = %v", err)
	}
	if err := registry.ValidateSlashCommands([]string{"profile", "missing-slash"}); err == nil || !strings.Contains(err.Error(), `no module owner for slash command "missing-slash"`) {
		t.Fatalf("ValidateSlashCommands missing owner error = %v", err)
	}
}
