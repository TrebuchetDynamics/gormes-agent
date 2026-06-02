package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func newTUIToolsConfigureFunc() tui.ToolsConfigureFunc {
	return func(req tui.ToolsConfigureRequest) (tui.ToolsConfigureResult, error) {
		return configureTUITools(req)
	}
}

func configureTUITools(req tui.ToolsConfigureRequest) (tui.ToolsConfigureResult, error) {
	action := normalizeSetupChoice(req.Action)
	if action != "enable" && action != "disable" {
		return tui.ToolsConfigureResult{}, fmt.Errorf("unsupported action %q", req.Action)
	}
	doc, toolCfg, err := loadSetupToolsConfig(config.ConfigPath())
	if err != nil {
		return tui.ToolsConfigureResult{}, err
	}
	status, err := toolCfg.PlatformStatus("cli")
	if err != nil {
		return tui.ToolsConfigureResult{}, err
	}
	known, err := knownTUIToolsets()
	if err != nil {
		return tui.ToolsConfigureResult{}, err
	}
	current := stringSet(status.RuntimeToolsets)
	selected := append([]string(nil), status.RuntimeToolsets...)
	var result tui.ToolsConfigureResult
	for _, raw := range req.Names {
		display := strings.TrimSpace(raw)
		name := normalizeSetupChoice(raw)
		if name == "" {
			continue
		}
		persistName := name
		if server, _, ok := strings.Cut(name, ":"); ok {
			server = normalizeSetupChoice(server)
			if server == "" || !toolCfg.MCPServers[server].Enabled {
				result.MissingServers = appendUniqueString(result.MissingServers, server)
				continue
			}
			persistName = server
		} else if !known[persistName] && !current[persistName] && !toolCfg.MCPServers[persistName].Enabled {
			result.Unknown = appendUniqueString(result.Unknown, displayOrName(display, persistName))
			continue
		}
		if action == "enable" {
			if !current[persistName] {
				current[persistName] = true
				selected = append(selected, persistName)
				result.Changed = appendUniqueString(result.Changed, displayOrName(display, persistName))
			}
			continue
		}
		if current[persistName] {
			current[persistName] = false
			result.Changed = appendUniqueString(result.Changed, displayOrName(display, persistName))
		}
	}
	if len(result.Changed) == 0 {
		return result, nil
	}
	persisted := make([]string, 0, len(selected))
	seen := map[string]struct{}{}
	for _, name := range selected {
		name = normalizeSetupChoice(name)
		if name == "" || !current[name] {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		persisted = append(persisted, name)
	}
	sort.Strings(persisted)
	if _, err := toolCfg.SavePlatformSelection("cli", persisted); err != nil {
		return tui.ToolsConfigureResult{}, err
	}
	doc["platform_toolsets"] = toolCfg.PlatformToolsets
	if err := writeSetupToolsConfig(config.ConfigPath(), doc); err != nil {
		return tui.ToolsConfigureResult{}, err
	}
	result.Reset = true
	return result, nil
}

func knownTUIToolsets() (map[string]bool, error) {
	options, err := setupToolOptions()
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(options))
	for _, option := range options {
		known[normalizeSetupChoice(option.Key)] = true
	}
	return known, nil
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func displayOrName(display, name string) string {
	if strings.TrimSpace(display) != "" {
		return strings.TrimSpace(display)
	}
	return name
}
