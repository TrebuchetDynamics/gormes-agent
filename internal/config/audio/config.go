package audio

type STTCfg struct {
	Enabled  bool           `toml:"enabled" yaml:"enabled"`
	Provider string         `toml:"provider" yaml:"provider"`
	Local    STTLocalCfg    `toml:"local" yaml:"local"`
	OpenAI   STTProviderCfg `toml:"openai" yaml:"openai"`
}

type STTLocalCfg struct {
	Model    string `toml:"model" yaml:"model"`
	Language string `toml:"language" yaml:"language"`
}

type STTProviderCfg struct {
	Model string `toml:"model" yaml:"model"`
}

type VoiceCfg struct {
	RecordKey string `toml:"record_key" yaml:"record_key"`
}
