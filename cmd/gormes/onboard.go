package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/spf13/cobra"
)

func newOnboardCommand() *cobra.Command {
	return newOnboardCommandWithSeams(defaultOnboardCommandSeams())
}

type onboardCommandSeams struct {
	IsTTY        func() bool
	PromptAction func(*cobra.Command, cli.OnboardStep, string) (string, error)
	RunAction    func(*cobra.Command, cli.OnboardStep) error
}

func defaultOnboardCommandSeams() onboardCommandSeams {
	return onboardCommandSeams{
		IsTTY:        isStdinTTY,
		PromptAction: promptOnboardAction,
		RunAction:    runOnboardActionRowBacked,
	}
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
	cmd := &cobra.Command{
		Use:          "onboard",
		Short:        "First-run status — see what's configured and what to do next",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			if wizard {
				if nonInteractive || !seams.IsTTY() {
					printOnboardWizardPlan(cmd, cfg, true)
					return nil
				}
				return runOnboardWizard(cmd, cfg, seams)
			}
			printOnboardStatus(cmd, cfg)
			return nil
		},
	}
	cmd.Flags().BoolVar(&wizard, "wizard", false, "show the first-run wizard plan")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "render the wizard without prompts or external launches")
	return cmd
}

func printOnboardStatus(cmd *cobra.Command, cfg config.Config) {
	skillsRoot := cfg.SkillsRoot()
	local, builtin := onboardSkillCounts(cfg)

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Gormes onboarding")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Home: %s\n", config.GormesHome())
	fmt.Fprintf(out, "Config: %s\n", config.ConfigPath())
	fmt.Fprintf(out, "Runtime skills root: %s\n", skillsRoot)
	fmt.Fprintf(out, "Runtime skills: %d local, %d bundled\n", local, builtin)
	fmt.Fprintln(out)

	providerConfigured := cfg.Hermes.Endpoint != "" && cfg.Hermes.APIKey != ""
	if !providerConfigured {
		fmt.Fprintln(out, "No provider configured yet — your agents can't run.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Quick setup:")
		fmt.Fprintln(out, "  gormes setup provider       # interactive guided setup")
		fmt.Fprintln(out)
	} else {
		fmt.Fprintf(out, "Provider: %s\n", onboardProviderLabel(cfg))
		fmt.Fprintf(out, "Endpoint: %s\n", cfg.Hermes.Endpoint)
		fmt.Fprintf(out, "Model: %s\n", cfg.Hermes.Model)
		fmt.Fprintln(out)
	}

	// Agent summary
	agentCount := len(cfg.Agents.List)
	if agentCount == 0 {
		agentCount = 1 // default main agent
	}
	defaultAgent := cfg.Agents.DefaultAgentID()
	fmt.Fprintf(out, "Agents: %d configured", agentCount)
	if defaultAgent != "" {
		fmt.Fprintf(out, " (default: %s)", defaultAgent)
	}
	fmt.Fprintln(out)

	for _, a := range cfg.Agents.List {
		marker := " "
		if strings.EqualFold(a.ID, defaultAgent) {
			marker = "★"
		}
		ws := a.Workspace
		if ws == "" {
			ws = "(default)"
		}
		fmt.Fprintf(out, "  %s %-12s workspace: %s\n", marker, a.ID, ws)
	}

	// Bindings summary
	bindingCount := len(cfg.Bindings)
	if bindingCount > 0 {
		fmt.Fprintf(out, "Bindings: %d channel→agent route(s)\n", bindingCount)
		for _, b := range cfg.Bindings {
			fmt.Fprintf(out, "  %s → %s (%s)\n", b.Match.Channel, b.AgentID, b.Match.AccountID)
		}
	} else {
		fmt.Fprintln(out, "Bindings: none — all channels route to default agent")
		fmt.Fprintln(out, "  gormes setup bindings      # assign channels to specific agents")
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Runtime skills are loaded from the skills root above plus bundled skills.")
	fmt.Fprintln(out, "Repo development skills under docs/development-skills are for agents building Gormes; they are not normal user/runtime skills unless you explicitly point GORMES_SKILLS_ROOT there.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Learning loop: manual/prompted skill capture works through skill_manage, and delegated runs can draft candidate skills. Fully automatic distill/promote/maintain is still partial.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Next steps:")
	if !providerConfigured {
		fmt.Fprintln(out, "  gormes setup provider   ← configure first")
	}
	fmt.Fprintln(out, "  gormes doctor --offline")
	fmt.Fprintln(out, "  gormes setup model")
	fmt.Fprintln(out, "  gormes auth add <provider>")
	fmt.Fprintln(out, "  gormes skills list")
	fmt.Fprintln(out, "  gormes gateway status")
	fmt.Fprintln(out, "  gormes dashboard")

	if _, ok := os.LookupEnv("GORMES_SKILLS_ROOT"); ok {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "GORMES_SKILLS_ROOT is set; runtime skill tools and `gormes skills` will use that override.")
	}
}

func printOnboardWizardPlan(cmd *cobra.Command, cfg config.Config, nonInteractive bool) {
	plan := buildOnboardPlanFromConfig(cfg)

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Gormes first-run wizard")
	if nonInteractive {
		fmt.Fprintln(out, "Mode: non-interactive plan; no prompts, browser probes, gateway starts, or dashboard launch will run.")
	} else {
		fmt.Fprintln(out, "Mode: wizard plan; prompts and external launches remain row-backed, so this view only reports the ordered flow.")
	}
	fmt.Fprintln(out)
	printOnboardPlanSteps(out, plan)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Interactive action prompting is available in a TTY; this plan is safe to run in CI and first-run diagnostics.")
}

