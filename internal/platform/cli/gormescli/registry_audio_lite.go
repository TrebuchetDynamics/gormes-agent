//go:build gormes_lite

package gormescli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func registerAudioTools(_ *tools.Registry, _ config.Config) {}

func audioToolsEnabled() bool { return false }
