package onboarding

import (
	"fmt"
	"strings"
)

const (
	OnboardStepModel     = "model"
	OnboardStepProvider  = "provider"
	OnboardStepAuth      = "auth"
	OnboardStepGateway   = "gateway"
	OnboardStepBrowser   = "browser"
	OnboardStepSkills    = "skills"
	OnboardStepDashboard = "dashboard"

	OnboardStatusConfigured = "configured"
	OnboardStatusMissing    = "missing"
	OnboardStatusAvailable  = "available"
)

type OnboardPlanInput struct {
	Provider       string
	Endpoint       string
	Model          string
	APIKeyPresent  bool
	GatewayTargets []string
	BrowserCDPURL  string
	LocalSkills    int
	BundledSkills  int
}

type OnboardPlan struct {
	Steps []OnboardStep
}

type OnboardStep struct {
	ID          string
	Title       string
	Status      string
	Detail      string
	NextCommand string
	SkipWarning string
}

func BuildOnboardPlan(input OnboardPlanInput) OnboardPlan {
	provider := strings.TrimSpace(input.Provider)
	endpoint := strings.TrimSpace(input.Endpoint)
	model := strings.TrimSpace(input.Model)
	browserCDPURL := strings.TrimSpace(input.BrowserCDPURL)

	modelStep := OnboardStep{
		ID:          OnboardStepModel,
		Title:       "Model",
		Status:      OnboardStatusMissing,
		Detail:      "No default model is configured.",
		NextCommand: "gormes setup model",
		SkipWarning: "Skipping model selection leaves the runtime on built-in defaults or provider-specific fallback behavior.",
	}
	if model != "" {
		modelStep.Status = OnboardStatusConfigured
		modelStep.Detail = fmt.Sprintf("Default model %s will be pre-filled.", model)
	}

	providerStep := OnboardStep{
		ID:          OnboardStepProvider,
		Title:       "Provider",
		Status:      OnboardStatusMissing,
		Detail:      "No provider endpoint is configured.",
		NextCommand: "gormes setup provider",
		SkipWarning: "Skipping provider setup means agents cannot make provider-backed turns until endpoint and model settings are added.",
	}
	if provider != "" || endpoint != "" {
		providerStep.Status = OnboardStatusConfigured
		providerStep.Detail = fmt.Sprintf("Provider %s at %s will be pre-filled.", providerLabel(provider), endpointLabel(endpoint))
	}

	authStep := OnboardStep{
		ID:          OnboardStepAuth,
		Title:       "Auth",
		Status:      OnboardStatusMissing,
		Detail:      "No provider credential was found.",
		NextCommand: "gormes auth add <provider>",
		SkipWarning: "Skipping auth leaves provider calls unavailable until a credential is stored or exported.",
	}
	if input.APIKeyPresent {
		authStep.Status = OnboardStatusConfigured
		authStep.Detail = "Provider credential present; secret value is not shown."
	}

	gatewayStep := OnboardStep{
		ID:          OnboardStepGateway,
		Title:       "Gateway",
		Status:      OnboardStatusMissing,
		Detail:      "No messaging platform is configured.",
		NextCommand: "gormes setup gateway",
		SkipWarning: "Skipping gateway setup keeps Telegram, Discord, Slack, and other messaging adapters disabled.",
	}
	if len(input.GatewayTargets) > 0 {
		gatewayStep.Status = OnboardStatusConfigured
		gatewayStep.Detail = "Configured messaging targets: " + strings.Join(input.GatewayTargets, ", ")
	}

	browserStep := OnboardStep{
		ID:          OnboardStepBrowser,
		Title:       "Browser/CDP",
		Status:      OnboardStatusMissing,
		Detail:      "No browser/CDP endpoint is configured.",
		NextCommand: "gormes doctor --offline",
		SkipWarning: "Skipping browser checks leaves browser automation unavailable until a local or cloud browser backend is configured.",
	}
	if browserCDPURL != "" {
		browserStep.Status = OnboardStatusConfigured
		browserStep.Detail = "Browser/CDP endpoint will be pre-filled: " + browserCDPURL
	}

	skillsStep := OnboardStep{
		ID:          OnboardStepSkills,
		Title:       "Skills",
		Status:      OnboardStatusAvailable,
		Detail:      fmt.Sprintf("Skill discovery sees %d local and %d bundled skills.", input.LocalSkills, input.BundledSkills),
		NextCommand: "gormes skills list",
		SkipWarning: "Skipping skill discovery means the runtime will still use configured skills, but the operator has not reviewed what is installed.",
	}

	dashboardStep := OnboardStep{
		ID:          OnboardStepDashboard,
		Title:       "Dashboard",
		Status:      OnboardStatusAvailable,
		Detail:      "Dashboard launch is available as a manual step.",
		NextCommand: "gormes dashboard",
		SkipWarning: "Skipping dashboard launch keeps onboarding in the terminal; runtime state can still be inspected with CLI commands.",
	}

	return OnboardPlan{Steps: []OnboardStep{
		modelStep,
		providerStep,
		authStep,
		gatewayStep,
		browserStep,
		skillsStep,
		dashboardStep,
	}}
}

func (p OnboardPlan) Step(id string) (OnboardStep, bool) {
	for _, step := range p.Steps {
		if step.ID == id {
			return step, true
		}
	}
	return OnboardStep{}, false
}

func providerLabel(provider string) string {
	if provider == "" {
		return "custom"
	}
	return provider
}

func endpointLabel(endpoint string) string {
	if endpoint == "" {
		return "(manual endpoint)"
	}
	return endpoint
}
