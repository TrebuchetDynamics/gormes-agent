package moa

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/moa/runtime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

const MixtureOfAgentsName = runtime.MixtureOfAgentsName

var ErrMOAInsufficientReferences = runtime.ErrMOAInsufficientReferences

type MOAConfig = runtime.MOAConfig
type MOATool = runtime.MOATool
type MOARouter = runtime.MOARouter

func NewMOATool(cfg MOAConfig, router MOARouter) toolkit.Tool {
	return runtime.NewMOATool(cfg, router)
}
