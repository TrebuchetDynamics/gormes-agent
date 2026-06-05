package gormescli

import (
	"io"

	apponboard "github.com/TrebuchetDynamics/gormes-agent/internal/app/onboard"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type OnboardRuntime = apponboard.Runtime

type OnboardBuildProvenance = apponboard.BuildProvenance

type OnboardStatusReportJSON = apponboard.StatusReportJSON

type OnboardFirstRunReadinessJSON = apponboard.FirstRunReadinessJSON

type OnboardAgentJSON = apponboard.AgentJSON

type OnboardBindingJSON = apponboard.BindingJSON

type OnboardWizardPlanJSON = apponboard.WizardPlanJSON

type OnboardWizardStepJSON = apponboard.WizardStepJSON

func WriteOnboardStatusJSON(out io.Writer, cfg config.Config, runtime OnboardRuntime) error {
	return apponboard.WriteStatusJSON(out, cfg, runtime)
}

func OnboardFirstRunReadinessFromPlan(plan cli.FirstRunPlan) OnboardFirstRunReadinessJSON {
	return apponboard.FirstRunReadinessFromPlan(plan)
}

func PrintOnboardStatus(out io.Writer, cfg config.Config, runtime OnboardRuntime) {
	apponboard.PrintStatus(out, cfg, runtime)
}

func PrintOnboardFirstRunReadiness(out io.Writer, plan cli.FirstRunPlan, runtime OnboardRuntime) {
	apponboard.PrintFirstRunReadiness(out, plan, runtime)
}

func WriteOnboardWizardPlanJSON(out io.Writer, cfg config.Config, nonInteractive bool, runtime OnboardRuntime) error {
	return apponboard.WriteWizardPlanJSON(out, cfg, nonInteractive, runtime)
}

func PrintOnboardWizardPlan(out io.Writer, cfg config.Config, nonInteractive bool) {
	apponboard.PrintWizardPlan(out, cfg, nonInteractive)
}

func RunOnboardWizard(in io.Reader, out io.Writer, cfg config.Config, runtime OnboardRuntime) error {
	return apponboard.RunWizard(in, out, cfg, runtime)
}

func BuildOnboardPlanFromConfig(cfg config.Config) cli.OnboardPlan {
	return apponboard.BuildPlan(cfg)
}

func PrintOnboardPlanSteps(out io.Writer, plan cli.OnboardPlan) {
	apponboard.PrintPlanSteps(out, plan)
}

func PrintOnboardStep(out io.Writer, index int, step cli.OnboardStep) {
	apponboard.PrintStep(out, index, step)
}

func DefaultOnboardStepAction(step cli.OnboardStep) string {
	return apponboard.DefaultStepAction(step)
}

func PromptOnboardAction(in io.Reader, step cli.OnboardStep, defaultAction string) (string, error) {
	return apponboard.PromptAction(in, step, defaultAction)
}

func NormalizeOnboardAction(runtime OnboardRuntime, action, defaultAction string) string {
	return apponboard.NormalizeAction(runtime, action, defaultAction)
}

func PrintOnboardReview(out io.Writer, step cli.OnboardStep) {
	apponboard.PrintReview(out, step)
}

func PrintOnboardSkip(out io.Writer, step cli.OnboardStep) {
	apponboard.PrintSkip(out, step)
}

func RunOnboardActionRowBacked(out io.Writer, step cli.OnboardStep) error {
	return apponboard.RunActionRowBacked(out, step)
}

func OnboardProviderLabel(cfg config.Config) string {
	return apponboard.ProviderLabel(cfg)
}

func OnboardProviderConfigured(cfg config.Config) bool {
	return apponboard.ProviderConfigured(cfg)
}

func OnboardAuthCommand(cfg config.Config) string {
	return apponboard.AuthCommand(cfg)
}

func OnboardSkillCounts(cfg config.Config) (local int, builtin int) {
	return apponboard.SkillCounts(cfg)
}

func CountOnboardSkills(rows []skills.SkillRow) (local int, builtin int) {
	return apponboard.CountSkills(rows)
}

func OnboardGatewayTargets(cfg config.Config) []string {
	return apponboard.GatewayTargets(cfg)
}
