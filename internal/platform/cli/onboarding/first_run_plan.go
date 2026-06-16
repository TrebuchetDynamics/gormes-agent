package onboarding

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/onboarding/firstrun"

type SetupTargetID = firstrun.SetupTargetID

const (
	SetupTargetTerminal SetupTargetID = firstrun.SetupTargetTerminal
	SetupTargetTelegram SetupTargetID = firstrun.SetupTargetTelegram
	SetupTargetWhatsApp SetupTargetID = firstrun.SetupTargetWhatsApp
	SetupTargetDiscord  SetupTargetID = firstrun.SetupTargetDiscord
	SetupTargetSlack    SetupTargetID = firstrun.SetupTargetSlack
	SetupTargetNavivox  SetupTargetID = firstrun.SetupTargetNavivox
)

type FirstRunActionID = firstrun.ActionID

const (
	FirstRunActionQuick           FirstRunActionID = firstrun.ActionQuick
	FirstRunActionFull            FirstRunActionID = firstrun.ActionFull
	FirstRunActionMigrateHermes   FirstRunActionID = firstrun.ActionMigrateHermes
	FirstRunActionMigrateOpenClaw FirstRunActionID = firstrun.ActionMigrateOpenClaw
)

type FirstRunStepID = firstrun.StepID

const (
	FirstRunStepProvider FirstRunStepID = firstrun.StepProvider
	FirstRunStepAuth     FirstRunStepID = firstrun.StepAuth
	FirstRunStepModel    FirstRunStepID = firstrun.StepModel
	FirstRunStepChannel  FirstRunStepID = firstrun.StepChannel
)

type FirstRunPlanInput = firstrun.PlanInput
type ChannelState = firstrun.ChannelState
type SetupTargetOption = firstrun.SetupTargetOption
type FirstRunAction = firstrun.Action
type FirstRunStep = firstrun.Step
type FirstRunPlan = firstrun.Plan

func BuildFirstRunPlan(input FirstRunPlanInput) FirstRunPlan {
	return firstrun.BuildPlan(input)
}

func DefaultFirstRunChannels(overrides map[SetupTargetID]ChannelState) []ChannelState {
	return firstrun.DefaultChannels(overrides)
}
