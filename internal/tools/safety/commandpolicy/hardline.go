package commandpolicy

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/commandpolicy/hardline"

type HardlinePattern = hardline.HardlinePattern

const PythonRuntimeDisabledMessage = hardline.PythonRuntimeDisabledMessage
const FindRootWalkCommand = hardline.FindRootWalkCommand

var HardlinePatterns = hardline.HardlinePatterns

func DetectHardline(cmd string) (bool, string) {
	return hardline.DetectHardline(cmd)
}
