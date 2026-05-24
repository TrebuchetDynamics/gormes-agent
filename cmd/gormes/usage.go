package main

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/providers"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

type usageInvocation = providermodule.UsageInvocation

type accountUsageHTTPClient struct{ client *http.Client }

var usageHTTPClient = providermodule.UsageHTTPClient
var inferUsageProvider = providermodule.InferUsageProvider
var firstUsageString = providermodule.FirstUsageString

func (c accountUsageHTTPClient) DoAccountUsageRequest(ctx context.Context, req hermes.AccountUsageHTTPRequest) (hermes.AccountUsageHTTPResponse, error) {
	return providermodule.AccountUsageHTTPClient{Client: c.client}.DoAccountUsageRequest(ctx, req)
}

func newUsageCommand() *cobra.Command {
	return providermodule.NewUsageCommand(providerCommandOptions())
}

func runUsageCommand(cmd *cobra.Command, invocation usageInvocation) error {
	return providermodule.RunUsageCommand(cmd, invocation, providerCommandOptions())
}

func providerCommandOptions() providermodule.Options {
	return providermodule.Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
	}
}
