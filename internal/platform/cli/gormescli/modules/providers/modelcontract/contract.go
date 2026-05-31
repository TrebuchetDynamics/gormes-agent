package modelcontract

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// Seams is the shared contract for commands that drive the provider/model
// picker and persist the selected model-like value.
type Seams struct {
	IsTTY            func() bool
	LoadCurrent      func() (cli.ProviderModel, error)
	ListProviders    func() ([]cli.ProviderMenuEntry, error)
	ChooseProvider   func(entries []cli.ProviderMenuEntry, defaultIndex int) (int, error)
	ChooseModel      func(provider string, current string) (string, error)
	PersistSelection func(cli.Selection) error
}

func NormalizedModelChooser(seams Seams) func(provider string, current string) (string, error) {
	if seams.ChooseModel == nil {
		return nil
	}
	return func(provider string, current string) (string, error) {
		model, err := seams.ChooseModel(provider, current)
		if err != nil {
			return "", err
		}
		return llm.NormalizeProviderModelID(provider, model), nil
	}
}
