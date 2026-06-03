package onboard

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type Runtime struct {
	Build             BuildProvenance
	Home              string
	ConfigPath        string
	BuildFirstRunPlan func(config.Config, cli.SetupTargetID, bool) cli.FirstRunPlan
	PromptAction      func(io.Reader, cli.OnboardStep, string) (string, error)
	RunAction         func(io.Writer, cli.OnboardStep) error
	NormalizeChoice   func(string) string
	FirstRunCommand   func(string) string
}

type StatusReportJSON struct {
	Build              BuildProvenance       `json:"build"`
	Home               string                `json:"home"`
	ConfigPath         string                `json:"config_path"`
	SkillsRoot         string                `json:"skills_root"`
	SkillsLocal        int                   `json:"skills_local"`
	SkillsBundled      int                   `json:"skills_bundled"`
	FirstRun           FirstRunReadinessJSON `json:"first_run"`
	ProviderConfigured bool                  `json:"provider_configured"`
	Provider           string                `json:"provider,omitempty"`
	Endpoint           string                `json:"endpoint,omitempty"`
	Model              string                `json:"model,omitempty"`
	AuthConfigured     bool                  `json:"auth_configured"`
	DefaultAgent       string                `json:"default_agent,omitempty"`
	Agents             []AgentJSON           `json:"agents,omitempty"`
	Bindings           []BindingJSON         `json:"bindings,omitempty"`
}

type FirstRunReadinessJSON struct {
	Ready       bool     `json:"ready"`
	Target      string   `json:"target"`
	NextCommand string   `json:"next_command"`
	Missing     []string `json:"missing"`
	Summary     string   `json:"summary,omitempty"`
}

type AgentJSON struct {
	ID        string `json:"id"`
	Workspace string `json:"workspace,omitempty"`
	Default   bool   `json:"default,omitempty"`
}

type BindingJSON struct {
	Channel   string `json:"channel"`
	AccountID string `json:"account_id,omitempty"`
	AgentID   string `json:"agent_id"`
}

type WizardPlanJSON struct {
	Build BuildProvenance  `json:"build"`
	Mode  string           `json:"mode"`
	Steps []WizardStepJSON `json:"steps"`
}

type WizardStepJSON struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Detail      string `json:"detail"`
	NextCommand string `json:"next_command"`
	SkipWarning string `json:"skip_warning"`
}

func WriteStatusJSON(out io.Writer, cfg config.Config, runtime Runtime) error {
	report := BuildStatusReport(cfg, runtime)
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(body))
	return err
}

func BuildStatusReport(cfg config.Config, runtime Runtime) StatusReportJSON {
	local, builtin := SkillCounts(cfg)
	defaultAgent := cfg.Agents.DefaultAgentID()
	firstRun := buildFirstRunPlan(runtime, cfg, cli.SetupTargetTerminal, false)
	report := StatusReportJSON{
		Build:              runtime.Build,
		Home:               firstNonEmpty(runtime.Home, config.GormesHome()),
		ConfigPath:         firstNonEmpty(runtime.ConfigPath, config.ConfigPath()),
		SkillsRoot:         cfg.SkillsRoot(),
		SkillsLocal:        local,
		SkillsBundled:      builtin,
		FirstRun:           FirstRunReadinessFromPlan(firstRun),
		ProviderConfigured: ProviderConfigured(cfg),
		Provider:           strings.TrimSpace(cfg.Hermes.Provider),
		Endpoint:           strings.TrimSpace(cfg.Hermes.Endpoint),
		Model:              strings.TrimSpace(cfg.Hermes.Model),
		AuthConfigured:     config.ConfiguredProviderAuthPresent(cfg),
		DefaultAgent:       defaultAgent,
	}
	for _, a := range cfg.Agents.List {
		report.Agents = append(report.Agents, AgentJSON{ID: a.ID, Workspace: a.Workspace, Default: strings.EqualFold(a.ID, defaultAgent)})
	}
	for _, b := range cfg.Bindings {
		report.Bindings = append(report.Bindings, BindingJSON{Channel: b.Match.Channel, AccountID: b.Match.AccountID, AgentID: b.AgentID})
	}
	return report
}

func FirstRunReadinessFromPlan(plan cli.FirstRunPlan) FirstRunReadinessJSON {
	missing := make([]string, 0, len(plan.MissingSteps))
	for _, step := range plan.MissingSteps {
		missing = append(missing, string(step.ID))
	}
	return FirstRunReadinessJSON{Ready: plan.Ready, Target: string(plan.Target), NextCommand: plan.NextCommand, Missing: missing, Summary: plan.Summary}
}

