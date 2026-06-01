package toolsets

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets/picker"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const PlatformToolsetIssueDuplicateToolsetKey = picker.PlatformToolsetIssueDuplicateToolsetKey

// EffectiveToolsetOption is one deterministic row for future setup/tools
// pickers.
type EffectiveToolsetOption = picker.EffectiveToolsetOption

// EffectiveToolsetReport records the picker rows plus degraded-mode evidence
// for duplicate plugin/built-in declarations.
type EffectiveToolsetReport = picker.EffectiveToolsetReport

func EffectiveToolsetPickerOptions(inventory plugins.Inventory) (EffectiveToolsetReport, error) {
	return picker.EffectiveToolsetPickerOptions(inventory)
}

func EffectiveToolsetPickerOptionsFromManifest(manifest tools.UpstreamToolParityManifest, inventory plugins.Inventory) EffectiveToolsetReport {
	return picker.EffectiveToolsetPickerOptionsFromManifest(manifest, inventory)
}
