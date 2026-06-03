package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// acpClientReportJSON wraps acp.ClientResult with build provenance so
// fleet automation bridging ACP sessions across machines can attribute
// each connect/degraded outcome to the binary version that emitted it.
// Existing ClientResult fields stay top-level via struct embedding —
// callers parsing the old shape continue to work because Go's JSON
// decoder ignores the unknown `build` field by default.
type acpClientReportJSON = gormescli.ACPClientReportJSON

type acpBrowserBootstrapReportJSON = gormescli.ACPBrowserBootstrapReportJSON

func newACPCommand() *cobra.Command {
	var (
		setupBrowser bool
		jsonOut      bool
		opts         gormescli.ACPBrowserBootstrapOptions
	)
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Run ACP bridge tools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if setupBrowser {
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

func runACPSetupBrowserCommand(cmd *cobra.Command, opts gormescli.ACPBrowserBootstrapOptions, jsonOut bool) error {
	return acpExitError(gormescli.ACPRunSetupBrowser(cmd.Context(), cmd.OutOrStdout(), opts, jsonOut, acpBuildProvenance()))
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
	return acpExitError(gormescli.ACPRunServe(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()))
}

func newACPClientCommand() *cobra.Command {
	var (
		opts          gormescli.ACPClientOptions
		provenanceRaw string
		jsonOut       bool
	)
	opts.ServerCommand = "gormes"

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Connect a debug ACP client to the Go-native ACP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode, err := gormescli.ACPParseProvenanceMode(provenanceRaw)
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
	cmd.Flags().StringVar(&provenanceRaw, "provenance", string(gormescli.ACPProvenanceOff), "provenance mode: off, meta, or meta+receipt")
	cmd.Flags().StringVar(&opts.CWD, "cwd", "", "working directory to expose to the ACP bridge")
	cmd.Flags().StringVar(&opts.ServerCommand, "server", opts.ServerCommand, "ACP server command label")
	cmd.Flags().StringArrayVar(&opts.ServerArgs, "server-args", nil, "additional ACP server argument, repeatable")
	cmd.Flags().BoolVar(&opts.ServerVerbose, "server-verbose", false, "request verbose ACP server logging")
	cmd.Flags().BoolVar(&opts.Verbose, "verbose", false, "print verbose ACP client diagnostics")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "write machine-readable ACP client result")
	return cmd
}

func runACPClientCommand(cmd *cobra.Command, opts gormescli.ACPClientOptions, jsonOut bool) error {
	return acpExitError(gormescli.ACPRunClient(cmd.Context(), cmd.OutOrStdout(), opts, jsonOut, acpBuildProvenance()))
}

func acpExitError(err error) error {
	if err == nil {
		return nil
	}
	if code := gormescli.ACPExitCode(err); code != 0 {
		return newExitCodeError(code, err)
	}
	return err
}

func acpBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}
