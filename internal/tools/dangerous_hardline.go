package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"

// HardlinePattern is a single entry in the unconditional hardline blocklist.
type HardlinePattern = safety.HardlinePattern

var HardlinePatterns = safety.HardlinePatterns

const findRootWalkCommand = safety.FindRootWalkCommand

// DetectHardline reports whether cmd matches any unconditional hardline rule.
func DetectHardline(cmd string) (bool, string) {
	return safety.DetectHardline(cmd)
}
