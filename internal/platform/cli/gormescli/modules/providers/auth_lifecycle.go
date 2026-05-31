package providers

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/authreport"

// AuthLifecycleReportJSON is the shared lifecycle wire shape for provider auth
// add/remove/reset/logout reports.
type AuthLifecycleReportJSON = authreport.LifecycleReportJSON

type AuthRemovedJSON = authreport.RemovedJSON

func WriteAuthLifecycleJSON(out interface{ Write(p []byte) (int, error) }, report AuthLifecycleReportJSON) error {
	return authreport.WriteLifecycleJSON(out, report)
}
