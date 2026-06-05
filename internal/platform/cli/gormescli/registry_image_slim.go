//go:build slim

package gormescli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func registerImageGenerationTool(reg *tools.Registry, _ config.Config) {
	reg.MustRegister(tools.NewImageGenTool(nil))
}

func registerVideoAnalyzeTool(reg *tools.Registry, _ config.Config) {
	reg.MustRegister(tools.NewVideoAnalyzeTool(tools.VideoAnalyzeConfig{}))
}
