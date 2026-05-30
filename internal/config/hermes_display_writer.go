package config

import (
	"fmt"

	configwriter "github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"
)

// SetGormesDisplayPlatformToolProgress persists the Gormes-native gateway
// display.platforms.<platform>.tool_progress override used by /verbose.
func SetGormesDisplayPlatformToolProgress(platform, mode string) error {
	key := normalizeDisplayPlatformKey(platform)
	if key == "" {
		return fmt.Errorf("config: empty display platform")
	}
	normalizedMode, ok := normalizeHermesToolProgressMode(mode)
	if !ok {
		return fmt.Errorf("config: empty tool progress mode")
	}
	return setGormesDisplayPlatformToolProgressAt(ConfigPath(), key, normalizedMode)
}

func setGormesDisplayPlatformToolProgressAt(path, platform, mode string) error {
	return configwriter.SetDisplayPlatformToolProgress(path, platform, mode)
}
