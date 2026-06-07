package onboarding

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/onboarding/wizard"

const (
	OnboardStepModel     = wizard.StepModel
	OnboardStepProvider  = wizard.StepProvider
	OnboardStepAuth      = wizard.StepAuth
	OnboardStepGateway   = wizard.StepGateway
	OnboardStepBrowser   = wizard.StepBrowser
	OnboardStepSkills    = wizard.StepSkills
	OnboardStepDashboard = wizard.StepDashboard

	OnboardStatusConfigured = wizard.StatusConfigured
	OnboardStatusMissing    = wizard.StatusMissing
	OnboardStatusAvailable  = wizard.StatusAvailable
)

type OnboardPlanInput = wizard.PlanInput
type OnboardPlan = wizard.Plan
type OnboardStep = wizard.Step

func BuildOnboardPlan(input OnboardPlanInput) OnboardPlan {
	return wizard.BuildPlan(input)
}
