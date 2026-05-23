package main

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func newTUISkinConfigFunc(cfg config.Config) tui.SkinConfigFunc {
	current := normalizeTUISkinName(cfg.TUI.Theme)
	return func(req tui.SkinConfigRequest) (tui.SkinConfigResult, error) {
		requested := strings.TrimSpace(req.Name)
		if requested == "" {
			if loaded, err := config.Load(nil); err == nil {
				current = normalizeTUISkinName(loaded.TUI.Theme)
			}
			return tui.SkinConfigResult{Name: current}, nil
		}
		skin, ok := tui.ResolveBuiltinSkin(requested)
		if !ok {
			return tui.SkinConfigResult{}, fmt.Errorf("unknown skin %q", requested)
		}
		if err := config.WriteTOMLValue(config.ConfigPath(), "tui.theme", skin.Name); err != nil {
			return tui.SkinConfigResult{}, err
		}
		current = skin.Name
		return tui.SkinConfigResult{Name: current}, nil
	}
}

func normalizeTUISkinName(name string) string {
	if skin, ok := tui.ResolveBuiltinSkin(name); ok {
		return skin.Name
	}
	return "default"
}
