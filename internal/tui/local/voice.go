package local

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/preferences"
)

func NewVoiceToggleFunc(cfg config.Config) tui.VoiceToggleFunc {
	return preferences.NewVoiceToggleFunc(cfg)
}

func VoiceRequirementsDetails(cfg config.Config) string {
	return preferences.VoiceRequirementsDetails(cfg)
}
