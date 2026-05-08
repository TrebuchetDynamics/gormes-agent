package plugins

import (
	"slices"
	"testing"
)

func TestTeamsPipelinePluginMetadata(t *testing.T) {
	t.Run("LoadsManifest", testTeamsPipelinePluginMetadataLoadsManifest)
	t.Run("CLICommandCapability", testTeamsPipelinePluginMetadataCLICommandCapability)
	t.Run("RuntimeEvidence", testTeamsPipelinePluginMetadataRuntimeEvidence)
	t.Run("NoToolInventory", testTeamsPipelinePluginMetadataNoToolInventory)
}

func teamsPipelinePluginFixture(t *testing.T) string {
	t.Helper()
	return writePluginFixture(t, "teams_pipeline", map[string]string{
		"plugin.yaml": TeamsPipelinePluginYAML,
		"__init__.py": TeamsPipelinePluginInitPy,
	})
}

func loadTeamsPipelineForTest(t *testing.T) PluginStatus {
	t.Helper()
	status := LoadTeamsPipeline(teamsPipelinePluginFixture(t), LoadOptions{
		Source:               SourceBundled,
		CurrentGormesVersion: "1.0.0",
		EnvLookup:            func(string) bool { return false },
		AuthLookup:           func(string) bool { return false },
	})
	if status.RuntimeCodeExecuted {
		t.Fatal("Teams pipeline plugin metadata load executed runtime code")
	}
	return status
}

func testTeamsPipelinePluginMetadataLoadsManifest(t *testing.T) {
	status := loadTeamsPipelineForTest(t)

	if status.State != StateDisabled {
		t.Fatalf("state = %q, want disabled; evidence=%+v", status.State, status.Evidence)
	}
	if status.Manifest.Name != "teams_pipeline" {
		t.Fatalf("manifest name = %q, want teams_pipeline", status.Manifest.Name)
	}
	if status.Manifest.Version != "0.1.0" {
		t.Fatalf("manifest version = %q, want 0.1.0", status.Manifest.Version)
	}
	if status.Manifest.Author != "NousResearch" {
		t.Fatalf("manifest author = %q, want NousResearch", status.Manifest.Author)
	}
	if !slices.Equal(status.Manifest.Platforms, []string{"linux", "macos", "windows"}) {
		t.Fatalf("manifest platforms = %#v, want [linux macos windows]", status.Manifest.Platforms)
	}
}

func testTeamsPipelinePluginMetadataCLICommandCapability(t *testing.T) {
	status := loadTeamsPipelineForTest(t)

	if got := capabilityNames(status.Capabilities, CapabilityCLICommand); !slices.Equal(got, []string{"teams-pipeline"}) {
		t.Fatalf("cli command capabilities = %#v, want [teams-pipeline]", got)
	}
	capability := findCapability(status.Capabilities, CapabilityCLICommand, "teams-pipeline")
	if capability == nil {
		t.Fatal("teams-pipeline CLI capability missing")
	}
	if capability.State != StateDisabled {
		t.Fatalf("teams-pipeline state = %q, want disabled", capability.State)
	}
	assertEvidence(t, capability.Evidence, EvidenceExecutionDisabled, "runtime")
	assertEvidence(t, capability.Evidence, EvidenceTeamsPipelineRuntimeUnavailable, "teams_pipeline")
}

func testTeamsPipelinePluginMetadataRuntimeEvidence(t *testing.T) {
	status := loadTeamsPipelineForTest(t)

	assertEvidence(t, status.Evidence, EvidenceExecutionDisabled, "runtime")
	assertEvidence(t, status.Evidence, EvidenceTeamsPipelineRuntimeUnavailable, "teams_pipeline")
	assertEvidence(t, status.Evidence, EvidenceGraphCredentialsRequired, "MSGRAPH_CLIENT_ID")
	assertEvidence(t, status.Evidence, EvidenceTeamsDeliveryTargetRequired, "teams_delivery")
}

func testTeamsPipelinePluginMetadataNoToolInventory(t *testing.T) {
	status := loadTeamsPipelineForTest(t)

	if len(status.Tools) != 0 {
		t.Fatalf("tools = %#v, want none", toolMetadataNames(status.Tools))
	}
	if got := capabilityNames(status.Capabilities, CapabilityTool); len(got) != 0 {
		t.Fatalf("tool capabilities = %#v, want none", got)
	}
}
