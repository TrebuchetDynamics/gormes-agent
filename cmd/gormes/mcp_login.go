package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// mcpLoginReportJSON is the wire shape for `mcp login <name> --json`.
// Fleet automation orchestrating MCP token refresh parses this to
// reason about every typed evidence value without scraping prose.
// Raw tokens are NEVER present — the result struct doesn't carry them.
type mcpLoginReportJSON struct {
	Build     buildProvenanceJSON `json:"build"`
	Server    string              `json:"server"`
	Evidence  string              `json:"evidence"`
	Message   string              `json:"message,omitempty"`
	Available []string            `json:"available,omitempty"`
}

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
	return cmd
}

func newMCPLoginCommand(runtime mcpLoginRuntime) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "login <name>",
		Short:        "Refresh OAuth login for an MCP server",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPLoginCommand(cmd.Context(), cmd, runtime, args[0])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, server, evidence, message, available}")
	return cmd
}

func runMCPLoginCommand(ctx context.Context, cmd *cobra.Command, runtime mcpLoginRuntime, serverName string) error {
	loadConfig := runtime.loadConfig
	if loadConfig == nil {
		loadConfig = loadDefaultMCPConfig
	}
	resolution, err := loadConfig()
	if err != nil {
		return newExitCodeError(2, fmt.Errorf("mcp_config_unavailable: %w", err))
	}
	store := runtime.store
	if store == nil {
		store = tools.NewMCPOAuthStore()
	}
	flow := runtime.flow
	if flow == nil {
		flow = tools.NoninteractiveLoginFlow()
	}
	result, err := tools.RunMCPLogin(ctx, resolution, store, flow, serverName)
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		body, marshalErr := json.MarshalIndent(mcpLoginReportJSON{
			Build:     newBuildProvenance(),
			Server:    result.Server,
			Evidence:  string(result.Evidence),
			Message:   result.Message,
			Available: result.Available,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
	} else if result.Evidence == tools.MCPLoginEvidenceSaved {
		// Success path emits to stdout. On the error path cobra
		// renders the returned `result` error on stderr already; the
		// previous stdout print duplicated that line, so operators
		// saw two identical "evidence=..." rows. Only print to
		// stdout when there's nothing else carrying the message.
		fmt.Fprintln(cmd.OutOrStdout(), result.Error())
	}
	switch result.Evidence {
	case tools.MCPLoginEvidenceSaved:
		return nil
	case tools.MCPLoginEvidenceServerUnknown, tools.MCPLoginEvidenceAuthNotOAuth, tools.MCPLoginEvidenceNoninteractiveRequired, tools.MCPLoginEvidenceFlowFailed, tools.MCPLoginEvidenceStateStoreUnwritable:
		return newExitCodeError(2, result)
	default:
		return newExitCodeError(2, result)
	}
}

func loadDefaultMCPConfig() (tools.MCPConfigResolution, error) {
	return tools.MCPConfigResolution{}, nil
}
