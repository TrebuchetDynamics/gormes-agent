package update

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/service"

type ServiceRestartPollOutcome = service.ServiceRestartPollOutcome

const (
	ServiceRestartPollRestarted           = service.ServiceRestartPollRestarted
	ServiceRestartPollTimeout             = service.ServiceRestartPollTimeout
	ServiceRestartPollManagerUnavailable  = service.ServiceRestartPollManagerUnavailable
	ServiceRestartPollCrashedAfterRestart = service.ServiceRestartPollCrashedAfterRestart
)

type ServiceRestartPollReport = service.ServiceRestartPollReport
