package config

import "fmt"

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
	if path == "" {
		return fmt.Errorf("config: Gormes config path unavailable")
	}
	doc, err := readTOMLDoc(path)
	if err != nil {
		return err
	}
	display := ensureTOMLMap(doc, "display")
	platforms := ensureTOMLMap(display, "platforms")
	platformCfg := ensureTOMLMap(platforms, platform)
	platformCfg["tool_progress"] = mode
	return writeTOMLDoc(path, doc)
}

func ensureTOMLMap(parent map[string]any, key string) map[string]any {
	if child, ok := parent[key].(map[string]any); ok {
		return child
	}
	child := map[string]any{}
	parent[key] = child
	return child
}
