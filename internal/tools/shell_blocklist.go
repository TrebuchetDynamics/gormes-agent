package tools

var ShellBlocklistPatterns = DangerousPatterns

func IsShellCommandBlocked(cmd string) bool {
	result := GuardCommand(cmd, ApprovalModeManual)
	return result.Description != "" && !result.Approved
}
