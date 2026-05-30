package approval

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"

type PatternMatch struct {
	Pattern     string
	Description string
}

type CheckResult struct {
	Approved    bool   `json:"approved"`
	Hardline    bool   `json:"hardline,omitempty"`
	Dangerous   bool   `json:"dangerous,omitempty"`
	Description string `json:"description,omitempty"`
	Message     string `json:"message,omitempty"`
}

func hardlineBlockResult(desc string) CheckResult {
	return CheckResult{
		Approved:    false,
		Hardline:    true,
		Description: desc,
		Message: "BLOCKED (hardline): " + desc + ". This command is on the unconditional " +
			"blocklist and cannot be executed via the agent — not even with --yolo, " +
			"/yolo, approvals.mode=off, or cron approve mode.",
	}
}

func CheckHardline(command string) CheckResult {
	if match, desc := safety.DetectHardline(command); match {
		return hardlineBlockResult(desc)
	}
	return CheckResult{Approved: true}
}

func CheckDangerous(command string) (bool, string) {
	return safety.DetectDangerous(command)
}

func CheckAll(command string) CheckResult {
	if result := CheckHardline(command); !result.Approved {
		return result
	}
	if match, desc := CheckDangerous(command); match {
		return CheckResult{
			Approved:    false,
			Dangerous:   true,
			Description: desc,
			Message:     "This command matched dangerous pattern: " + desc + ". Approval required.",
		}
	}
	return CheckResult{Approved: true}
}
