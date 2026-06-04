package bridgecmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type Options struct {
	BindHost        string
	BindPort        int
	GatewayHost     string
	GatewayPort     int
	GormesBin       string
	Out             io.Writer
	ServerFactory   func(gormescli.BridgeConfig) Server
	ShutdownTimeout time.Duration
}

func Run(ctx context.Context, opts Options) error {
	cfg := gormescli.BridgeConfig{
		BindHost:    opts.BindHost,
		BindPort:    opts.BindPort,
		GatewayHost: opts.GatewayHost,
		GatewayPort: opts.GatewayPort,
		GormesBin:   opts.GormesBin,
	}
	factory := opts.ServerFactory
	if factory == nil {
		factory = func(cfg gormescli.BridgeConfig) Server { return gormescli.NewBridgeServer(cfg) }
	}
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	shutdownTimeout := opts.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 5 * time.Second
	}

	srv := factory(cfg)
	fmt.Fprintf(out, "Navivox bridge starting on %s\n", cfg.BindAddr())
	fmt.Fprintf(out, "Proxying to gateway at %s\n", cfg.GatewayAddr())
	fmt.Fprintf(out, "Health: http://%s/health\n", cfg.BindAddr())
	fmt.Fprintf(out, "Bootstrap: POST http://%s/bootstrap/termux\n", cfg.BindAddr())

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("bridge: %w", err)
	}

	<-ctx.Done()
	slog.Info("bridge shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Stop(shutdownCtx)
}
