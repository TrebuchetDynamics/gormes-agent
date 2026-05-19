package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/bridge"
)

func newBootstrapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap and provision Gormes runtime",
		Long: `Bootstrap commands for provisioning the Gormes runtime environment.

Currently supported:
  gormes bootstrap termux          Auto-provision Gormes in Termux
  gormes bootstrap termux --dry-run  Preview provisioning steps`,
		SilenceUsage: true,
	}

	cmd.AddCommand(newBootstrapTermuxCommand())
	return cmd
}

func newBootstrapTermuxCommand() *cobra.Command {
	var dryRun bool
	var stream bool
	var gatewayPort int

	cmd := &cobra.Command{
		Use:   "termux",
		Short: "Auto-provision Gormes in Termux environment",
		Long: `Provision Gormes runtime in Termux with idempotent detection.

Steps:
  1. Check platform compatibility
  2. Verify Termux installation
  3. Check Termux:API availability
  4. Detect existing gateway (idempotent)
  5. Install/update gormes binary
  6. Configure Termux:Boot
  7. Start gateway
  8. Verify gateway health

Use --dry-run to preview without making changes.
Use --stream for SSE event output.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBootstrapTermuxCommand(cmd, dryRun, stream, gatewayPort)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview steps without executing")
	cmd.Flags().BoolVar(&stream, "stream", false, "output SSE-style events")
	cmd.Flags().IntVar(&gatewayPort, "gateway-port", 8766, "gateway port to check")

	return cmd
}

func runBootstrapTermuxCommand(cmd *cobra.Command, dryRun bool, stream bool, gatewayPort int) error {
	cfg := bridge.DefaultConfig()
	cfg.GatewayPort = gatewayPort
	cfg.GormesBin = resolveGormesBinPath()

	if stream {
		return runBootstrapTermuxStream(cmd, cfg, dryRun)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
	defer cancel()

	result := bridge.RunBootstrapTermux(ctx, cfg, dryRun)

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func runBootstrapTermuxStream(cmd *cobra.Command, cfg bridge.Config, dryRun bool) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 30 * time.Second}
	addr := cfg.BindAddr()

	url := fmt.Sprintf("http://%s/bootstrap/termux?stream=true", addr)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(fmt.Sprintf(`{"dry_run": %t}`, dryRun)))
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Bridge not running at %s, running bootstrap directly\n", addr)
		return runBootstrapTermuxDirect(cmd, cfg, dryRun)
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			cmd.OutOrStdout().Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	return nil
}

func runBootstrapTermuxDirect(cmd *cobra.Command, cfg bridge.Config, dryRun bool) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
	defer cancel()

	result := bridge.RunBootstrapTermux(ctx, cfg, dryRun)

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
