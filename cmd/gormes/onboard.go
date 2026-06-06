package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	tuiapp "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/tuiapp"
)

// onboardStatusReportJSON is the internal first-run readiness JSON shape.
// Public fleet automation should query doctor readiness JSON; tests keep
// this helper shape stable without registering a public command.
// Build provenance leads — same convention as the rest of the `--json`
// arc. Secrets stay out: only `auth_configured` signals key presence.
type onboardStatusReportJSON = gormescli.OnboardStatusReportJSON

type onboardFirstRunReadinessJSON = gormescli.OnboardFirstRunReadinessJSON

type onboardAgentJSON = gormescli.OnboardAgentJSON

type onboardBindingJSON = gormescli.OnboardBindingJSON

type onboardCommandSeams struct {
	IsTTY        func() bool
	PromptAction func(*cobra.Command, cli.OnboardStep, string) (string, error)
	RunAction    func(*cobra.Command, cli.OnboardStep) error
}

func defaultOnboardCommandSeams() onboardCommandSeams {
	return onboardCommandSeams{IsTTY: isStdinTTY, PromptAction: promptOnboardAction, RunAction: runOnboardActionRowBacked}
}

func newOnboardCommandWithSeams(seams onboardCommandSeams) *cobra.Command {
	if seams.IsTTY == nil {
		seams.IsTTY = isStdinTTY
	}
	if seams.PromptAction == nil {
		seams.PromptAction = promptOnboardAction
	}
	if seams.RunAction == nil {
		seams.RunAction = runOnboardActionRowBacked
	}

	var wizard bool
	var nonInteractive bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "onboard",
		Short:        "Use gormes setup",
		Hidden:       true,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			runtime := onboardRuntime(cmd, seams)
			if asJSON {
				if wizard {
					return writeOnboardWizardPlanJSON(cmd, cfg, nonInteractive || !seams.IsTTY())
				}
				return writeOnboardStatusJSON(cmd, cfg)
			}
			if wizard {
				if nonInteractive || !seams.IsTTY() {
					gormescli.PrintOnboardWizardPlan(cmd.OutOrStdout(), cfg, true)
					return nil
				}
				return runOnboardWizardWithRuntime(cmd, cfg, runtime)
			}
			gormescli.PrintOnboardStatus(cmd.OutOrStdout(), cfg, runtime)
			return nil
		},
	}
	cmd.Flags().BoolVar(&wizard, "wizard", false, "show the first-run wizard plan")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "render the wizard without prompts or external launches")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, home, config_path, provider, auth_configured, agents, bindings, ...}")
	return cmd
}

func onboardRuntime(cmd *cobra.Command, seams onboardCommandSeams) gormescli.OnboardRuntime {
	build := newBuildProvenance()
	runtime := gormescli.OnboardRuntime{
		Build:             gormescli.OnboardBuildProvenance{Version: build.Version, GitCommit: build.GitCommit},
		Home:              config.GormesHome(),
		ConfigPath:        config.ConfigPath(),
		BuildFirstRunPlan: tuiapp.BuildFirstRunPlanFromConfig,
		NormalizeChoice:   normalizeSetupChoice,
		FirstRunCommand:   tuiapp.FirstRunGuidanceCommand,
	}
	if seams.PromptAction != nil {
		runtime.PromptAction = func(_ io.Reader, step cli.OnboardStep, defaultAction string) (string, error) {
			return seams.PromptAction(cmd, step, defaultAction)
		}
	}
	if seams.RunAction != nil {
		runtime.RunAction = func(_ io.Writer, step cli.OnboardStep) error {
			return seams.RunAction(cmd, step)
		}
	}
	return runtime
}

func writeOnboardStatusJSON(cmd *cobra.Command, cfg config.Config) error {
	return gormescli.WriteOnboardStatusJSON(cmd.OutOrStdout(), cfg, onboardRuntime(cmd, defaultOnboardCommandSeams()))
}

func onboardFirstRunReadinessFromPlan(plan cli.FirstRunPlan) onboardFirstRunReadinessJSON {
	return gormescli.OnboardFirstRunReadinessFromPlan(plan)
}

func printOnboardStatus(cmd *cobra.Command, cfg config.Config) {
	gormescli.PrintOnboardStatus(cmd.OutOrStdout(), cfg, onboardRuntime(cmd, defaultOnboardCommandSeams()))
}