func PrintStatus(out io.Writer, cfg config.Config, runtime Runtime) {
	skillsRoot := cfg.SkillsRoot()
	local, builtin := SkillCounts(cfg)
	fmt.Fprintln(out, "Gormes onboarding")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Home: %s\n", firstNonEmpty(runtime.Home, config.GormesHome()))
	fmt.Fprintf(out, "Config: %s\n", firstNonEmpty(runtime.ConfigPath, config.ConfigPath()))
	fmt.Fprintf(out, "Runtime skills root: %s\n", skillsRoot)
	fmt.Fprintf(out, "Runtime skills: %d local, %d bundled\n", local, builtin)
	fmt.Fprintln(out)

	PrintFirstRunReadiness(out, buildFirstRunPlan(runtime, cfg, cli.SetupTargetTerminal, false), runtime)
	fmt.Fprintln(out)

	providerConfigured := ProviderConfigured(cfg)
	authConfigured := config.ConfiguredProviderAuthPresent(cfg)
	if !providerConfigured {
		fmt.Fprintln(out, "No provider configured yet — your agents can't run.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Quick setup:")
		fmt.Fprintln(out, "  gormes setup provider       # interactive guided setup")
		fmt.Fprintln(out)
	} else {
		fmt.Fprintf(out, "Provider: %s\n", ProviderLabel(cfg))
		fmt.Fprintf(out, "Endpoint: %s\n", cfg.Hermes.Endpoint)
		fmt.Fprintf(out, "Model: %s\n", cfg.Hermes.Model)
		if authConfigured {
			fmt.Fprintln(out, "Auth: configured")
		} else {
			fmt.Fprintf(out, "Auth: missing — run `%s` or configure a provider API key.\n", AuthCommand(cfg))
		}
		fmt.Fprintln(out)
	}

	agentCount := len(cfg.Agents.List)
	if agentCount == 0 {
		agentCount = 1
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
	fmt.Fprintln(out, "Repo development skills under development-skills are for agents building Gormes; they are not normal user/runtime skills unless you explicitly point GORMES_SKILLS_ROOT there.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Learning loop: manual/prompted skill capture works through skill_manage, and delegated runs can draft candidate skills. Fully automatic distill/promote/maintain is still partial.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Next steps:")
	if !providerConfigured {
		fmt.Fprintln(out, "  gormes setup provider   ← configure first")
	} else if !authConfigured {
		fmt.Fprintf(out, "  %s   ← configure auth\n", AuthCommand(cfg))
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

func PrintFirstRunReadiness(out io.Writer, plan cli.FirstRunPlan, runtime Runtime) {
	status := "ready"
	if !plan.Ready {
		status = "setup needed"
	}
	fmt.Fprintf(out, "First-run readiness: %s\n", status)
	if plan.Summary != "" {
		fmt.Fprintf(out, "%s\n", plan.Summary)
	}
	for _, step := range plan.MissingSteps {
		if step.Detail == "" {
			continue
		}
		fmt.Fprintf(out, "%s: %s\n", step.Label, step.Detail)
	}
	if command := firstRunCommand(runtime, plan.NextCommand); command != "" {
		fmt.Fprintf(out, "Next: %s\n", command)
	}
}

func WriteWizardPlanJSON(out io.Writer, cfg config.Config, nonInteractive bool, runtime Runtime) error {
	report := BuildWizardPlanReport(cfg, nonInteractive, runtime)
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(body))
	return err
}

func BuildWizardPlanReport(cfg config.Config, nonInteractive bool, runtime Runtime) WizardPlanJSON {
	plan := BuildPlan(cfg)
	mode := "interactive"
	if nonInteractive {
		mode = "non-interactive"
	}
	report := WizardPlanJSON{Build: runtime.Build, Mode: mode, Steps: make([]WizardStepJSON, 0, len(plan.Steps))}
	for _, step := range plan.Steps {
		report.Steps = append(report.Steps, WizardStepJSON{ID: step.ID, Title: step.Title, Status: step.Status, Detail: step.Detail, NextCommand: step.NextCommand, SkipWarning: step.SkipWarning})
	}
	return report
}

func PrintWizardPlan(out io.Writer, cfg config.Config, nonInteractive bool) {
	fmt.Fprintln(out, "Gormes first-run wizard")
	if nonInteractive {
		fmt.Fprintln(out, "Mode: non-interactive plan; no prompts, browser probes, gateway starts, or dashboard launch will run.")
	} else {
		fmt.Fprintln(out, "Mode: wizard plan; prompts and external launches remain row-backed, so this view only reports the ordered flow.")
	}
	fmt.Fprintln(out)
	PrintPlanSteps(out, BuildPlan(cfg))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Interactive action prompting is available in a TTY; this plan is safe to run in CI and first-run diagnostics.")
}

func RunWizard(in io.Reader, out io.Writer, cfg config.Config, runtime Runtime) error {
	plan := BuildPlan(cfg)
	fmt.Fprintln(out, "Gormes first-run wizard")
	fmt.Fprintln(out, "Mode: interactive action runner; selected actions delegate through safe command seams.")
	fmt.Fprintln(out)
	for i, step := range plan.Steps {
		PrintStep(out, i, step)
		defaultAction := DefaultStepAction(step)
		fmt.Fprintf(out, "   Action for %s [run/review/skip] (%s):\n", step.Title, defaultAction)
		action, err := promptAction(runtime, in, step, defaultAction)
		if err != nil {
			return err
		}
		action = NormalizeAction(runtime, action, defaultAction)
		switch action {
		case "run":
			if err := runAction(runtime, out, step); err != nil {
				return err
			}
		case "review":
			PrintReview(out, step)
		case "skip":
			PrintSkip(out, step)
		default:
			return fmt.Errorf("onboard_action_invalid: %s", action)
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Onboarding wizard finished.")
	return nil
}

func BuildPlan(cfg config.Config) cli.OnboardPlan {
	local, builtin := SkillCounts(cfg)
	return cli.BuildOnboardPlan(cli.OnboardPlanInput{Provider: cfg.Hermes.Provider, Endpoint: cfg.Hermes.Endpoint, Model: cfg.Hermes.Model, APIKeyPresent: config.ConfiguredProviderAuthPresent(cfg), GatewayTargets: GatewayTargets(cfg), BrowserCDPURL: cfg.Browser.CDPURL, LocalSkills: local, BundledSkills: builtin})
}

func PrintPlanSteps(out io.Writer, plan cli.OnboardPlan) {
	for i, step := range plan.Steps {
		PrintStep(out, i, step)
	}
}

func PrintStep(out io.Writer, index int, step cli.OnboardStep) {
	fmt.Fprintf(out, "%d. %s: %s\n", index+1, step.Title, step.Status)
	fmt.Fprintf(out, "   %s\n", step.Detail)
	fmt.Fprintf(out, "   Next: %s\n", step.NextCommand)
	fmt.Fprintf(out, "   Skip warning: %s\n", step.SkipWarning)
}

func DefaultStepAction(step cli.OnboardStep) string {
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

func PromptAction(in io.Reader, _ cli.OnboardStep, defaultAction string) (string, error) {
	var input string
	_, err := fmt.Fscanln(in, &input)
	if err != nil {
		if err.Error() == "unexpected newline" || strings.Contains(err.Error(), "expected") {
			return defaultAction, nil
		}
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func NormalizeAction(runtime Runtime, action, defaultAction string) string {
	if runtime.NormalizeChoice != nil {
		action = runtime.NormalizeChoice(action)
	} else {
		action = strings.TrimSpace(strings.ToLower(action))
	}
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

func PrintReview(out io.Writer, step cli.OnboardStep) {
	fmt.Fprintf(out, "review: %s current=%s command=%q\n", step.ID, step.Status, step.NextCommand)
	fmt.Fprintf(out, "review_detail: %s\n", step.Detail)
}

func PrintSkip(out io.Writer, step cli.OnboardStep) {
	if step.Status == cli.OnboardStatusMissing {
		fmt.Fprintf(out, "skip_warning: step=%s %s\n", step.ID, step.SkipWarning)
		return
	}
	fmt.Fprintf(out, "skip: step=%s\n", step.ID)
}

func RunActionRowBacked(out io.Writer, step cli.OnboardStep) error {
	fmt.Fprintf(out, "onboard_action_row_backed: step=%s recommended_command=%q\n", step.ID, step.NextCommand)
	return nil
}

func ProviderLabel(cfg config.Config) string {
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	if provider == "" {
		return "custom"
	}
	return provider
}

func ProviderConfigured(cfg config.Config) bool {
	return strings.TrimSpace(cfg.Hermes.Endpoint) != "" && strings.TrimSpace(cfg.Hermes.Model) != ""
}

func AuthCommand(cfg config.Config) string {
	provider := strings.TrimSpace(cfg.Hermes.Provider)
	if provider == "" {
		return "gormes setup provider"
	}
	return "gormes auth add " + provider
}

func SkillCounts(cfg config.Config) (local int, builtin int) {
	rows := skills.ListInstalledSkillsFromRoots(cfg.SkillsRoot(), skills.BundledRoot(), skills.ListOptions{}, nil)
	return CountSkills(rows)
}

func CountSkills(rows []skills.SkillRow) (local int, builtin int) {
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

func GatewayTargets(cfg config.Config) []string {
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

func buildFirstRunPlan(runtime Runtime, cfg config.Config, target cli.SetupTargetID, interactive bool) cli.FirstRunPlan {
	if runtime.BuildFirstRunPlan != nil {
		return runtime.BuildFirstRunPlan(cfg, target, interactive)
	}
	return cli.BuildFirstRunPlan(cli.FirstRunPlanInput{Interactive: interactive, Provider: cfg.Hermes.Provider, Endpoint: cfg.Hermes.Endpoint, Model: cfg.Hermes.Model, APIKeyPresent: config.ConfiguredProviderAuthPresent(cfg), Target: target})
}

func firstRunCommand(runtime Runtime, command string) string {
	if runtime.FirstRunCommand != nil {
		return runtime.FirstRunCommand(command)
	}
	return strings.TrimSpace(command)
}

func promptAction(runtime Runtime, in io.Reader, step cli.OnboardStep, defaultAction string) (string, error) {
	if runtime.PromptAction != nil {
		return runtime.PromptAction(in, step, defaultAction)
	}
	return PromptAction(in, step, defaultAction)
}

func runAction(runtime Runtime, out io.Writer, step cli.OnboardStep) error {
	if runtime.RunAction != nil {
		return runtime.RunAction(out, step)
	}
	return RunActionRowBacked(out, step)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
