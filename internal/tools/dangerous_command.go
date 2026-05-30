package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"

// DangerousPatterns is the recoverable dangerous-command pattern table,
// ported from Hermes DANGEROUS_PATTERNS at tools/approval.py@eb28145f.
var DangerousPatterns = safety.DangerousPatterns

// BlockedResult is the pure guard result returned for commands that match a
// hardline or recoverable dangerous-command rule.
type BlockedResult = safety.BlockedResult

// DetectDangerous reports whether cmd matches any recoverable dangerous rule.
func DetectDangerous(cmd string) (bool, string) {
	return safety.DetectDangerous(cmd)
}

// GuardCommand applies the pure dangerous-command guard. Hardline matches
// always block first; recoverable dangerous matches require approval.
func GuardCommand(cmd, mode string) BlockedResult {
	approvalMode := NormalizeApprovalMode(mode)
	return safety.GuardCommand(cmd, approvalMode.Mode)
}

// GuardCronCommand applies Hermes' approvals.cron_mode contract for
// noninteractive cron turns.
func GuardCronCommand(cmd string, mode any) BlockedResult {
	cronMode := NormalizeCronApprovalMode(mode)
	return safety.GuardCronCommand(cmd, cronMode.Mode)
}
