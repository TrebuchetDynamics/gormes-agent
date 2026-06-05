package gormescli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	appacp "github.com/TrebuchetDynamics/gormes-agent/internal/app/acp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/acp"
)

type ACPClientReportJSON = appacp.ClientReportJSON

type ACPBrowserBootstrapReportJSON = appacp.BrowserBootstrapReportJSON

type ACPBrowserBootstrapOptions = acp.BrowserBootstrapOptions

type ACPClientOptions = acp.ClientOptions

type ACPClientResult = acp.ClientResult

type ACPProvenanceMode = acp.ProvenanceMode

type ACPCommandOptions struct {
	BuildProvenance func() BuildProvenance
	ExitError       func(code int, err error) error
}

const (
	ACPProvenanceOff           = acp.ProvenanceOff
	ACPClientEvidenceConnected = acp.ClientEvidenceConnected
	ACPClientEvidenceRowBacked = acp.ClientEvidenceRowBacked
)

func ACPDefaultClientOptions() ACPClientOptions {
	return ACPClientOptions{ServerCommand: "gormes"}
}

func NewACPCommand(opts ACPCommandOptions) *cobra.Command {
	if opts.BuildProvenance == nil {
		opts.BuildProvenance = func() BuildProvenance { return BuildProvenance{} }
	}
	var (
		setupBrowser bool
		jsonOut      bool
		browserOpts  ACPBrowserBootstrapOptions
	)
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Run ACP bridge tools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if setupBrowser {
				return acpCommandExitError(opts, ACPRunSetupBrowser(cmd.Context(), cmd.OutOrStdout(), browserOpts, jsonOut, opts.BuildProvenance()))
			}
			return cmd.Help()
		},
	}
	cmd.Flags().BoolVar(&setupBrowser, "setup-browser", false, "plan or run browser tool bootstrap for ACP registry installs")
	cmd.Flags().BoolVar(&browserOpts.DryRun, "dry-run", false, "preview ACP browser bootstrap without installing packages")
	cmd.Flags().BoolVarP(&browserOpts.AssumeYes, "yes", "y", false, "approve ACP browser bootstrap package installation")
	cmd.Flags().BoolVar(&browserOpts.SkipChromium, "skip-chromium", false, "install agent-browser without downloading Playwright Chromium")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "write machine-readable ACP setup-browser report")
	cmd.AddCommand(newACPServeCommand(opts))
	cmd.AddCommand(newACPClientCommand(opts))
	return cmd
}

func newACPServeCommand(opts ACPCommandOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the Go-native ACP JSON-RPC stdio server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return acpCommandExitError(opts, ACPRunServe(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()))
		},
	}
}

func newACPClientCommand(opts ACPCommandOptions) *cobra.Command {
	var (
		clientOpts    = ACPDefaultClientOptions()
		provenanceRaw string
		jsonOut       bool
	)

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Connect a debug ACP client to the Go-native ACP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode, err := ACPParseProvenanceMode(provenanceRaw)
			if err != nil {
				return acpCommandExitCodeError(opts, 2, err)
			}
			clientOpts.ProvenanceMode = mode
			return acpCommandExitError(opts, ACPRunClient(cmd.Context(), cmd.OutOrStdout(), clientOpts, jsonOut, opts.BuildProvenance()))
		},
	}
	cmd.Flags().StringVar(&clientOpts.SessionKey, "session", "", "session key to bridge, for example agent:main:main")
	cmd.Flags().StringVar(&clientOpts.SessionLabel, "session-label", "", "session title or id to resolve before bridging")
	cmd.Flags().BoolVar(&clientOpts.RequireExisting, "require-existing", false, "fail when the resolved session key does not already exist")
	cmd.Flags().BoolVar(&clientOpts.ResetSession, "reset-session", false, "clear and reinitialize the resolved session key before connecting")
	cmd.Flags().BoolVar(&clientOpts.NoPrefixCWD, "no-prefix-cwd", false, "do not prepend working-directory provenance to bridged prompts")
	cmd.Flags().StringVar(&provenanceRaw, "provenance", string(ACPProvenanceOff), "provenance mode: off, meta, or meta+receipt")
	cmd.Flags().StringVar(&clientOpts.CWD, "cwd", "", "working directory to expose to the ACP bridge")
	cmd.Flags().StringVar(&clientOpts.ServerCommand, "server", clientOpts.ServerCommand, "ACP server command label")
	cmd.Flags().StringArrayVar(&clientOpts.ServerArgs, "server-args", nil, "additional ACP server argument, repeatable")
	cmd.Flags().BoolVar(&clientOpts.ServerVerbose, "server-verbose", false, "request verbose ACP server logging")
	cmd.Flags().BoolVar(&clientOpts.Verbose, "verbose", false, "print verbose ACP client diagnostics")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "write machine-readable ACP client result")
	return cmd
}

func acpCommandExitError(opts ACPCommandOptions, err error) error {
	if err == nil {
		return nil
	}
	if code := ACPExitCode(err); code != 0 {
		return acpCommandExitCodeError(opts, code, err)
	}
	return err
}

func acpCommandExitCodeError(opts ACPCommandOptions, code int, err error) error {
	if opts.ExitError != nil {
		return opts.ExitError(code, err)
	}
	return err
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
