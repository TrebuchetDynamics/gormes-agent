package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SetHermesDisplayPlatformToolProgress persists the Hermes gateway
// display.platforms.<platform>.tool_progress override used by /verbose.
func SetHermesDisplayPlatformToolProgress(platform, mode string) error {
	key := normalizeDisplayPlatformKey(platform)
	if key == "" {
		return fmt.Errorf("config: empty display platform")
	}
	normalizedMode, ok := normalizeHermesToolProgressMode(mode)
	if !ok {
		return fmt.Errorf("config: empty tool progress mode")
	}
	return setHermesDisplayPlatformToolProgressAt(hermesConfigPath(), key, normalizedMode)
}

func setHermesDisplayPlatformToolProgressAt(path, platform, mode string) error {
	if path == "" {
		return fmt.Errorf("config: Hermes config path unavailable")
	}
	doc := map[string]any{}
	body, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	if len(body) > 0 {
		if err := yaml.Unmarshal(body, &doc); err != nil {
			return fmt.Errorf("config: parse %s: %w", path, err)
		}
	}
	display := ensureYAMLMap(doc, "display")
	platforms := ensureYAMLMap(display, "platforms")
	platformCfg := ensureYAMLMap(platforms, platform)
	platformCfg["tool_progress"] = mode
	return writeYAMLAtomic(path, doc)
}

func ensureYAMLMap(parent map[string]any, key string) map[string]any {
	if child, ok := parent[key].(map[string]any); ok {
		return child
	}
	child := map[string]any{}
	parent[key] = child
	return child
}

func writeYAMLAtomic(path string, doc map[string]any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}
	body, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("config: marshal yaml: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config.yaml.*")
	if err != nil {
		return fmt.Errorf("config: tempfile: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("config: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("config: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config: close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config: rename temp -> %s: %w", path, err)
	}
	return nil
}
