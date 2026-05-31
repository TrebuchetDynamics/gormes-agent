package safety

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/commandpolicy"

// HardlinePattern is a single entry in the unconditional hardline blocklist.
type HardlinePattern = commandpolicy.HardlinePattern

const PythonRuntimeDisabledMessage = commandpolicy.PythonRuntimeDisabledMessage
const FindRootWalkCommand = commandpolicy.FindRootWalkCommand

// HardlinePatterns is the unconditional hardline blocklist, ported from
// Hermes HARDLINE_PATTERNS (tools/approval.py@eb28145f).
var HardlinePatterns = commandpolicy.HardlinePatterns

// DetectHardline reports whether cmd matches any unconditional hardline rule.
func DetectHardline(cmd string) (bool, string) {
	return commandpolicy.DetectHardline(cmd)
}
