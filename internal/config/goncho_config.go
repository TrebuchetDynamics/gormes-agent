package config

import gonchoconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/goncho"

type GonchoCfg = gonchoconfig.GonchoCfg

func defaultGonchoCfg() GonchoCfg {
	return gonchoconfig.DefaultConfig()
}

func normalizeGonchoConfig(cfg *GonchoCfg) error {
	return cfg.NormalizeAndValidate()
}
