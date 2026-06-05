package auxiliary

import "strings"

type AuxiliaryCfg struct {
	Curator AuxiliaryTaskCfg `toml:"curator" yaml:"curator"`
	Vision  AuxiliaryTaskCfg `toml:"vision" yaml:"vision"`
}

type CuratorCfg struct {
	Auxiliary AuxiliaryTaskCfg `toml:"auxiliary" yaml:"auxiliary"`
}

type AuxiliaryTaskCfg struct {
	Provider  string         `toml:"provider" yaml:"provider"`
	Model     string         `toml:"model" yaml:"model"`
	BaseURL   string         `toml:"base_url" yaml:"base_url"`
	APIKey    string         `toml:"api_key" yaml:"api_key"`
	Timeout   int            `toml:"timeout" yaml:"timeout"`
	ExtraBody map[string]any `toml:"extra_body" yaml:"extra_body"`
}

func NormalizeTask(task *AuxiliaryTaskCfg, defaultCurator bool) {
	task.Provider = strings.TrimSpace(task.Provider)
	task.Model = strings.TrimSpace(task.Model)
	task.BaseURL = strings.TrimSpace(task.BaseURL)
	task.APIKey = strings.TrimSpace(task.APIKey)
	if defaultCurator {
		if task.Provider == "" {
			task.Provider = "auto"
		}
		if task.Timeout == 0 {
			task.Timeout = 600
		}
		if task.ExtraBody == nil {
			task.ExtraBody = map[string]any{}
		}
	}
}
