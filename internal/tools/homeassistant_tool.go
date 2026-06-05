package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/homeassistant"

const (
	HomeAssistantEvidenceOK               = homeassistant.HomeAssistantEvidenceOK
	HomeAssistantEvidenceUnavailable      = homeassistant.HomeAssistantEvidenceUnavailable
	HomeAssistantEvidenceValidationFailed = homeassistant.HomeAssistantEvidenceValidationFailed
)

type HomeAssistantConfig = homeassistant.HomeAssistantConfig
type HomeAssistantClient = homeassistant.HomeAssistantClient
type HomeAssistantState = homeassistant.HomeAssistantState
type HomeAssistantServiceDomain = homeassistant.HomeAssistantServiceDomain
type HomeAssistantService = homeassistant.HomeAssistantService
type HomeAssistantServiceField = homeassistant.HomeAssistantServiceField

func NewHomeAssistantTools(cfg HomeAssistantConfig) []Tool {
	return homeassistant.NewHomeAssistantTools(cfg)
}
