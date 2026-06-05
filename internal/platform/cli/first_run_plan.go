package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/onboarding"

type SetupTargetID = onboarding.SetupTargetID

const (
	SetupTargetTerminal = onboarding.SetupTargetTerminal
	SetupTargetTelegram = onboarding.SetupTargetTelegram
	SetupTargetWhatsApp = onboarding.SetupTargetWhatsApp
	SetupTargetDiscord  = onboarding.SetupTargetDiscord
	SetupTargetSlack    = onboarding.SetupTargetSlack
	SetupTargetNavivox  = onboarding.SetupTargetNavivox
)

type FirstRunActionID = onboarding.FirstRunActionID

const (
	FirstRunActionQuick           = onboarding.FirstRunActionQuick
	FirstRunActionFull            = onboarding.FirstRunActionFull
	FirstRunActionMigrateHermes   = onboarding.FirstRunActionMigrateHermes
	FirstRunActionMigrateOpenClaw = onboarding.FirstRunActionMigrateOpenClaw
)

type FirstRunStepID = onboarding.FirstRunStepID

const (
	FirstRunStepProvider = onboarding.FirstRunStepProvider
	FirstRunStepAuth     = onboarding.FirstRunStepAuth
	FirstRunStepModel    = onboarding.FirstRunStepModel
	FirstRunStepChannel  = onboarding.FirstRunStepChannel
)

type FirstRunPlanInput = onboarding.FirstRunPlanInput

type ChannelState = onboarding.ChannelState

type SetupTargetOption = onboarding.SetupTargetOption

type FirstRunAction = onboarding.FirstRunAction

type FirstRunStep = onboarding.FirstRunStep

type FirstRunPlan = onboarding.FirstRunPlan

func BuildFirstRunPlan(input FirstRunPlanInput) FirstRunPlan {
	return onboarding.BuildFirstRunPlan(input)
}

func DefaultFirstRunChannels(overrides map[SetupTargetID]ChannelState) []ChannelState {
	return onboarding.DefaultFirstRunChannels(overrides)
}
