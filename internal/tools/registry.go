package tools

// RegisterHomeAssistantTools adds the Hermes Home Assistant control tools when
// HASS_TOKEN is configured. Missing credentials leave the registry unchanged.
func RegisterHomeAssistantTools(r *Registry, cfg HomeAssistantConfig) {
	if r == nil {
		return
	}
	for _, tool := range NewHomeAssistantTools(cfg) {
		r.MustRegister(tool)
	}
}
