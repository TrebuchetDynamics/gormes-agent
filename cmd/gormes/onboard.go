package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/spf13/cobra"
)

func newOnboardCommand() *cobra.Command {
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
				printOnboardWizardPlan(cmd, cfg, nonInteractive || !isStdinTTY())
				return nil
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
	local, builtin := onboardSkillCounts(cfg)
	plan := cli.BuildOnboardPlan(cli.OnboardPlanInput{
		Provider:       cfg.Hermes.Provider,
		Endpoint:       cfg.Hermes.Endpoint,
		Model:          cfg.Hermes.Model,
		APIKeyPresent:  cfg.Hermes.APIKey != "",
		GatewayTargets: onboardGatewayTargets(cfg),
		BrowserCDPURL:  cfg.Browser.CDPURL,
		LocalSkills:    local,
		BundledSkills:  builtin,
	})

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Gormes first-run wizard")
	if nonInteractive {
		fmt.Fprintln(out, "Mode: non-interactive plan; no prompts, browser probes, gateway starts, or dashboard launch will run.")
	} else {
		fmt.Fprintln(out, "Mode: wizard plan; prompts and external launches remain row-backed, so this view only reports the ordered flow.")
	}
	fmt.Fprintln(out)
	for i, step := range plan.Steps {
		fmt.Fprintf(out, "%d. %s: %s\n", i+1, step.Title, step.Status)
		fmt.Fprintf(out, "   %s\n", step.Detail)
		fmt.Fprintf(out, "   Next: %s\n", step.NextCommand)
		fmt.Fprintf(out, "   Skip warning: %s\n", step.SkipWarning)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Full interactive prompting remains in progress; this plan is safe to run in CI and first-run diagnostics.")
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
