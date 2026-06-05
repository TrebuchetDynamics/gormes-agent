package kernel

import kerneladmission "github.com/TrebuchetDynamics/gormes-agent/internal/kernel/admission"

var (
	ErrEmptyInput    = kerneladmission.ErrEmptyInput
	ErrInputTooLarge = kerneladmission.ErrInputTooLarge
	ErrTooManyLines  = kerneladmission.ErrTooManyLines
	ErrTurnInFlight  = kerneladmission.ErrTurnInFlight
)

type Admission = kerneladmission.Admission
