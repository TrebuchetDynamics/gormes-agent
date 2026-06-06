package gormescli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// newRootCommand returns a root Cobra command with the doctor command
// registered using NewDoctorCommand with default options. Used by doctor
// test files moved from cmd/gormes root.
func newRootCommand() *cobra.Command {
	return newRootCommandWithOptions(defaultDoctorTestOptions())
}

// newRootCommandWithProvenance returns a root command with a doctor command
// configured with specific build provenance (version, commit).
func newRootCommandWithProvenance(version, commit string, _ bool) *cobra.Command {
	return newRootCommandWithOptions(DoctorCommandOptions{
		BuildProvenance: func() BuildProvenance {
			return BuildProvenance{Version: version, GitCommit: commit}
		},
		BuildFirstRunPlan:      testBuildFirstRunPlan,
		FirstRunGuidanceCommand: identityGuidanceCommand,
	})
}

// defaultDoctorTestOptions returns DoctorCommandOptions with default test-
// friendly function injections (BuildProvenance, BuildFirstRunPlan, guidance).
func defaultDoctorTestOptions() DoctorCommandOptions {
	return DoctorCommandOptions{
		BuildProvenance: func() BuildProvenance {
			return BuildProvenance{Version: "test-version", GitCommit: "abc123"}
		},
		BuildFirstRunPlan:      testBuildFirstRunPlan,
		FirstRunGuidanceCommand: identityGuidanceCommand,
	}
}

// identityGuidanceCommand is a test-friendly FirstRunGuidanceCommand that
// just trims whitespace (matching tuiapp.FirstRunGuidanceCommand's behavior).
func identityGuidanceCommand(command string) string {
	return strings.TrimSpace(command)
}

// testBuildFirstRunPlan wraps cli.BuildFirstRunPlan to match the
// DoctorCommandOptions.BuildFirstRunPlan signature. It mirrors the subset
// of tuiapp.BuildFirstRunPlanFromConfig that can be expressed without
// importing tuiapp (which would create a gormescli→tuiapp→gormescli cycle).
func testBuildFirstRunPlan(cfg config.Config, target cli.SetupTargetID, _ bool) cli.FirstRunPlan {
	return cli.BuildFirstRunPlan(cli.FirstRunPlanInput{
		Target:        target,
		Provider:      cfg.Hermes.Provider,
		Endpoint:      cfg.Hermes.Endpoint,
		Model:         cfg.Hermes.Model,
		APIKeyPresent: ConfiguredProviderAuthPresent(cfg),
		Channels:      testChannelStates(cfg),
	})
}

// testChannelStates mirrors tuiapp.firstRunChannelStates using only
// imports available to the gormescli package.
func testChannelStates(cfg config.Config) []cli.ChannelState {
	states := []cli.ChannelState{
		{
			Target:         cli.SetupTargetTelegram,
			Label:          "Telegram",
			Configured:     strings.TrimSpace(cfg.Telegram.BotToken) != "",
			Detail:         "Telegram channel",
			SetupCommand:   "gormes setup --quick --target telegram",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetWhatsApp,
			Label:          "WhatsApp",
			Configured:     false,
			Detail:         "WhatsApp channel",
			SetupCommand:   "gormes whatsapp --plan",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetDiscord,
			Label:          "Discord",
			Configured:     cfg.Discord.Enabled(),
			Detail:         "Discord channel",
			SetupCommand:   "gormes setup --quick --target discord",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetSlack,
			Label:          "Slack",
			Configured:     cfg.Slack.Enabled,
			Detail:         "Slack channel",
			SetupCommand:   "gormes setup --quick --target slack",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetNavivox,
			Label:          "Navivox",
			Configured:     cfg.Navivox.Port != 0 || cfg.Navivox.BindHost != "" || cfg.Navivox.AuthMode == "token",
			Detail:         "Navivox channel",
			SetupCommand:   "gormes navivox setup",
			HandoffCommand: "gormes gateway",
		},
	}
	return states
}

// newRootCommandWithOptions creates a root command using the given options.
func newRootCommandWithOptions(opts DoctorCommandOptions) *cobra.Command {
	return newRootCommandWithFactoryForTest("doctor", func() *cobra.Command {
		return NewDoctorCommand(opts)
	})
}