package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/moa"

const MixtureOfAgentsName = moa.MixtureOfAgentsName

var ErrMOAInsufficientReferences = moa.ErrMOAInsufficientReferences

type MOAConfig = moa.MOAConfig
type MOATool = moa.MOATool
type MOARouter = moa.MOARouter

func NewMOATool(cfg MOAConfig, router MOARouter) Tool {
	return moa.NewMOATool(cfg, router)
}
