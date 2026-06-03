package acpapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/acpreport"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/acp"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type ClientReportJSON struct {
	Build BuildProvenance `json:"build"`
	acp.ClientResult
}

type BrowserBootstrapReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	acp.BrowserBootstrapReport
}

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e ExitError) Unwrap() error { return e.Err }

func ExitCode(err error) int {
	var exit ExitError
	if errors.As(err, &exit) {
		return exit.Code
	}
	return 0
}

func RunSetupBrowser(ctx context.Context, stdout io.Writer, opts acp.BrowserBootstrapOptions, jsonOut bool, build BuildProvenance) error {
	if opts.HomeDir == "" {
		opts.HomeDir = config.GormesHome()
	}
	report := acp.RunBrowserBootstrap(ctx, opts)
	if jsonOut {
		envelope := BrowserBootstrapReportJSON{Build: build, Action: "acp_setup_browser", BrowserBootstrapReport: report}
		if err := json.NewEncoder(stdout).Encode(envelope); err != nil {
			return err
		}
	} else {
		acpreport.WriteBrowserBootstrapText(stdout, report)
	}
	if !report.OK {
		msg := report.Message
		if msg == "" {
			msg = report.Evidence.Code
		}
		return ExitError{Code: 1, Err: errors.New(msg)}
	}
	return nil
}

func RunServe(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("acp server session store unavailable: %w", err)}
	}
	defer smap.Close()

	runtime := acp.NewSessionRuntime(acp.SessionRuntimeConfig{SessionMap: smap})
	server := acp.NewJSONRPCServer(runtime)
	server.Diagnostics = acp.NewStdioDiagnostics(stderr)
	if err := server.Handle(ctx, stdin, stdout); err != nil {
		return ExitError{Code: 1, Err: err}
	}
	return nil
}

func RunClient(ctx context.Context, stdout io.Writer, opts acp.ClientOptions, jsonOut bool, build BuildProvenance) error {
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("acp client session store unavailable: %w", err)}
	}
	defer smap.Close()

	result, err := (acp.ClientBridge{Resolver: acp.NewSessionMapResolver(smap), Connector: acp.LocalClientConnector{}}).Run(ctx, opts)
	if err != nil {
		return ExitError{Code: 2, Err: err}
	}
	if jsonOut {
		envelope := ClientReportJSON{Build: build, ClientResult: result}
		if err := json.NewEncoder(stdout).Encode(envelope); err != nil {
			return err
		}
	} else {
		acpreport.WriteClientText(stdout, result)
	}
	if !result.OK {
		return ExitError{Code: 1, Err: errors.New(result.Message)}
	}
	return nil
}
