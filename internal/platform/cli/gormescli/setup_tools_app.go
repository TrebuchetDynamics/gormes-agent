package gormescli

import (
	appsetup "github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets"
)

type SetupToolOption = appsetup.ToolOption
type SetupInvalidToolSelectionError = appsetup.InvalidToolSelectionError

func SetupToolOptions() ([]SetupToolOption, error) {
	return appsetup.ToolOptions()
}

func LoadSetupToolsConfig(path string) (map[string]any, toolsets.PlatformToolsetConfig, error) {
	return appsetup.LoadToolsConfig(path)
}

func WriteSetupToolsConfig(path string, doc map[string]any) error {
	return appsetup.WriteToolsConfig(path, doc)
}

func ParseSetupToolSelection(input string, options []SetupToolOption, current []string) ([]string, error) {
	return appsetup.ParseToolSelection(input, options, current)
}

func SetupToolsProviderRows(selected []string) []appsetup.ToolsProviderRow {
	return appsetup.ToolsProviderRows(selected)
}
