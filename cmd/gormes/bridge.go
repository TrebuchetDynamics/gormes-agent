package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newBridgeCommand() *cobra.Command {
	var bindHost string
	var bindPort int
	var gatewayHost string
	var gatewayPort int

	cmd := &cobra.Command{
		Use:   "bridge",
		Short: "Start the Navivox bridge HTTP server",
		Long: `Start a localhost HTTP bridge that proxies requests to the Gormes gateway
and provides bootstrap endpoints for Navivox Termux auto-provisioning.

The bridge binds to 127.0.0.1:8765 by default and proxies to the gateway
at 127.0.0.1:8766. This is the control plane for the Navivox Flutter app.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgeCommand(cmd, bindHost, bindPort, gatewayHost, gatewayPort)
		},
	}

	cmd.Flags().StringVar(&bindHost, "bind", "127.0.0.1", "bridge bind host")
	cmd.Flags().IntVar(&bindPort, "port", 8765, "bridge bind port")
	cmd.Flags().StringVar(&gatewayHost, "gateway-host", "127.0.0.1", "gateway host")
	cmd.Flags().IntVar(&gatewayPort, "gateway-port", 8766, "gateway port")

	return cmd
}

func runBridgeCommand(cmd *cobra.Command, bindHost string, bindPort int, gatewayHost string, gatewayPort int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return gormescli.RunBridgeCommand(ctx, gormescli.BridgeCommandOptions{
		BindHost:    bindHost,
		BindPort:    bindPort,
		GatewayHost: gatewayHost,
		GatewayPort: gatewayPort,
		GormesBin:   resolveGormesBinPath(),
		Out:         cmd.ErrOrStderr(),
	})
}

func resolveGormesBinPath() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "gormes"
}
