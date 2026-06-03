package gormescli

import (
	"context"
	"io"

	appmcplogin "github.com/TrebuchetDynamics/gormes-agent/internal/app/mcplogin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type MCPLoginBuildProvenance = appmcplogin.BuildProvenance

type MCPLoginRuntime = appmcplogin.Runtime

type MCPLoginOptions = appmcplogin.Options

type MCPLoginReportJSON = appmcplogin.ReportJSON

func RunMCPLoginCommand(ctx context.Context, runtime MCPLoginRuntime, serverName string, asJSON bool, stdout io.Writer, build BuildProvenance) error {
	return appmcplogin.Run(ctx, runtime, appmcplogin.Options{
		ServerName: serverName,
		JSON:       asJSON,
		Stdout:     stdout,
		Build: appmcplogin.BuildProvenance{
			Version:   build.Version,
			GitCommit: build.GitCommit,
		},
	})
}

func LoadDefaultMCPConfig() (tools.MCPConfigResolution, error) {
	return appmcplogin.LoadDefaultMCPConfig()
}
