package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/spf13/cobra"
)

func newOnboardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "onboard",
		Short:        "First-run status — see what's configured and what to do next",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			printOnboardStatus(cmd, cfg)
			return nil
		},
	}
	return cmd
}

func printOnboardStatus(cmd *cobra.Command, cfg config.Config) {
	skillsRoot := cfg.SkillsRoot()
	rows := skills.ListInstalledSkillsFromRoots(skillsRoot, skills.BundledRoot(), skills.ListOptions{}, nil)
	local, builtin := countOnboardSkills(rows)

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

func onboardProviderLabel(cfg config.Config) string {
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	if provider == "" {
		return "custom"
	}
	return provider
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
