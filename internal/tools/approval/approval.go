package approval

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/approval/check"
)

type PatternMatch = check.PatternMatch
type CheckResult = check.CheckResult

func CheckHardline(command string) CheckResult {
	return check.CheckHardline(command)
}

func CheckDangerous(command string) (bool, string) {
	return check.CheckDangerous(command)
}

func CheckAll(command string) CheckResult {
	return check.CheckAll(command)
}
