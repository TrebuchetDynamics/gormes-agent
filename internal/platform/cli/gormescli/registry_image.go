//go:build !slim

package gormescli

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func registerImageGenerationTool(reg *tools.Registry, cfg config.Config) {
	providers := map[string]tools.ImageGenerator{
		"fal": tools.NewFALGenImageProvider(imageGenConfigString(cfg.ImageGen, "fal_api_key")),
	}
	reg.MustRegister(tools.NewImageGenTool(tools.NewImageGenRunner(tools.ImageGenConfig{
		Disabled:     imageGenConfigBool(cfg.ImageGen, "disabled"),
		Provider:     imageGenConfigString(cfg.ImageGen, "provider"),
		DefaultModel: imageGenConfigString(cfg.ImageGen, "model"),
	}, providers)))
}

func registerVideoAnalyzeTool(reg *tools.Registry, _ config.Config) {
	reg.MustRegister(tools.NewVideoAnalyzeTool(tools.VideoAnalyzeConfig{
		ProviderFactory: func(_ context.Context) tools.VideoAnalyzeProvider { return nil },
	}))
}

func imageGenConfigString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func imageGenConfigBool(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	value, ok := values[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on", "enabled":
			return true
		}
	}
	return false
}
