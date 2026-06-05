package gormescli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/mcplogin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const mcpRowBackedRow = "ACP server side"

type MCPLoginBuildProvenance = mcplogin.BuildProvenance
type MCPLoginReportJSON = mcplogin.ReportJSON
type MCPLoginRuntime = mcplogin.Runtime
type MCPLoginOptions = mcplogin.Options

type MCPCommandOptions struct {
	BuildProvenance func() BuildProvenance
	ExitCodeError   func(int, error) error
}

func (opts MCPCommandOptions) withDefaults() MCPCommandOptions {
	if opts.BuildProvenance == nil {
		opts.BuildProvenance = func() BuildProvenance { return BuildProvenance{} }
	}
	if opts.ExitCodeError == nil {
		opts.ExitCodeError = NewExitCodeError
	}
	return opts
}

func (opts MCPCommandOptions) buildProvenance() BuildProvenance {
	if opts.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return opts.BuildProvenance()
}

func (opts MCPCommandOptions) exitCodeError(code int, err error) error {
	if opts.ExitCodeError == nil {
		return NewExitCodeError(code, err)
	}
	return opts.ExitCodeError(code, err)
}

func NewMCPCommand(opts MCPCommandOptions) *cobra.Command {
	return NewMCPCommandWithRuntime(MCPLoginRuntime{}, opts)
}

func NewMCPCommandWithRuntime(runtime MCPLoginRuntime, opts MCPCommandOptions) *cobra.Command {
	opts = opts.withDefaults()
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage Hermes-compatible MCP servers",
		// Accept arbitrary args so cobra delegates to RunE instead of
		// emitting a default "unknown command" error to stderr. RunE
		// then routes: no args → help; one+ arg → structured
		// unknown-subcommand response (JSON when --json is set, else
		// the human-readable cobra-style message).
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if asJSON {
					return emitMCPJSONSubcommandRequired(cmd, opts)
				}
				return cmd.Help()
			}
			msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
			if asJSON {
				return emitMCPJSONInputError(cmd, "unknown_subcommand", msg, opts)
			}
			return fmt.Errorf("%s", msg)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON on invalid invocation: {build, action: 'unknown_subcommand', error}")
	cmd.AddCommand(newMCPLoginCommand(runtime, opts))
	cmd.AddCommand(newMCPRowBackedCommands(opts)...)
	return cmd
}

func newMCPRowBackedCommands(opts MCPCommandOptions) []*cobra.Command {
	return []*cobra.Command{
		newMCPRowBackedCommand(RowBackedCommandSpec{
			Use:   "serve",
			Short: "Run a Hermes-compatible MCP server",
			Row:   mcpRowBackedRow,
		}, opts),
		newMCPRowBackedCommand(RowBackedCommandSpec{
			Use:   "add <name>",
			Short: "Add an MCP server",
			Row:   mcpRowBackedRow,
		}, opts),
		newMCPRowBackedCommand(RowBackedCommandSpec{
			Use:         "remove <name>",
			Aliases:     []string{"rm"},
			Short:       "Remove an MCP server",
			Row:         mcpRowBackedRow,
			Destructive: true,
			FlagSet:     mcpUnavailableYesFlag,
		}, opts),
		newMCPRowBackedCommand(RowBackedCommandSpec{
			Use:     "list",
			Aliases: []string{"ls"},
			Short:   "List MCP servers",
			Row:     mcpRowBackedRow,
		}, opts),
		newMCPRowBackedCommand(RowBackedCommandSpec{
			Use:   "test <name>",
			Short: "Test an MCP server",
			Row:   mcpRowBackedRow,
		}, opts),
		newMCPRowBackedCommand(RowBackedCommandSpec{
			Use:     "configure <name>",
			Aliases: []string{"config"},
			Short:   "Configure an MCP server",
			Row:     mcpRowBackedRow,
		}, opts),
	}
}

func newMCPRowBackedCommand(spec RowBackedCommandSpec, opts MCPCommandOptions, children ...*cobra.Command) *cobra.Command {
	return NewRowBackedCommand(spec, RowBackedCommandOptions{BuildProvenance: opts.BuildProvenance}, children...)
}

func mcpUnavailableYesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
}

func newMCPLoginCommand(runtime MCPLoginRuntime, opts MCPCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "login <name>",
		Short:        "Refresh OAuth login for an MCP server",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPLoginCommand(cmd, runtime, opts, args[0])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, server, evidence, message, available}")
	return cmd
}

func runMCPLoginCommand(cmd *cobra.Command, runtime MCPLoginRuntime, opts MCPCommandOptions, serverName string) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	build := opts.buildProvenance()
	err := RunMCPLogin(cmd.Context(), runtime, MCPLoginOptions{
		ServerName: serverName,
		JSON:       asJSON,
		Stdout:     cmd.OutOrStdout(),
		Build:      MCPLoginBuildProvenance{Version: build.Version, GitCommit: build.GitCommit},
	})
	if err != nil {
		return opts.exitCodeError(MCPLoginExitCodeForError(err), err)
	}
	return nil
}

func emitMCPJSONInputError(cmd *cobra.Command, action, errMsg string, opts MCPCommandOptions) error {
	err := EmitJSONInputError(cmd, action, errMsg, opts.buildProvenance())
	return opts.exitCodeError(1, err)
}

func emitMCPJSONSubcommandRequired(cmd *cobra.Command, opts MCPCommandOptions) error {
	available := make([]string, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		available = append(available, child.Name())
	}
	parent := cmd.CommandPath()
	report := struct {
		Build     BuildProvenance `json:"build"`
		Action    string          `json:"action"`
		Parent    string          `json:"parent"`
		Available []string        `json:"available"`
		Error     string          `json:"error"`
	}{
		Build:     opts.buildProvenance(),
		Action:    "subcommand_required",
		Parent:    parent,
		Available: available,
		Error:     fmt.Sprintf("subcommand required for %q; choose one of: %s", parent, strings.Join(available, ", ")),
	}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
	return opts.exitCodeError(1, fmt.Errorf("%s", report.Error))
}

func RunMCPLogin(ctx context.Context, runtime MCPLoginRuntime, opts MCPLoginOptions) error {
	return mcplogin.Run(ctx, runtime, opts)
}

func MCPLoginExitCodeForError(err error) int {
	return mcplogin.ExitCodeForError(err)
}

func LoadDefaultMCPConfig() (tools.MCPConfigResolution, error) {
	return mcplogin.LoadDefaultMCPConfig()
}
