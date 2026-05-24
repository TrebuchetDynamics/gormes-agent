package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/acp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

// acpClientReportJSON wraps acp.ClientResult with build provenance so
// fleet automation bridging ACP sessions across machines can attribute
// each connect/degraded outcome to the binary version that emitted it.
// Existing ClientResult fields stay top-level via struct embedding —
// callers parsing the old shape continue to work because Go's JSON
// decoder ignores the unknown `build` field by default.
type acpClientReportJSON struct {
	Build buildProvenanceJSON `json:"build"`
	acp.ClientResult
}

type acpBrowserBootstrapReportJSON struct {
	Build  buildProvenanceJSON `json:"build"`
	Action string              `json:"action"`
	acp.BrowserBootstrapReport
}

func newACPCommand() *cobra.Command {
	var (
		setupBrowser bool
		jsonOut      bool
		opts         acp.BrowserBootstrapOptions
	)
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Run ACP bridge tools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if setupBrowser {
				opts.HomeDir = config.GormesHome()
				return runACPSetupBrowserCommand(cmd, opts, jsonOut)
			}
			return cmd.Help()
		},
	}
	cmd.Flags().BoolVar(&setupBrowser, "setup-browser", false, "plan or run browser tool bootstrap for ACP registry installs")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "preview ACP browser bootstrap without installing packages")
	cmd.Flags().BoolVarP(&opts.AssumeYes, "yes", "y", false, "approve ACP browser bootstrap package installation")
	cmd.Flags().BoolVar(&opts.SkipChromium, "skip-chromium", false, "install agent-browser without downloading Playwright Chromium")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "write machine-readable ACP setup-browser report")
	cmd.AddCommand(newACPServeCommand())
	cmd.AddCommand(newACPClientCommand())
	return cmd
}

func runACPSetupBrowserCommand(cmd *cobra.Command, opts acp.BrowserBootstrapOptions, jsonOut bool) error {
	report := acp.RunBrowserBootstrap(cmd.Context(), opts)
	if jsonOut {
		envelope := acpBrowserBootstrapReportJSON{
			Build:                  newBuildProvenance(),
			Action:                 "acp_setup_browser",
			BrowserBootstrapReport: report,
		}
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(envelope); err != nil {
			return err
		}
	} else {
		writeACPSetupBrowserText(cmd.OutOrStdout(), report)
	}
	if !report.OK {
		msg := report.Message
		if msg == "" {
			msg = report.Evidence.Code
		}
		return newExitCodeError(1, errors.New(msg))
	}
	return nil
}

func writeACPSetupBrowserText(w io.Writer, report acp.BrowserBootstrapReport) {
	if report.DryRun {
		fmt.Fprintln(w, "ACP browser bootstrap dry-run")
	} else if report.Executed {
		fmt.Fprintln(w, "ACP browser bootstrap executed")
	} else {
		fmt.Fprintln(w, "ACP browser bootstrap")
	}
	if report.Evidence.Code != "" {
		fmt.Fprintf(w, "evidence: %s\n", report.Evidence.Code)
	}
	if report.Platform != "" {
		fmt.Fprintf(w, "platform: %s\n", report.Platform)
	}
	if report.NodePrefix != "" {
		fmt.Fprintf(w, "node_prefix: %s\n", report.NodePrefix)
	}
	for _, step := range report.Steps {
		fmt.Fprintf(w, "- %s: %s", step.Name, step.Status)
		if len(step.Command) > 0 {
			fmt.Fprintf(w, " `%s`", strings.Join(step.Command, " "))
		}
		if step.Message != "" {
			fmt.Fprintf(w, " — %s", step.Message)
		}
		fmt.Fprintln(w)
	}
	if report.Message != "" {
		fmt.Fprintf(w, "message: %s\n", report.Message)
	}
}

func newACPServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the Go-native ACP JSON-RPC stdio server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runACPServeCommand(cmd)
		},
	}
}

func runACPServeCommand(cmd *cobra.Command) error {
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return newExitCodeError(2, fmt.Errorf("acp server session store unavailable: %w", err))
	}
	defer smap.Close()

	runtime := acp.NewSessionRuntime(acp.SessionRuntimeConfig{
		SessionMap: smap,
	})
	server := acp.NewJSONRPCServer(runtime)
	server.Diagnostics = acp.NewStdioDiagnostics(cmd.ErrOrStderr())
	if err := server.Handle(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout()); err != nil {
		return newExitCodeError(1, err)
	}
	return nil
}

