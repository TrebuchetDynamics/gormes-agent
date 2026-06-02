package main

import (
	"io"

	"github.com/spf13/cobra"

	navivoxapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type navivoxConnectInfoEntry = navivoxapp.ConnectInfoEntry
type navivoxConnectInfoReport = navivoxapp.ConnectInfoReport

type navivoxConnectInfoOptions struct {
	jsonOut        bool
	openNavivox    bool
	noOpenNavivox  bool
	printDeeplink  bool
	androidPackage string
}

func syncNavivoxAppSeams() { navivoxapp.SetVPNHostList(vpnhostList) }

func newNavivoxCommand() *cobra.Command {
	syncNavivoxAppSeams()
	return navivoxapp.NewCommand(navivoxapp.CommandOptions{VPNHostList: vpnhostList})
}

func runNavivoxConnectInfo(cmd *cobra.Command, cfg config.NavivoxCfg, jsonOut bool) error {
	syncNavivoxAppSeams()
	return navivoxapp.RunConnectInfo(cmd, cfg, jsonOut)
}

func runNavivoxConnectInfoForConfig(cmd *cobra.Command, cfg config.Config, jsonOut bool) error {
	syncNavivoxAppSeams()
	return navivoxapp.RunConnectInfoForConfig(cmd, cfg, jsonOut)
}

func runNavivoxConnectInfoForConfigWithOptions(cmd *cobra.Command, cfg config.Config, opts navivoxConnectInfoOptions) error {
	syncNavivoxAppSeams()
	return navivoxapp.RunConnectInfoForConfigWithFlags(cmd, cfg, opts.jsonOut, opts.openNavivox, opts.noOpenNavivox, opts.printDeeplink, opts.androidPackage)
}

func buildNavivoxConnectInfoEntries(cmd *cobra.Command, cfg config.NavivoxCfg) []navivoxConnectInfoEntry {
	syncNavivoxAppSeams()
	return navivoxapp.BuildConnectInfoEntries(cmd, cfg)
}

func buildNavivoxConnectInfoEntriesForConfig(cmd *cobra.Command, cfg config.Config) []navivoxConnectInfoEntry {
	syncNavivoxAppSeams()
	return navivoxapp.BuildConnectInfoEntriesForConfig(cmd, cfg)
}

func navivoxServerBindHostPort(bind string, cfg config.NavivoxCfg) (string, int) {
	return navivoxapp.NavivoxServerBindHostPort(bind, cfg)
}

func navivoxConnectInfoURLs(host string, port int) (baseURL, webSocketURL string) {
	return navivoxapp.NavivoxConnectInfoURLs(host, port)
}

func writeNavivoxConnectInfoJSON(out io.Writer, entries []navivoxConnectInfoEntry) error {
	return navivoxapp.WriteConnectInfoJSON(out, entries)
}

func writeNavivoxConnectInfoText(out io.Writer, cfg config.NavivoxCfg, entries []navivoxConnectInfoEntry) error {
	return navivoxapp.WriteConnectInfoText(out, cfg, entries)
}
