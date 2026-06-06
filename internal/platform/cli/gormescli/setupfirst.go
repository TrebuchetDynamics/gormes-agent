package gormescli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setupfirst"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type SetupFirstRunAction = setupfirst.Action

type SetupFirstRunOption = setupfirst.Option

type SetupFirstRunSourceSeams = setupfirst.SourceSeams

const (
	SetupFirstRunActionQuick           = setupfirst.ActionQuick
	SetupFirstRunActionFull            = setupfirst.ActionFull
	SetupFirstRunActionMigrateHermes   = setupfirst.ActionMigrateHermes
	SetupFirstRunActionMigrateOpenClaw = setupfirst.ActionMigrateOpenClaw
)

type SetupQuickSeams struct {
	ChooseSetupTarget    func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error)
	RunSetupProvider     func(*cobra.Command, bool) error
	RunProviderLiveTest  func(*cobra.Command) error
	LoadCurrentModel     func() (cli.ProviderModel, error)
	RunSetupModelSection func(*cobra.Command, bool) error
	RunWhatsAppSetup     func(*cobra.Command) error
	RunTelegramSetup     func(*cobra.Command) error
	RunGatewayPlatform   func(*cobra.Command, string) error
	LaunchChat           func(*cobra.Command) error

	DetectHermesMigrationSource   func() string
	DetectOpenClawMigrationSource func() string
	NewExitCodeError              func(int, error) error
}

type SetupTargetPromptOptions struct {
	PromptString       func(*cobra.Command, string, string) (string, error)
	NewExitCodeError   func(int, error) error
	PickShouldFallback func(error) bool
}

func FirstRunSetupOptions(seams SetupFirstRunSourceSeams) []SetupFirstRunOption {
	return setupfirst.FirstRunSetupOptions(seams)
}

func RunSetupQuick(cmd *cobra.Command, seams SetupQuickSeams, nonInteractive bool, requestedTarget cli.SetupTargetID) error {
	runtime := setupfirst.QuickRuntime{
		Out:             cmd.OutOrStdout(),
		NonInteractive:  nonInteractive,
		RequestedTarget: requestedTarget,
		ChooseSetupTarget: func(targets []cli.SetupTargetOption, defaultIndex int) (cli.SetupTargetID, error) {
			if seams.ChooseSetupTarget == nil {
				return "", fmt.Errorf("setup target chooser unavailable")
			}
			return seams.ChooseSetupTarget(cmd, targets, defaultIndex)
		},
		RunSetupProvider: func(nonInteractive bool) error {
			if seams.RunSetupProvider == nil {
				return fmt.Errorf("setup provider seam unavailable")
			}
			return seams.RunSetupProvider(cmd, nonInteractive)
		},
		RunProviderLiveTest: func() error {
			if seams.RunProviderLiveTest == nil {
				return fmt.Errorf("provider live test seam unavailable")
			}
			return seams.RunProviderLiveTest(cmd)
		},
		LoadCurrentModel: seams.LoadCurrentModel,
		RunSetupModelSection: func(nonInteractive bool) error {
			if seams.RunSetupModelSection == nil {
				return fmt.Errorf("setup model seam unavailable")
			}
			return seams.RunSetupModelSection(cmd, nonInteractive)
		},
		RunWhatsAppSetup: func() error {
			if seams.RunWhatsAppSetup == nil {
				return fmt.Errorf("whatsapp setup seam unavailable")
			}
			return seams.RunWhatsAppSetup(cmd)
		},
		RunTelegramSetup: func() error {
			if seams.RunTelegramSetup == nil {
				return fmt.Errorf("telegram setup seam unavailable")
			}
			return seams.RunTelegramSetup(cmd)
		},
		RunGatewayPlatform: func(platform string) error {
			if seams.RunGatewayPlatform == nil {
				return fmt.Errorf("gateway platform setup seam unavailable")
			}
			return seams.RunGatewayPlatform(cmd, platform)
		},
		LaunchChat: func() error {
			if seams.LaunchChat == nil {
				return fmt.Errorf("launch chat seam unavailable")
			}
			return seams.LaunchChat(cmd)
		},
		BuildFirstRunPlan: func(cfg config.Config, target cli.SetupTargetID, interactive bool) cli.FirstRunPlan {
			return setupfirst.BuildFirstRunPlanFromConfig(cfg, target, interactive, setupfirst.SourceSeams{
				DetectHermesMigrationSource:   seams.DetectHermesMigrationSource,
				DetectOpenClawMigrationSource: seams.DetectOpenClawMigrationSource,
			})
		},
		SetupNavivoxProviderCommand: SetupNavivoxProviderSetupCommand,
		NewExitCodeError:            seams.NewExitCodeError,
	}
	return setupfirst.RunQuick(runtime)
}

