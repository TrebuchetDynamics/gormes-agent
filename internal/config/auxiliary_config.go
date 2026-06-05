package config

import auxiliaryconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/auxiliary"

type AuxiliaryCfg = auxiliaryconfig.AuxiliaryCfg
type CuratorCfg = auxiliaryconfig.CuratorCfg
type AuxiliaryTaskCfg = auxiliaryconfig.AuxiliaryTaskCfg

func normalizeAuxiliaryTask(task *AuxiliaryTaskCfg, defaultCurator bool) {
	auxiliaryconfig.NormalizeTask(task, defaultCurator)
}
