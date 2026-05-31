package safety

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/commandpolicy"

// DangerousPatterns is the recoverable dangerous-command pattern table,
// ported from Hermes DANGEROUS_PATTERNS at tools/approval.py@eb28145f.
var DangerousPatterns = commandpolicy.DangerousPatterns

// BlockedResult is the pure guard result returned for commands that match a
// hardline or recoverable dangerous-command rule.
type BlockedResult = commandpolicy.BlockedResult

// DetectDangerous reports whether cmd matches any recoverable dangerous rule.
func DetectDangerous(cmd string) (bool, string) {
	return commandpolicy.DetectDangerous(cmd)
}

// GuardCommand applies the pure dangerous-command guard.
func GuardCommand(cmd, mode string) BlockedResult {
	return commandpolicy.GuardCommand(cmd, mode)
}

// GuardCronCommand applies Hermes' approvals.cron_mode contract for
// noninteractive cron turns.
func GuardCronCommand(cmd string, mode any) BlockedResult {
	return commandpolicy.GuardCronCommand(cmd, mode)
}
