package bridgecmd

import (
	"context"
	"io"
	"time"

	appbridgecmd "github.com/TrebuchetDynamics/gormes-agent/internal/app/bridgecmd"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type Server = appbridgecmd.Server

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
	return appbridgecmd.Run(ctx, appbridgecmd.Options{
		BindHost:        opts.BindHost,
		BindPort:        opts.BindPort,
		GatewayHost:     opts.GatewayHost,
		GatewayPort:     opts.GatewayPort,
		GormesBin:       opts.GormesBin,
		Out:             opts.Out,
		ServerFactory:   opts.ServerFactory,
		ShutdownTimeout: opts.ShutdownTimeout,
	})
}
