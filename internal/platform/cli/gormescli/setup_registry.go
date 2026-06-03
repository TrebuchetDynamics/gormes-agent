package gormescli

import appsetup "github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"

type SetupSection = appsetup.SetupSection
type SetupRegistry = appsetup.SetupRegistry
type SetupChoice = appsetup.Choice

const (
	SetupModuleGateway   = appsetup.SetupModuleGateway
	SetupModuleNavivox   = appsetup.SetupModuleNavivox
	SetupModuleProviders = appsetup.SetupModuleProviders
	SetupModuleTools     = appsetup.SetupModuleTools
	SetupModuleTTS       = appsetup.SetupModuleTTS
	SetupModuleTUI       = appsetup.SetupModuleTUI
)

func NewSetupRegistry(sections []SetupSection) (*SetupRegistry, error) {
	return appsetup.NewSetupRegistry(sections)
}

func MustSetupRegistry(sections []SetupSection) *SetupRegistry {
	return appsetup.MustSetupRegistry(sections)
}

func SetupSectionOwnership(section string) string {
	return appsetup.SectionOwnership(section)
}

func SetupTTSProviderOptions() []SetupChoice {
	return appsetup.TTSProviderOptions()
}

func SetupTerminalBackendOptions() []SetupChoice {
	return appsetup.TerminalBackendOptions()
}

func SetupTerminalBackendLabel(value string) string {
	return appsetup.TerminalBackendLabel(value)
}

func SetupTTSProviderLabel(value string) string {
	return appsetup.TTSProviderLabel(value)
}

func SetupParsePositiveInt(value string) (int, bool) {
	return appsetup.ParsePositiveInt(value)
}

func SetupParseCompressionThreshold(value string) (float64, bool) {
	return appsetup.ParseCompressionThreshold(value)
}

func SetupIsKnownToolProgressMode(value string) bool {
	return appsetup.IsKnownToolProgressMode(value)
}

func SetupToolProgressModeIndex(value string) int {
	return appsetup.ToolProgressModeIndex(value)
}

func SetupIsKnownSessionResetPolicy(value string) bool {
	return appsetup.IsKnownSessionResetPolicy(value)
}
