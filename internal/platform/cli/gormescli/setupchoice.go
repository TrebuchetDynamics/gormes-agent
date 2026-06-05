package gormescli

import appsetupchoice "github.com/TrebuchetDynamics/gormes-agent/internal/app/setupchoice"

type SetupOptionChoice = appsetupchoice.Choice

func NormalizeSetupAnswer(answer string, options []SetupOptionChoice, defaultID string) string {
	return appsetupchoice.NormalizeAnswer(answer, options, defaultID)
}

func ParseSetupYesNo(value string, defaultValue bool) (bool, bool) {
	return appsetupchoice.YesNo(value, defaultValue)
}

func NormalizeSetupValue(value string) string {
	return appsetupchoice.NormalizeValue(value)
}

func StripSetupInputNoise(answer string) string {
	return appsetupchoice.StripInputNoise(answer)
}
