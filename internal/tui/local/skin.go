package local

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/preferences"
)

func NewSkinConfigFunc(cfg config.Config) tui.SkinConfigFunc {
	return preferences.NewSkinConfigFunc(cfg)
}

func NormalizeSkinName(name string) string {
	return preferences.NormalizeSkinName(name)
}
