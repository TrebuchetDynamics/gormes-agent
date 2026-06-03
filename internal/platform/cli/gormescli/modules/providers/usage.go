package providers

import (
	"github.com/spf13/cobra"

	appusage "github.com/TrebuchetDynamics/gormes-agent/internal/app/usage"
)

type UsageSeams = appusage.UsageSeams

type UsageReportJSON = appusage.UsageReportJSON

type UsageInvocation = appusage.UsageInvocation

type AccountUsageHTTPClient = appusage.AccountUsageHTTPClient

var UsageHTTPClient = appusage.UsageHTTPClient

func DefaultUsageSeams() UsageSeams {
	return appusage.DefaultUsageSeams()
}

func NewUsageCommand(opts Options) *cobra.Command {
	return appusage.NewUsageCommand(usageOptions(opts))
}

func NewUsageCommandWithSeams(seams UsageSeams, opts Options) *cobra.Command {
	return appusage.NewUsageCommandWithSeams(seams, usageOptions(opts))
}

func RunUsageCommand(cmd *cobra.Command, invocation UsageInvocation, opts Options) error {
	return appusage.RunUsageCommand(cmd, invocation, usageOptions(opts))
}

func InferUsageProvider(configuredProvider, model string) string {
	return appusage.InferUsageProvider(configuredProvider, model)
}

func FirstUsageString(values ...string) string {
	return appusage.FirstUsageString(values...)
}

func usageOptions(opts Options) appusage.Options {
	return appusage.Options{
		BuildProvenance: func() appusage.BuildProvenance {
			build := opts.buildProvenance()
			return appusage.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
		},
	}
}
