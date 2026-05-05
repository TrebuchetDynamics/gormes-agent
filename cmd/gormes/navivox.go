package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func newNavivoxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "navivox",
		Short: "Run the Navivox SSH stdio channel",
	}
	cmd.AddCommand(newNavivoxServeCommand())
	return cmd
}

func newNavivoxServeCommand() *cobra.Command {
	var stdio bool
	cmd := &cobra.Command{
		Use:          "serve",
		Short:        "Serve the Navivox protocol over stdin/stdout",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !stdio {
				return newExitCodeError(2, fmt.Errorf("navivox serve requires --stdio"))
			}
			status := navivox.StaticStatusProvider{StatusValue: navivox.ServerStatus{
				GormesVersion: Version,
				ConfigVersion: fmt.Sprintf("%d", config.CurrentConfigVersion),
				Protocol:      navivox.ProtocolVersion,
				Features:      navivox.DefaultFeatures(),
				ActiveChannels: []string{
					navivox.PlatformName,
				},
			}}
			return navivox.NewServer(navivox.ServerOptions{Status: status}).Serve(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&stdio, "stdio", false, "serve Navivox frames over stdin/stdout")
	return cmd
}
