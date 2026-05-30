package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/moa"

type MoATool = moa.MoATool
type MoARequest = moa.MoARequest

func NewMoATool() *MoATool {
	return moa.NewMoATool()
}
