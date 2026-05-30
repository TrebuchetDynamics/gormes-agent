package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/onboarding"

const (
	OnboardStepModel     = onboarding.OnboardStepModel
	OnboardStepProvider  = onboarding.OnboardStepProvider
	OnboardStepAuth      = onboarding.OnboardStepAuth
	OnboardStepGateway   = onboarding.OnboardStepGateway
	OnboardStepBrowser   = onboarding.OnboardStepBrowser
	OnboardStepSkills    = onboarding.OnboardStepSkills
	OnboardStepDashboard = onboarding.OnboardStepDashboard

	OnboardStatusConfigured = onboarding.OnboardStatusConfigured
	OnboardStatusMissing    = onboarding.OnboardStatusMissing
	OnboardStatusAvailable  = onboarding.OnboardStatusAvailable
)

type OnboardPlanInput = onboarding.OnboardPlanInput

type OnboardPlan = onboarding.OnboardPlan

type OnboardStep = onboarding.OnboardStep

func BuildOnboardPlan(input OnboardPlanInput) OnboardPlan { return onboarding.BuildOnboardPlan(input) }
