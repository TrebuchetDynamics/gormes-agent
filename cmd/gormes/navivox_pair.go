package main

import (
	"context"

	"github.com/spf13/cobra"

	navivoxapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type navivoxPairOptions struct {
	host           string
	port           int
	noWait         bool
	openNavivox    bool
	noOpenNavivox  bool
	androidPackage string
	printDeeplink  bool
	portExplicit   bool
}

type navivoxPairRuntimeStore = navivoxapp.PairRuntimeStore

type navivoxPairTarget = navivoxapp.Target

var newNavivoxPairRuntimeStore = func(path string) navivoxPairRuntimeStore {
	return gateway.NewRuntimeStatusStore(path)
}

func newNavivoxPairCommand() *cobra.Command {
	return navivoxapp.NewPairCommand()
}

func runNavivoxPair(cmd *cobra.Command, opts navivoxPairOptions) error {
	navivoxapp.SetVPNHostList(vpnhostList)
	previous := navivoxapp.SetPairRuntimeStoreFactory(newNavivoxPairRuntimeStore)
	defer navivoxapp.SetPairRuntimeStoreFactory(previous)
	return navivoxapp.RunPair(cmd, navivoxapp.PairOptions{
		Host:           opts.host,
		Port:           opts.port,
		NoWait:         opts.noWait,
		OpenNavivox:    opts.openNavivox,
		NoOpenNavivox:  opts.noOpenNavivox,
		AndroidPackage: opts.androidPackage,
		PrintDeeplink:  opts.printDeeplink,
		PortExplicit:   opts.portExplicit,
	})
}

func ensureNoLiveGatewayForNavivoxPair(ctx context.Context) error {
	return navivoxapp.EnsureNoLiveGatewayForPair(ctx)
}

func resolveNavivoxPairTarget(ctx context.Context, requestedHost string) (navivoxPairTarget, error) {
	navivoxapp.SetVPNHostList(vpnhostList)
	return navivoxapp.ResolvePairTarget(ctx, requestedHost)
}

func navivoxPairExposureForHost(host string) string {
	return navivoxapp.ExposureForHost(host)
}

func navivoxPairLoopbackHost(host string) bool {
	return navivoxapp.LoopbackHost(host)
}

func navivoxPairLANIPv4() string {
	return navivoxapp.LANIPv4()
}

func navivoxPairDescriptor(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	return navivoxapp.PairDescriptorForConfig(cfg, baseURL, wsURL)
}

func writeNavivoxPairQR(path, descriptor string) error {
	return navivoxapp.WritePairQR(path, descriptor)
}

func generateNavivoxSetupToken() (string, error) {
	return navivoxapp.GenerateSetupToken()
}
