package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

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
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage Hermes-compatible MCP servers",
	}
	cmd.AddCommand(newMCPLoginCommand(runtime))
	return cmd
}

func newMCPLoginCommand(runtime mcpLoginRuntime) *cobra.Command {
	return &cobra.Command{
		Use:   "login <name>",
		Short: "Refresh OAuth login for an MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPLoginCommand(cmd.Context(), cmd, runtime, args[0])
		},
	}
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
	fmt.Fprintln(cmd.OutOrStdout(), result.Error())
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
	paths := defaultHermesConfigCandidates()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return tools.ParseMCPConfigYAML(data, tools.MCPConfigOptions{})
	}
	return tools.MCPConfigResolution{}, nil
}

func defaultHermesConfigCandidates() []string {
	var paths []string
	hermesHome := strings.TrimSpace(os.Getenv("HERMES_HOME"))
	if hermesHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			hermesHome = filepath.Join(home, ".hermes")
		}
	}
	profile := strings.TrimSpace(os.Getenv("HERMES_PROFILE"))
	if profile == "" {
		profile = "mineru"
	}
	if hermesHome != "" {
		paths = append(paths, filepath.Join(hermesHome, "profiles", profile, "config.yaml"))
		paths = append(paths, filepath.Join(hermesHome, "config.yaml"))
	}
	return paths
}
