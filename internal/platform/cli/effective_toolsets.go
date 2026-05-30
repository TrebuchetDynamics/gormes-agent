package cli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const PlatformToolsetIssueDuplicateToolsetKey = toolsets.PlatformToolsetIssueDuplicateToolsetKey

// EffectiveToolsetOption is one deterministic row for future setup/tools
// pickers.
type EffectiveToolsetOption = toolsets.EffectiveToolsetOption

// EffectiveToolsetReport records the picker rows plus degraded-mode evidence
// for duplicate plugin/built-in declarations.
type EffectiveToolsetReport = toolsets.EffectiveToolsetReport

func EffectiveToolsetPickerOptions(inventory plugins.Inventory) (EffectiveToolsetReport, error) {
	return toolsets.EffectiveToolsetPickerOptions(inventory)
}

func EffectiveToolsetPickerOptionsFromManifest(manifest tools.UpstreamToolParityManifest, inventory plugins.Inventory) EffectiveToolsetReport {
	return toolsets.EffectiveToolsetPickerOptionsFromManifest(manifest, inventory)
}