func runOnboardWizard(cmd *cobra.Command, cfg config.Config, seams onboardCommandSeams) error {
	plan := buildOnboardPlanFromConfig(cfg)
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Gormes first-run wizard")
	fmt.Fprintln(out, "Mode: interactive action runner; selected actions delegate through safe command seams.")
	fmt.Fprintln(out)
	for i, step := range plan.Steps {
		printOnboardStep(out, i, step)
		defaultAction := defaultOnboardStepAction(step)
		fmt.Fprintf(out, "   Action for %s [run/review/skip] (%s):\n", step.Title, defaultAction)
		action, err := seams.PromptAction(cmd, step, defaultAction)
		if err != nil {
			return err
		}
		action = normalizeOnboardAction(action, defaultAction)
		switch action {
		case "run":
			if err := seams.RunAction(cmd, step); err != nil {
				return err
			}
		case "review":
			printOnboardReview(out, step)
		case "skip":
			printOnboardSkip(out, step)
		default:
			return newExitCodeError(2, fmt.Errorf("onboard_action_invalid: %s", action))
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Onboarding wizard finished.")
	return nil
}

func buildOnboardPlanFromConfig(cfg config.Config) cli.OnboardPlan {
	local, builtin := onboardSkillCounts(cfg)
	return cli.BuildOnboardPlan(cli.OnboardPlanInput{
		Provider:       cfg.Hermes.Provider,
		Endpoint:       cfg.Hermes.Endpoint,
		Model:          cfg.Hermes.Model,
		APIKeyPresent:  cfg.Hermes.APIKey != "",
		GatewayTargets: onboardGatewayTargets(cfg),
		BrowserCDPURL:  cfg.Browser.CDPURL,
		LocalSkills:    local,
		BundledSkills:  builtin,
	})
}

func printOnboardPlanSteps(out io.Writer, plan cli.OnboardPlan) {
	for i, step := range plan.Steps {
		printOnboardStep(out, i, step)
	}
}

func printOnboardStep(out io.Writer, index int, step cli.OnboardStep) {
	fmt.Fprintf(out, "%d. %s: %s\n", index+1, step.Title, step.Status)
	fmt.Fprintf(out, "   %s\n", step.Detail)
	fmt.Fprintf(out, "   Next: %s\n", step.NextCommand)
	fmt.Fprintf(out, "   Skip warning: %s\n", step.SkipWarning)
}

func defaultOnboardStepAction(step cli.OnboardStep) string {
	switch step.Status {
	case cli.OnboardStatusMissing:
		return "run"
	case cli.OnboardStatusConfigured:
		return "review"
	}
	if step.ID == cli.OnboardStepDashboard {
		return "skip"
	}
	return "review"
}

func promptOnboardAction(cmd *cobra.Command, _ cli.OnboardStep, defaultAction string) (string, error) {
	var input string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &input)
	if err != nil {
		if err.Error() == "unexpected newline" || strings.Contains(err.Error(), "expected") {
			return defaultAction, nil
		}
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func normalizeOnboardAction(action, defaultAction string) string {
	action = normalizeSetupChoice(action)
	if action == "" {
		return defaultAction
	}
	switch action {
	case "r":
		return "run"
	case "s":
		return "skip"
	case "v":
		return "review"
	default:
		return action
	}
}

func printOnboardReview(out io.Writer, step cli.OnboardStep) {
	fmt.Fprintf(out, "review: %s current=%s command=%q\n", step.ID, step.Status, step.NextCommand)
	fmt.Fprintf(out, "review_detail: %s\n", step.Detail)
}

func printOnboardSkip(out io.Writer, step cli.OnboardStep) {
	if step.Status == cli.OnboardStatusMissing {
		fmt.Fprintf(out, "skip_warning: step=%s %s\n", step.ID, step.SkipWarning)
		return
	}
	fmt.Fprintf(out, "skip: step=%s\n", step.ID)
}

func runOnboardActionRowBacked(cmd *cobra.Command, step cli.OnboardStep) error {
	fmt.Fprintf(cmd.OutOrStdout(), "onboard_action_row_backed: step=%s recommended_command=%q\n", step.ID, step.NextCommand)
	return nil
}

func onboardProviderLabel(cfg config.Config) string {
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	if provider == "" {
		return "custom"
	}
	return provider
}

func onboardSkillCounts(cfg config.Config) (local int, builtin int) {
	skillsRoot := cfg.SkillsRoot()
	rows := skills.ListInstalledSkillsFromRoots(skillsRoot, skills.BundledRoot(), skills.ListOptions{}, nil)
	return countOnboardSkills(rows)
}

func countOnboardSkills(rows []skills.SkillRow) (local int, builtin int) {
	for _, row := range rows {
		switch row.Source {
		case "builtin":
			builtin++
		default:
			local++
		}
	}
	return local, builtin
}

func onboardGatewayTargets(cfg config.Config) []string {
	var targets []string
	if cfg.Gateway.ProxyURL != "" || cfg.Gateway.ProxyKey != "" {
		targets = append(targets, "gateway proxy")
	}
	if cfg.Telegram.BotToken != "" {
		targets = append(targets, "telegram")
	}
	if cfg.Discord.Enabled() {
		targets = append(targets, "discord")
	}
	if cfg.Slack.Enabled || cfg.Slack.BotToken != "" || cfg.Slack.AppToken != "" || cfg.Slack.AllowedChannelID != "" {
		targets = append(targets, "slack")
	}
	if len(cfg.Bindings) > 0 {
		targets = append(targets, "bindings")
	}
	return targets
}
