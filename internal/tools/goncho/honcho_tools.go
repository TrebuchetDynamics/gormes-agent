package gonchotools

import (
	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	honchoadapter "github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho/honcho"
)

// RegisterHonchoTools adds the Honcho-compatible tool surface backed by the
// in-binary Goncho service.
func RegisterHonchoTools(reg *tools.Registry, svc *goncho.Service) {
	honchoadapter.RegisterHonchoTools(reg, svc)
}

type HonchoProfileTool = honchoadapter.HonchoProfileTool
type HonchoSearchTool = honchoadapter.HonchoSearchTool
type HonchoContextTool = honchoadapter.HonchoContextTool
type HonchoChatTool = honchoadapter.HonchoChatTool
type HonchoReasoningTool = honchoadapter.HonchoReasoningTool
type HonchoConcludeTool = honchoadapter.HonchoConcludeTool
