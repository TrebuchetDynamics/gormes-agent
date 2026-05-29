package redaction

type RedactionConfig struct {
	Enabled bool
}

func DefaultRedactionConfig() RedactionConfig {
	return RedactionConfig{Enabled: true}
}

func (c RedactionConfig) IsEnabled() bool {
	return c.Enabled
}

func (c RedactionConfig) DisabledReason() string {
	if c.Enabled {
		return ""
	}
	return "operator opt-out: redaction.enabled=false in config"
}
