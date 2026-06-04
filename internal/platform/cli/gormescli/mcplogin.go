package gormescli

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/mcplogin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type MCPLoginBuildProvenance = mcplogin.BuildProvenance
type MCPLoginReportJSON = mcplogin.ReportJSON
type MCPLoginRuntime = mcplogin.Runtime
type MCPLoginOptions = mcplogin.Options

func RunMCPLogin(ctx context.Context, runtime MCPLoginRuntime, opts MCPLoginOptions) error {
	return mcplogin.Run(ctx, runtime, opts)
}

func MCPLoginExitCodeForError(err error) int {
	return mcplogin.ExitCodeForError(err)
}

func LoadDefaultMCPConfig() (tools.MCPConfigResolution, error) {
	return mcplogin.LoadDefaultMCPConfig()
}
