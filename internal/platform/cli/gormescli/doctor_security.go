package gormescli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/securityruntime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

func DoctorSecurityAdvisoriesStatus(ackID, home string) doctor.CheckResult {
	return securityruntime.DoctorSecurityAdvisoriesStatus(ackID, home)
}