func printOnboardFirstRunReadiness(out io.Writer, plan cli.FirstRunPlan) {
	gormescli.PrintOnboardFirstRunReadiness(out, plan, gormescli.OnboardRuntime{FirstRunCommand: tuiapp.FirstRunGuidanceCommand})
}

// onboardWizardPlanJSON is the internal onboarding wizard-plan JSON shape.
// The structured plan mirrors the text ladder field-for-field so setup
// integrations can render the same step ordering, status, next-command,
// and skip-warning copy.
type onboardWizardPlanJSON = gormescli.OnboardWizardPlanJSON

type onboardWizardStepJSON = gormescli.OnboardWizardStepJSON

func writeOnboardWizardPlanJSON(cmd *cobra.Command, cfg config.Config, nonInteractive bool) error {
	return gormescli.WriteOnboardWizardPlanJSON(cmd.OutOrStdout(), cfg, nonInteractive, onboardRuntime(cmd, defaultOnboardCommandSeams()))
}

func printOnboardWizardPlan(cmd *cobra.Command, cfg config.Config, nonInteractive bool) {
	gormescli.PrintOnboardWizardPlan(cmd.OutOrStdout(), cfg, nonInteractive)
}

func runOnboardWizard(cmd *cobra.Command, cfg config.Config, seams onboardCommandSeams) error {
	return runOnboardWizardWithRuntime(cmd, cfg, onboardRuntime(cmd, seams))
}

func runOnboardWizardWithRuntime(cmd *cobra.Command, cfg config.Config, runtime gormescli.OnboardRuntime) error {
	err := gormescli.RunOnboardWizard(cmd.InOrStdin(), cmd.OutOrStdout(), cfg, runtime)
	if err != nil && strings.HasPrefix(err.Error(), "onboard_action_invalid: ") {
		return newExitCodeError(2, err)
	}
	return err
}

func buildOnboardPlanFromConfig(cfg config.Config) cli.OnboardPlan {
	return gormescli.BuildOnboardPlanFromConfig(cfg)
}

func printOnboardPlanSteps(out io.Writer, plan cli.OnboardPlan) {
	gormescli.PrintOnboardPlanSteps(out, plan)
}

func printOnboardStep(out io.Writer, index int, step cli.OnboardStep) {
	gormescli.PrintOnboardStep(out, index, step)
}

func defaultOnboardStepAction(step cli.OnboardStep) string {
	return gormescli.DefaultOnboardStepAction(step)
}

func promptOnboardAction(cmd *cobra.Command, step cli.OnboardStep, defaultAction string) (string, error) {
	return gormescli.PromptOnboardAction(cmd.InOrStdin(), step, defaultAction)
}

func normalizeOnboardAction(action, defaultAction string) string {
	return gormescli.NormalizeOnboardAction(gormescli.OnboardRuntime{NormalizeChoice: normalizeSetupChoice}, action, defaultAction)
}

func printOnboardReview(out io.Writer, step cli.OnboardStep) {
	gormescli.PrintOnboardReview(out, step)
}

func printOnboardSkip(out io.Writer, step cli.OnboardStep) {
	gormescli.PrintOnboardSkip(out, step)
}

func runOnboardActionRowBacked(cmd *cobra.Command, step cli.OnboardStep) error {
	return gormescli.RunOnboardActionRowBacked(cmd.OutOrStdout(), step)
}

func onboardProviderLabel(cfg config.Config) string {
	return gormescli.OnboardProviderLabel(cfg)
}

func onboardProviderConfigured(cfg config.Config) bool {
	return gormescli.OnboardProviderConfigured(cfg)
}

func onboardAuthCommand(cfg config.Config) string {
	return gormescli.OnboardAuthCommand(cfg)
}

func onboardSkillCounts(cfg config.Config) (local int, builtin int) {
	return gormescli.OnboardSkillCounts(cfg)
}

func countOnboardSkills(rows []skills.SkillRow) (local int, builtin int) {
	return gormescli.CountOnboardSkills(rows)
}

func onboardGatewayTargets(cfg config.Config) []string {
	return gormescli.OnboardGatewayTargets(cfg)
}

func onboardInvalidActionError(action string) error {
	return newExitCodeError(2, fmt.Errorf("onboard_action_invalid: %s", action))
}
