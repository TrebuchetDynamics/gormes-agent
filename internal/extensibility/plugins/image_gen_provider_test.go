package plugins

import (
	"reflect"
	"testing"
)

func TestImageGenProviderCapabilitiesFromInventory(t *testing.T) {
	inventory := Inventory{
		Capabilities: []CapabilityStatus{
			{Plugin: "spotify", Kind: CapabilityTool, Name: "spotify_playback", State: StateDisabled},
			{Plugin: "zeta", Kind: CapabilityImageGen, Name: "zeta", State: StateDisabled, Evidence: []Evidence{{Code: EvidenceExecutionDisabled}}},
			{Plugin: "alpha", Kind: CapabilityImageGen, Name: "alpha", State: StateDisabled},
		},
	}

	got := ImageGenProviderCapabilities(inventory)
	names := make([]string, 0, len(got))
	for _, row := range got {
		names = append(names, row.Name)
		if row.Kind != CapabilityImageGen {
			t.Fatalf("ImageGenProviderCapabilities returned non-image capability: %+v", row)
		}
	}
	if !reflect.DeepEqual(names, []string{"alpha", "zeta"}) {
		t.Fatalf("image provider names = %v, want sorted alpha/zeta", names)
	}

	got[1].Evidence[0].Code = "mutated"
	if inventory.Capabilities[1].Evidence[0].Code != EvidenceExecutionDisabled {
		t.Fatalf("ImageGenProviderCapabilities did not clone evidence: %+v", inventory.Capabilities[1].Evidence)
	}
}
