package main

import (
	"fmt"
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/spf13/cobra"
)

func newOnboardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "onboard",
		Short:        "Show first-run setup status and next steps",
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
		fmt.Fprintln(out, "  gormes config set hermes.endpoint https://api.openai.com/v1")
		fmt.Fprintln(out, "  gormes config set hermes.api_key sk-...")
		fmt.Fprintln(out, "  gormes config set hermes.model gpt-4o")
		fmt.Fprintln(out, "  gormes --oneshot \"hello\"     # test it")
		fmt.Fprintln(out)
	} else {
		fmt.Fprintf(out, "Provider: %s (model: %s)\n", cfg.Hermes.Endpoint, cfg.Hermes.Model)
		fmt.Fprintln(out)
	}

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
