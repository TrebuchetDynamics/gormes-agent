package acp

import (
	"context"
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type ClientReportJSON = gormescli.ACPClientReportJSON

type BrowserBootstrapReportJSON = gormescli.ACPBrowserBootstrapReportJSON

type BrowserBootstrapOptions = gormescli.ACPBrowserBootstrapOptions

type ClientOptions = gormescli.ACPClientOptions

type BuildProvenance = gormescli.BuildProvenance

type ProvenanceMode = gormescli.ACPProvenanceMode

const ProvenanceOff = gormescli.ACPProvenanceOff

func DefaultClientOptions() ClientOptions {
	return ClientOptions{ServerCommand: "gormes"}
}

func ParseProvenanceMode(raw string) (ProvenanceMode, error) {
	return gormescli.ACPParseProvenanceMode(raw)
}

func RunSetupBrowser(ctx context.Context, stdout io.Writer, opts BrowserBootstrapOptions, jsonOut bool, build BuildProvenance) error {
	return gormescli.ACPRunSetupBrowser(ctx, stdout, opts, jsonOut, build)
}

func RunServe(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	return gormescli.ACPRunServe(ctx, stdin, stdout, stderr)
}

func RunClient(ctx context.Context, stdout io.Writer, opts ClientOptions, jsonOut bool, build BuildProvenance) error {
	return gormescli.ACPRunClient(ctx, stdout, opts, jsonOut, build)
}

func ExitCode(err error) int {
	return gormescli.ACPExitCode(err)
}
