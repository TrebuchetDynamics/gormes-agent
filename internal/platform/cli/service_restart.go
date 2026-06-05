package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/service"

const (
	DefaultServiceRestartDelay        = service.DefaultServiceRestartDelay
	DefaultMaxServiceRestartDelay     = service.DefaultMaxServiceRestartDelay
	DefaultServiceRestartPollTimeout  = service.DefaultServiceRestartPollTimeout
	DefaultServiceRestartPollInterval = service.DefaultServiceRestartPollInterval
)

type ServiceManagerKind = service.ServiceManagerKind

const (
	ServiceManagerSystemd     = service.ServiceManagerSystemd
	ServiceManagerUnsupported = service.ServiceManagerUnsupported
)

type SystemdPATHOptions = service.SystemdPATHOptions

func SystemdPATHEnvironment(options SystemdPATHOptions) string {
	return service.SystemdPATHEnvironment(options)
}

type ServiceRestartDelayEvidenceKind = service.ServiceRestartDelayEvidenceKind

const (
	RestartDelayDefaulted     = service.RestartDelayDefaulted
	RestartDelayMalformed     = service.RestartDelayMalformed
	RestartDelayInfinite      = service.RestartDelayInfinite
	RestartDelayMissing       = service.RestartDelayMissing
	RestartDelayUnsupported   = service.RestartDelayUnsupported
	ServiceManagerUnavailable = service.ServiceManagerUnavailable
	RestartDelayBounded       = service.RestartDelayBounded
)

type ServiceRestartDelaySource = service.ServiceRestartDelaySource
type ServiceRestartDelayReport = service.ServiceRestartDelayReport
type ServiceRestartDelayEvidence = service.ServiceRestartDelayEvidence

type ServiceActiveStatus = service.ServiceActiveStatus

const (
	ServiceActiveStatusActive     = service.ServiceActiveStatusActive
	ServiceActiveStatusInactive   = service.ServiceActiveStatusInactive
	ServiceActiveStatusActivating = service.ServiceActiveStatusActivating
	ServiceActiveStatusFailed     = service.ServiceActiveStatusFailed
	ServiceActiveStatusUnknown    = service.ServiceActiveStatusUnknown
)

type ServiceRestartPollOutcome = service.ServiceRestartPollOutcome

const (
	ServiceRestartPollRestarted           = service.ServiceRestartPollRestarted
	ServiceRestartPollTimeout             = service.ServiceRestartPollTimeout
	ServiceRestartPollManagerUnavailable  = service.ServiceRestartPollManagerUnavailable
	ServiceRestartPollCrashedAfterRestart = service.ServiceRestartPollCrashedAfterRestart
)

type ServiceRestartPollEvidenceKind = service.ServiceRestartPollEvidenceKind

const (
	ServiceRestartPollActiveEvidence              = service.ServiceRestartPollActiveEvidence
	ServiceRestartPollCooldownEvidence            = service.ServiceRestartPollCooldownEvidence
	ServiceRestartPollTimeoutEvidence             = service.ServiceRestartPollTimeoutEvidence
	ServiceRestartPollManagerUnavailableEvidence  = service.ServiceRestartPollManagerUnavailableEvidence
	ServiceRestartPollCrashedAfterRestartEvidence = service.ServiceRestartPollCrashedAfterRestartEvidence
	ServiceRestartPollRetryEvidence               = service.ServiceRestartPollRetryEvidence
)

type ServiceActiveStatusCheck = service.ServiceActiveStatusCheck
type ServiceActiveStatusRunner = service.ServiceActiveStatusRunner
type ServiceRestartPollClock = service.ServiceRestartPollClock
type ServiceRestartPollOptions = service.ServiceRestartPollOptions
type ServiceRestartPollReport = service.ServiceRestartPollReport
type ServiceRestartPollEvidence = service.ServiceRestartPollEvidence

func PollServiceRestartActive(options ServiceRestartPollOptions) ServiceRestartPollReport {
	return service.PollServiceRestartActive(options)
}

func ParseServiceRestartDelay(source ServiceRestartDelaySource) ServiceRestartDelayReport {
	return service.ParseServiceRestartDelay(source)
}
