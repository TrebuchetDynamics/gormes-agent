package gormescli

import (
	"context"
	"io"

	appacp "github.com/TrebuchetDynamics/gormes-agent/internal/app/acp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/acp"
)

type ACPClientReportJSON = appacp.ClientReportJSON

type ACPBrowserBootstrapReportJSON = appacp.BrowserBootstrapReportJSON

type ACPBrowserBootstrapOptions = acp.BrowserBootstrapOptions

type ACPClientOptions = acp.ClientOptions

type ACPClientResult = acp.ClientResult

type ACPProvenanceMode = acp.ProvenanceMode

const (
	ACPProvenanceOff           = acp.ProvenanceOff
	ACPClientEvidenceConnected = acp.ClientEvidenceConnected
	ACPClientEvidenceRowBacked = acp.ClientEvidenceRowBacked
)

func ACPDefaultClientOptions() ACPClientOptions {
	return ACPClientOptions{ServerCommand: "gormes"}
}

func ACPParseProvenanceMode(raw string) (ACPProvenanceMode, error) {
	return acp.ParseProvenanceMode(raw)
}

func ACPRunSetupBrowser(ctx context.Context, stdout io.Writer, opts ACPBrowserBootstrapOptions, jsonOut bool, build BuildProvenance) error {
	return appacp.RunSetupBrowser(ctx, stdout, opts, jsonOut, appacp.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit})
}

func ACPRunServe(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	return appacp.RunServe(ctx, stdin, stdout, stderr)
}

func ACPRunClient(ctx context.Context, stdout io.Writer, opts ACPClientOptions, jsonOut bool, build BuildProvenance) error {
	return appacp.RunClient(ctx, stdout, opts, jsonOut, appacp.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit})
}

func ACPExitCode(err error) int {
	return appacp.ExitCode(err)
}
