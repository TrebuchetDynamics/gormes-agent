package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/statusreport"

const DefaultStatusProgressPath = statusreport.DefaultStatusProgressPath

type StatusReportOptions = statusreport.StatusReportOptions
type StatusBlocker = statusreport.StatusBlocker

func CollectStatusBlockers(opts StatusReportOptions) ([]StatusBlocker, error) {
	return statusreport.CollectStatusBlockers(opts)
}

func RenderStatusReport(opts StatusReportOptions) (string, error) {
	return statusreport.RenderStatusReport(opts)
}
