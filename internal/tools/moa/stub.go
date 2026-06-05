package moa

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/moa/stub"

type MoATool = stub.MoATool
type MoARequest = stub.MoARequest

func NewMoATool() *MoATool {
	return stub.NewMoATool()
}
