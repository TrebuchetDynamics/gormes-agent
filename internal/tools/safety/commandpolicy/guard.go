package commandpolicy

import policyguard "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/commandpolicy/guard"

var DangerousPatterns = policyguard.DangerousPatterns

type BlockedResult = policyguard.BlockedResult

func DetectDangerous(cmd string) (bool, string) {
	return policyguard.DetectDangerous(cmd)
}

func GuardCommand(cmd, mode string) BlockedResult {
	return policyguard.GuardCommand(cmd, mode)
}

func GuardCronCommand(cmd string, mode any) BlockedResult {
	return policyguard.GuardCronCommand(cmd, mode)
}
