package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/cronjob"

const CronjobToolName = cronjob.CronjobToolName

type CronjobToolConfig = cronjob.CronjobToolConfig
type CronjobTool = cronjob.CronjobTool
type CronjobRunNowRequest = cronjob.CronjobRunNowRequest

func NewCronjobTool(cfg CronjobToolConfig) *CronjobTool {
	return cronjob.NewCronjobTool(cfg)
}