func PrintQuickSetupTargets(cmd *cobra.Command, targets []cli.SetupTargetOption) {
	setupfirst.PrintQuickSetupTargets(cmd.OutOrStdout(), targets)
}

func PromptSetupTarget(cmd *cobra.Command, targets []cli.SetupTargetOption, defaultOption int, opts SetupTargetPromptOptions) (cli.SetupTargetID, error) {
	if len(targets) == 0 {
		return cli.SetupTargetTerminal, nil
	}
	if defaultOption < 0 || defaultOption >= len(targets) {
		defaultOption = 0
	}
	if stdin, ok := cmd.InOrStdin().(*os.File); ok {
		selected, err := RunTUIPick(cmd.Context(), stdin, cmd.OutOrStdout(), "Where should quick setup take you first?", SetupTargetPickerChoices(targets), string(targets[defaultOption].ID))
		if err == nil && selected != "" {
			return cli.SetupTargetID(selected), nil
		}
		pickShouldFallback := opts.PickShouldFallback
		if pickShouldFallback == nil {
			pickShouldFallback = TUIPickShouldFallback
		}
		if err != nil && !pickShouldFallback(err) {
			return "", err
		}
	}

	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Where should quick setup take you first?")
	fmt.Fprintln(out)
	for i, target := range targets {
		marker := "( )"
		label := target.Label
		if i == defaultOption {
			marker = "(*)"
			label = cli.Bold(out, label)
		}
		fmt.Fprintf(out, "  %d. %s %s\n", i+1, marker, label)
	}
	fmt.Fprintln(out)

	defaultText := strconv.Itoa(defaultOption + 1)
	promptString := opts.PromptString
	if promptString == nil {
		return "", fmt.Errorf("setup target prompt string seam unavailable")
	}
	answer, err := promptString(cmd, fmt.Sprintf("Select target [%s]: ", defaultText), defaultText)
	if err != nil {
		return "", err
	}
	answer = strings.ToLower(strings.TrimSpace(cli.StripANSI(answer)))
	if answer == "" {
		return targets[defaultOption].ID, nil
	}
	if n, err := strconv.Atoi(answer); err == nil && n >= 1 && n <= len(targets) {
		return targets[n-1].ID, nil
	}
	for _, target := range targets {
		if answer == string(target.ID) || strings.Contains(strings.ToLower(target.Label), answer) {
			return target.ID, nil
		}
	}
	err = fmt.Errorf("setup_target_invalid_selection: %s", answer)
	if opts.NewExitCodeError != nil {
		return "", opts.NewExitCodeError(2, err)
	}
	return "", err
}

func SetupTargetPickerChoices(targets []cli.SetupTargetOption) []TUIPickChoice {
	choices := make([]TUIPickChoice, len(targets))
	for i, target := range targets {
		choices[i] = TUIPickChoice{ID: string(target.ID), Label: target.Label}
	}
	return choices
}

func RunSetupProviderLiveTest(ctx context.Context) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return err
	}
	client, err := NewProviderHTTPClient(cfg, cfg.Hermes.Provider)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return client.Health(ctx)
}

func NormalizeSetupQuickTarget(target cli.SetupTargetID) cli.SetupTargetID {
	return setupfirst.NormalizeQuickTarget(target)
}

func ParseSetupQuickTarget(target string) (cli.SetupTargetID, bool) {
	return setupfirst.ParseQuickTarget(target)
}

func IsSetupQuickChannelTarget(target cli.SetupTargetID) bool {
	return setupfirst.IsQuickChannelTarget(target)
}

func SetupQuickMissingCore(plan cli.FirstRunPlan) bool {
	return setupfirst.QuickMissingCore(plan)
}

func SetupQuickNextCoreSetupCommand(plan cli.FirstRunPlan) string {
	return setupfirst.NextCoreSetupCommand(plan)
}

func SetupTargetDefaultIndex(targets []cli.SetupTargetOption, target cli.SetupTargetID) int {
	return setupfirst.TargetDefaultIndex(targets, target)
}
