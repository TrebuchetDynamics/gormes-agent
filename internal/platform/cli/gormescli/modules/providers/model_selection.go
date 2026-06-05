package providers

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/modelcontract"

// ModelCommandSeams is the shared contract for commands that drive the
// provider/model picker and persist the selected model-like value.
type ModelCommandSeams = modelcontract.Seams

func normalizedModelChooser(seams ModelCommandSeams) func(provider string, current string) (string, error) {
	return modelcontract.NormalizedModelChooser(seams)
}
