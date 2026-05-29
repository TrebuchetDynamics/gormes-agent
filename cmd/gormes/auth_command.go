package main

import (
	"github.com/spf13/cobra"

	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
)

func newAuthCommand() *cobra.Command {
	return providermodule.NewAuthCommandWithSeams(defaultAuthCommandSeams(), providerCommandOptions())
}

func defaultAuthCommandSeams() providermodule.AuthSeams {
	return providermodule.AuthSeams{
		RunBare:                    runAuthBareCommand,
		RunAdd:                     runAuthAddCommandFromProvider,
		RunList:                    runAuthListCommand,
		RunRemove:                  runAuthRemoveCommand,
		RunReset:                   runAuthResetCommand,
		RunStatus:                  runAuthStatusCommand,
		RunLogout:                  runAuthLogoutCommand,
		EmitJSONSubcommandRequired: emitJSONSubcommandRequired,
		EmitJSONInputError:         emitJSONInputError,
	}
}

func runAuthAddCommandFromProvider(cmd *cobra.Command, opts providermodule.AuthAddOptions) error {
	return runAuthAddCommand(cmd, authAddOptions{
		Provider:                    opts.Provider,
		AuthType:                    opts.AuthType,
		Label:                       opts.Label,
		APIKey:                      opts.APIKey,
		InferenceURL:                opts.InferenceURL,
		PortalURL:                   opts.PortalURL,
		ClientID:                    opts.ClientID,
		Scope:                       opts.Scope,
		NoBrowser:                   opts.NoBrowser,
		Timeout:                     opts.Timeout,
		Insecure:                    opts.Insecure,
		CABundle:                    opts.CABundle,
		EmergencyImportFromCodexCLI: opts.EmergencyImportFromCodexCLI,
	})
}
