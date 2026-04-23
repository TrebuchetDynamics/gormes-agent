package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/skills"
)

func configuredSkillsRuntime(cfg config.Config) *skills.Runtime {
	return skills.NewRuntimeWithConfig(skills.RuntimeConfig{
		Root:               cfg.SkillsRoot(),
		ExternalDirs:       cfg.Skills.ExternalDirs,
		MaxDocumentBytes:   cfg.Skills.MaxDocumentBytes,
		SelectionCap:       cfg.Skills.SelectionCap,
		UsageLogPath:       cfg.SkillsUsageLogPath(),
		PromptSnapshotPath: cfg.SkillsPromptSnapshotPath(),
	})
}
