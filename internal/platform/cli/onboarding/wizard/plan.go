package wizard

import (
	"fmt"
	"strings"
)

const (
	StepModel     = "model"
	StepProvider  = "provider"
	StepAuth      = "auth"
	StepGateway   = "gateway"
	StepBrowser   = "browser"
	StepSkills    = "skills"
	StepDashboard = "dashboard"

	StatusConfigured = "configured"
	StatusMissing    = "missing"
	StatusAvailable  = "available"
)

type PlanInput struct {
	Provider       string
	Endpoint       string
	Model          string
	APIKeyPresent  bool
	GatewayTargets []string
	BrowserCDPURL  string
	LocalSkills    int
	BundledSkills  int
}

type Plan struct {
	Steps []Step
}

type Step struct {
	ID          string
	Title       string
	Status      string
	Detail      string
	NextCommand string
	SkipWarning string
}

func BuildPlan(input PlanInput) Plan {
	provider := strings.TrimSpace(input.Provider)
	endpoint := strings.TrimSpace(input.Endpoint)
	model := strings.TrimSpace(input.Model)
	browserCDPURL := strings.TrimSpace(input.BrowserCDPURL)

	modelStep := Step{
		ID:          StepModel,
		Title:       "Model",
		Status:      StatusMissing,
		Detail:      "No default model is configured.",
		NextCommand: "gormes setup model",
		SkipWarning: "Skipping model selection leaves the runtime on built-in defaults or provider-specific fallback behavior.",
	}
	if model != "" {
		modelStep.Status = StatusConfigured
		modelStep.Detail = fmt.Sprintf("Default model %s will be pre-filled.", model)
	}

	providerStep := Step{
		ID:          StepProvider,
		Title:       "Provider",
		Status:      StatusMissing,
		Detail:      "No provider endpoint is configured.",
		NextCommand: "gormes setup provider",
		SkipWarning: "Skipping provider setup means agents cannot make provider-backed turns until endpoint and model settings are added.",
	}
	if provider != "" || endpoint != "" {
		providerStep.Status = StatusConfigured
		providerStep.Detail = fmt.Sprintf("Provider %s at %s will be pre-filled.", providerLabel(provider), endpointLabel(endpoint))
	}

	authStep := Step{
		ID:          StepAuth,
		Title:       "Auth",
		Status:      StatusMissing,
		Detail:      "No provider credential was found.",
		NextCommand: "gormes auth add <provider>",
		SkipWarning: "Skipping auth leaves provider calls unavailable until a credential is stored or exported.",
	}
	if input.APIKeyPresent {
		authStep.Status = StatusConfigured
		authStep.Detail = "Provider credential present; secret value is not shown."
	}

	gatewayStep := Step{
		ID:          StepGateway,
		Title:       "Gateway",
		Status:      StatusMissing,
		Detail:      "No messaging platform is configured.",
		NextCommand: "gormes setup gateway",
		SkipWarning: "Skipping gateway setup keeps Telegram, Discord, Slack, and other messaging adapters disabled.",
	}
	if len(input.GatewayTargets) > 0 {
		gatewayStep.Status = StatusConfigured
		gatewayStep.Detail = "Configured messaging targets: " + strings.Join(input.GatewayTargets, ", ")
	}

	browserStep := Step{
		ID:          StepBrowser,
		Title:       "Browser/CDP",
		Status:      StatusMissing,
		Detail:      "No browser/CDP endpoint is configured.",
		NextCommand: "gormes doctor --offline",
		SkipWarning: "Skipping browser checks leaves browser automation unavailable until a local or cloud browser backend is configured.",
	}
	if browserCDPURL != "" {
		browserStep.Status = StatusConfigured
		browserStep.Detail = "Browser/CDP endpoint will be pre-filled: " + browserCDPURL
	}

	skillsStep := Step{
		ID:          StepSkills,
		Title:       "Skills",
		Status:      StatusAvailable,
		Detail:      fmt.Sprintf("Skill discovery sees %d local and %d bundled skills.", input.LocalSkills, input.BundledSkills),
		NextCommand: "gormes skills list",
		SkipWarning: "Skipping skill discovery means the runtime will still use configured skills, but the operator has not reviewed what is installed.",
	}

	dashboardStep := Step{
		ID:          StepDashboard,
		Title:       "Dashboard",
		Status:      StatusAvailable,
		Detail:      "Dashboard launch is available as a manual step.",
		NextCommand: "gormes dashboard",
		SkipWarning: "Skipping dashboard launch keeps onboarding in the terminal; runtime state can still be inspected with CLI commands.",
	}

	return Plan{Steps: []Step{
		modelStep,
		providerStep,
		authStep,
		gatewayStep,
		browserStep,
		skillsStep,
		dashboardStep,
	}}
}

func (p Plan) Step(id string) (Step, bool) {
	for _, step := range p.Steps {
		if step.ID == id {
			return step, true
		}
	}
	return Step{}, false
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
