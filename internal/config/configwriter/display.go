package configwriter

import "fmt"

// SetDisplayPlatformToolProgress persists display.platforms.<platform>.tool_progress
// in the config document at path. Callers own platform/mode normalization.
func SetDisplayPlatformToolProgress(path, platform, mode string) error {
	if path == "" {
		return fmt.Errorf("config: Gormes config path unavailable")
	}
	doc, err := ReadTOMLDoc(path)
	if err != nil {
		return err
	}
	display := ensureTOMLMap(doc, "display")
	platforms := ensureTOMLMap(display, "platforms")
	platformCfg := ensureTOMLMap(platforms, platform)
	platformCfg["tool_progress"] = mode
	return WriteTOMLDoc(path, doc)
}

func ensureTOMLMap(parent map[string]any, key string) map[string]any {
	if child, ok := parent[key].(map[string]any); ok {
		return child
	}
	child := map[string]any{}
	parent[key] = child
	return child
}
