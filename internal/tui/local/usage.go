package local

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/preferences"
)

func NewAccountUsageFunc(cfg config.Config) tui.AccountUsageFunc {
	return preferences.NewAccountUsageFunc(cfg)
}
