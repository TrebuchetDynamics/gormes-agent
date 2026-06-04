package bridgecmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type fakeServer struct {
	start func(context.Context) error
	stop  func(context.Context) error
}

func (f fakeServer) Start(ctx context.Context) error {
	if f.start != nil {
		return f.start(ctx)
	}
	return nil
}

func (f fakeServer) Stop(ctx context.Context) error {
	if f.stop != nil {
		return f.stop(ctx)
	}
	return nil
}

func TestRunPrintsBridgeEndpointsAndStopsAfterContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)
	var gotCfg gormescli.BridgeConfig
	var out bytes.Buffer

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, Options{
			BindHost:    "127.0.0.2",
			BindPort:    1234,
			GatewayHost: "127.0.0.3",
			GatewayPort: 5678,
			GormesBin:   "/tmp/gormes",
			Out:         &out,
			ServerFactory: func(cfg gormescli.BridgeConfig) Server {
				gotCfg = cfg
				return fakeServer{
					start: func(context.Context) error {
						started <- struct{}{}
						return nil
					},
					stop: func(context.Context) error {
						stopped <- struct{}{}
						return nil
					},
				}
			},
		})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("server did not start")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Run error = %v", err)
	}

	if gotCfg.BindAddr() != "127.0.0.2:1234" || gotCfg.GatewayAddr() != "127.0.0.3:5678" || gotCfg.GormesBin != "/tmp/gormes" {
		t.Fatalf("cfg = %+v", gotCfg)
	}
	for _, want := range []string{
		"Navivox bridge starting on 127.0.0.2:1234",
		"Proxying to gateway at 127.0.0.3:5678",
		"Health: http://127.0.0.2:1234/health",
		"Bootstrap: POST http://127.0.0.2:1234/bootstrap/termux",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunWrapsStartError(t *testing.T) {
	startErr := errors.New("listen failed")
	err := Run(context.Background(), Options{
		ServerFactory: func(gormescli.BridgeConfig) Server {
			return fakeServer{start: func(context.Context) error { return startErr }}
		},
	})
	if !errors.Is(err, startErr) || !strings.Contains(err.Error(), "bridge: listen failed") {
		t.Fatalf("err = %v", err)
	}
}
