package main

import (
	"fmt"

	"github.com/spf13/cobra"

	mcplogincmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/mcplogin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type mcpLoginRuntime struct {
	loadConfig func() (tools.MCPConfigResolution, error)
	store      *tools.MCPOAuthStore
	flow       tools.MCPLoginFlow
}

func newMCPCommand() *cobra.Command {
	return newMCPCommandWithRuntime(mcpLoginRuntime{})
}

func newMCPCommandWithRuntime(runtime mcpLoginRuntime) *cobra.Command {
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
					return emitJSONSubcommandRequired(cmd)
				}
				return cmd.Help()
			}
			msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
			if asJSON {
				return emitJSONInputError(cmd, "unknown_subcommand", msg)
			}
			return fmt.Errorf("%s", msg)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON on invalid invocation: {build, action: 'unknown_subcommand', error}")
	cmd.AddCommand(newMCPLoginCommand(runtime))
	cmd.AddCommand(newMCPRowBackedCommands()...)
	return cmd
}

func newMCPRowBackedCommands() []*cobra.Command {
	return []*cobra.Command{
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "serve",
			Short: "Run a Hermes-compatible MCP server",
			Row:   hermesACPMCPRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "add <name>",
			Short: "Add an MCP server",
			Row:   hermesACPMCPRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "remove <name>",
			Aliases:     []string{"rm"},
			Short:       "Remove an MCP server",
			Row:         hermesACPMCPRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:     "list",
			Aliases: []string{"ls"},
			Short:   "List MCP servers",
			Row:     hermesACPMCPRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "test <name>",
			Short: "Test an MCP server",
			Row:   hermesACPMCPRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:     "configure <name>",
			Aliases: []string{"config"},
			Short:   "Configure an MCP server",
			Row:     hermesACPMCPRow,
		}),
	}
}

func newMCPLoginCommand(runtime mcpLoginRuntime) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "login <name>",
		Short:        "Refresh OAuth login for an MCP server",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPLoginCommand(cmd, runtime, args[0])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, server, evidence, message, available}")
	return cmd
}

func runMCPLoginCommand(cmd *cobra.Command, runtime mcpLoginRuntime, serverName string) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	err := mcplogincmd.Run(cmd.Context(), mcplogincmd.Runtime{
		LoadConfig: runtime.loadConfig,
		Store:      runtime.store,
		Flow:       runtime.flow,
	}, mcplogincmd.Options{
		ServerName: serverName,
		JSON:       asJSON,
		Stdout:     cmd.OutOrStdout(),
		Build:      mcpLoginBuildProvenance(),
	})
	if err != nil {
		return newExitCodeError(mcplogincmd.ExitCodeForError(err), err)
	}
	return nil
}

func loadDefaultMCPConfig() (tools.MCPConfigResolution, error) {
	return mcplogincmd.LoadDefaultMCPConfig()
}

func mcpLoginBuildProvenance() mcplogincmd.BuildProvenance {
	build := newBuildProvenance()
	return mcplogincmd.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}