func newACPClientCommand() *cobra.Command {
	var (
		opts          acp.ClientOptions
		provenanceRaw string
		jsonOut       bool
	)
	opts.ServerCommand = "gormes"

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Connect a debug ACP client to the Go-native ACP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode, err := acp.ParseProvenanceMode(provenanceRaw)
			if err != nil {
				return newExitCodeError(2, err)
			}
			opts.ProvenanceMode = mode
			return runACPClientCommand(cmd, opts, jsonOut)
		},
	}
	cmd.Flags().StringVar(&opts.SessionKey, "session", "", "session key to bridge, for example agent:main:main")
	cmd.Flags().StringVar(&opts.SessionLabel, "session-label", "", "session title or id to resolve before bridging")
	cmd.Flags().BoolVar(&opts.RequireExisting, "require-existing", false, "fail when the resolved session key does not already exist")
	cmd.Flags().BoolVar(&opts.ResetSession, "reset-session", false, "clear and reinitialize the resolved session key before connecting")
	cmd.Flags().BoolVar(&opts.NoPrefixCWD, "no-prefix-cwd", false, "do not prepend working-directory provenance to bridged prompts")
	cmd.Flags().StringVar(&provenanceRaw, "provenance", string(acp.ProvenanceOff), "provenance mode: off, meta, or meta+receipt")
	cmd.Flags().StringVar(&opts.CWD, "cwd", "", "working directory to expose to the ACP bridge")
	cmd.Flags().StringVar(&opts.ServerCommand, "server", opts.ServerCommand, "ACP server command label")
	cmd.Flags().StringArrayVar(&opts.ServerArgs, "server-args", nil, "additional ACP server argument, repeatable")
	cmd.Flags().BoolVar(&opts.ServerVerbose, "server-verbose", false, "request verbose ACP server logging")
	cmd.Flags().BoolVar(&opts.Verbose, "verbose", false, "print verbose ACP client diagnostics")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "write machine-readable ACP client result")
	return cmd
}

func runACPClientCommand(cmd *cobra.Command, opts acp.ClientOptions, jsonOut bool) error {
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return newExitCodeError(2, fmt.Errorf("acp client session store unavailable: %w", err))
	}
	defer smap.Close()

	result, err := (acp.ClientBridge{
		Resolver:  acp.NewSessionMapResolver(smap),
		Connector: acp.LocalClientConnector{},
	}).Run(cmd.Context(), opts)
	if err != nil {
		return newExitCodeError(2, err)
	}

	if jsonOut {
		envelope := acpClientReportJSON{
			Build:        newBuildProvenance(),
			ClientResult: result,
		}
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(envelope); err != nil {
			return err
		}
	} else {
		writeACPClientText(cmd.OutOrStdout(), result)
	}
	if !result.OK {
		return newExitCodeError(1, errors.New(result.Message))
	}
	return nil
}

func writeACPClientText(w io.Writer, result acp.ClientResult) {
	if result.OK {
		fmt.Fprintln(w, "ACP client connected")
	} else {
		fmt.Fprintln(w, "ACP client degraded")
	}
	if result.SessionKey != "" {
		fmt.Fprintf(w, "session_key: %s\n", result.SessionKey)
	}
	if result.SessionID != "" {
		fmt.Fprintf(w, "session_id: %s\n", result.SessionID)
	}
	if result.SessionLabel != "" {
		fmt.Fprintf(w, "session_label: %s\n", result.SessionLabel)
	}
	if result.ProvenanceMode != "" {
		fmt.Fprintf(w, "provenance: %s\n", result.ProvenanceMode)
	}
	if result.Reset {
		fmt.Fprintln(w, "reset_session: true")
	}
	if result.Evidence.Code != "" {
		fmt.Fprintf(w, "evidence: %s\n", result.Evidence.Code)
	}
	if result.Evidence.Reason != "" {
		fmt.Fprintf(w, "reason: %s\n", result.Evidence.Reason)
	}
	if result.Message != "" {
		fmt.Fprintf(w, "message: %s\n", result.Message)
	}
}
