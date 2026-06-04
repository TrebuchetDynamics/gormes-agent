package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	telegram "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/acp"
	gormesruntime "github.com/TrebuchetDynamics/gormes-agent/internal/runtime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	tuilocal "github.com/TrebuchetDynamics/gormes-agent/internal/tui/local"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// newDoctorCommand returns a fresh doctor command. Constructor pattern
// (rather than a package-level var with init-time flag registration)
// avoids cross-test flag-value contamination on the shared cobra
// FlagSet — each newRootCommand() builds an independent instance.
func newDoctorCommand() *cobra.Command {
	cmd := buildDoctorCmd()
	cmd.Flags().Bool("offline", false, "skip the provider health check and validate local runtime checks")
	cmd.Flags().Bool("fix", false, "auto-remediate the source-backed fixable issues (config schema migrate), then re-report fixed vs still-manual")
	cmd.Flags().Bool("json", false, "emit a machine-readable {checks: [...]} JSON document (suitable for fleet-health monitoring)")
	cmd.Flags().String("target", "", "report first-run readiness for a setup target (terminal, telegram, whatsapp, discord, slack, navivox)")
	cmd.Flags().String("ack", "", "acknowledge (dismiss) a security advisory by id; the ack is recorded under ~/.gormes and survives restart")
	return cmd
}

// doctorReporter funnels each CheckResult through either the human
// streaming surface (text mode) or a buffered slice rendered at the
// end (JSON mode). Calling sites stay branch-free: every check uses
// the same Add() entry point.
type doctorReporter struct {
	w         io.Writer
	asJSON    bool
	fix       bool
	fixApply  func(class string) doctor.DoctorFixOutcome
	collected []doctor.CheckResult
	failed    bool
	target    *doctorTargetReadinessJSON
}

func (r *doctorReporter) Add(c doctor.CheckResult) {
	if c.Status == doctor.StatusFail {
		r.failed = true
	}
	// Human mode no longer streams each check flat at Add() time: checks are
	// accumulated and rendered grouped under `◆ <Section>` headers in
	// Finalize (upstream hermes-doctor parity). JSON mode is unaffected.
	r.collected = append(r.collected, c)
}

// doctorReportJSON is the wire shape for `gormes doctor --json`.
// Field order matters for consumer rendering — build provenance and
// summary fields lead, per-check array follows. Mirrors the
// convention update --json / status --json / restore --list --json
// use.
type doctorReportJSON struct {
	Build  buildProvenanceJSON        `json:"build"`
	Failed bool                       `json:"failed"`
	Target *doctorTargetReadinessJSON `json:"target,omitempty"`
	Checks []doctor.CheckResult       `json:"checks"`
}

type doctorTargetReadinessJSON struct {
	Name        string   `json:"name"`
	Ready       bool     `json:"ready"`
	Summary     string   `json:"summary"`
	NextCommand string   `json:"next_command"`
	Missing     []string `json:"missing"`
}

func (r *doctorReporter) Finalize() error {
	if !r.asJSON {
		style := doctor.RenderStyleForWriter(r.w)
		fmt.Fprint(r.w, doctor.RenderDoctorHeaderStyled("🩺 Gormes Doctor", style))
		fmt.Fprint(r.w, doctor.RenderDoctorStatusSummary(r.collected, style))
		fmt.Fprint(r.w, doctor.RenderSectionedReportWithStyle(r.collected, style))
		issues := doctor.CollectDoctorIssues(r.collected)
		fmt.Fprint(r.w, doctor.RenderDoctorIssuesSummary(issues))
		if r.fix {
			apply := r.fixApply
			if apply == nil {
				apply = doctorApplyFix
			}
			outcomes := doctor.RunDoctorFix(apply)
			fmt.Fprint(r.w, doctor.RenderDoctorFixReport(outcomes, residualManualIssues(issues)))
		}
		return nil
	}
	checks := r.collected
	if checks == nil {
		checks = []doctor.CheckResult{}
	}
	body, err := json.MarshalIndent(doctorReportJSON{
		Build:  newBuildProvenance(),
		Failed: r.failed,
		Target: r.target,
		Checks: checks,
	}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(r.w, string(body))
	return err
}

var doctorGitHubAuthRunner = doctor.DefaultGitHubAuthStatusRunner
var doctorNewTelegramClient = func(token string) error {
	_, err := telegram.NewRealClient(token)
	return err
}
var doctorNewDiscordSession = func(token string) error {
	return gormescli.CheckDiscordSession(token)
}

// doctorApplyFix performs the real, source-backed remediation for one
// auto-fixable class. The config-version class is migrated through the
// existing config.MigrateConfigFile seam — a pure local file op, so it
// runs identically with or without `--offline` (no network remediation).
// Wording is Gormes-owned (`gormes config migrate`, never `hermes`).
func doctorApplyFix(class string) doctor.DoctorFixOutcome {
	switch class {
	case doctor.FixClassConfigVersion:
		res, err := config.MigrateConfigFile(config.ConfigPath())
		if err != nil {
			return doctor.DoctorFixOutcome{
				Class:  class,
				Name:   "config schema",
				Detail: fmt.Sprintf("%v; fix manually with `gormes config migrate`", err),
			}
		}
		if res.Wrote {
			// FromVersion == ToVersion + Wrote means the current-version
			// file was missing default v2 tables. Reserve the "migrated
			// vX→vY" wording for an actual version increase.
			detail := fmt.Sprintf("updated config_version=%d defaults", res.ToVersion)
			if res.FromVersion != res.ToVersion {
				detail = fmt.Sprintf("migrated config_version v%d→v%d", res.FromVersion, res.ToVersion)
			}
			return doctor.DoctorFixOutcome{
				Class:  class,
				Name:   "config schema",
				Fixed:  true,
				Detail: detail,
			}
		}
		return doctor.DoctorFixOutcome{
			Class:     class,
			Name:      "config schema",
			AlreadyOK: true,
			Detail:    fmt.Sprintf("config_version v%d", res.ToVersion),
		}
	}
	return doctor.DoctorFixOutcome{Class: class, Name: class, AlreadyOK: true}
}

// residualManualIssues are the collected WARN/FAILs that `--fix` does not
// auto-remediate (non-fixable, or a fixable class not in the auto-fix
// registry such as the guided published-symlink repair). They are
// reported as still-manual rather than silently dropped.
func residualManualIssues(issues []doctor.DoctorIssue) []doctor.DoctorIssue {
	auto := make(map[string]bool)
	for _, c := range doctor.AutoFixableClasses() {
		auto[c] = true
	}
	manual := make([]doctor.DoctorIssue, 0, len(issues))
	for _, is := range issues {
		if auto[is.Class] {
			continue
		}
		manual = append(manual, is)
	}
	return manual
}

func buildDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "doctor",
		Short:         "Verify Gormes runtime: provider readiness + built-in tools",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			errOut := cmd.ErrOrStderr()
			asJSON, _ := cmd.Flags().GetBool("json")
			fix, _ := cmd.Flags().GetBool("fix")
			reporter := &doctorReporter{w: out, asJSON: asJSON, fix: fix}
			exitCode := 0
			markFailure := func(code int) {
				if code > exitCode {
					exitCode = code
				}
			}
			// Defer Finalize so JSON output renders even on early-return
			// failure paths. Errors from finalize are best-effort logged.
			defer func() {
				if err := reporter.Finalize(); err != nil {
					fmt.Fprintf(errOut, "doctor: emit json: %v\n", err)
				}
			}()

			reporter.Add(doctorBuildIdentityStatus())

			// ◆ Security Advisories renders FIRST (most urgent). Pure local
			// FS over the Gormes-owned catalog + ~/.gormes ack store, so it
			// is identical under --offline. --ack <id> persists a dismissal
			// here before the section is rendered.
			ackID, _ := cmd.Flags().GetString("ack")
			reporter.Add(doctorSecurityAdvisoriesStatus(strings.TrimSpace(ackID)))

			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}

			offline, _ := cmd.Flags().GetBool("offline")
			var target cli.SetupTargetID
			targetName, _ := cmd.Flags().GetString("target")
			if strings.TrimSpace(targetName) != "" {
				var ok bool
				target, ok = doctorSetupTargetFromString(targetName)
				if !ok {
					reporter.Add(doctor.CheckResult{
						Name:    "target readiness",
						Status:  doctor.StatusFail,
						Summary: doctorUnsupportedTargetSummary(targetName),
					})
					return newExitCodeError(2, fmt.Errorf("doctor: unsupported target %q", targetName))
				}
			}
			activatedCfg, secretSnapshot, secretActivationErr := activateGatewaySecretRuntime(cmd.Context(), cfg, nil)
			cfg = activatedCfg
			if target != "" {
				plan := buildFirstRunPlanFromConfig(cfg, target, false)
				reporter.target = doctorTargetReadinessFromPlan(plan)
				reporter.Add(doctorTargetReadinessStatus(plan))
			}
			secretRuntimeResult := doctorSecretRuntimeStatus(secretSnapshot, secretActivationErr)
			reporter.Add(secretRuntimeResult)
			if secretRuntimeResult.Status == doctor.StatusFail {
				markFailure(2)
			}
			reporter.Add(doctorAuthProvidersStatus(cmd.Context(), cfg))

			if !offline {
				providerName := cfg.Hermes.Provider
				if doctorProviderHealthUsesAuthReadiness(cfg) {
					if !configuredProviderAuthPresent(cfg) {
						reporter.Add(doctor.CheckResult{Name: "provider health", Status: doctor.StatusFail, Summary: fmt.Sprintf("auth missing (%s)", doctorProviderHealthTarget(cfg))})
						markFailure(1)
					} else {
						reporter.Add(doctor.CheckResult{Name: "provider health", Status: doctor.StatusPass, Summary: fmt.Sprintf("auth-ready (%s)", doctorProviderHealthTarget(cfg))})
					}
				} else {
					c, err := newProviderHTTPClient(cfg, providerName)
					if err != nil {
						redactedErr := friendlyProviderSetupDetail(redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey))
						reporter.Add(doctor.CheckResult{Name: "provider setup", Status: doctor.StatusFail, Summary: redactedErr})
						markFailure(1)
					} else {
						ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
						defer cancel()
						if err := c.Health(ctx); err != nil {
							target := doctorProviderHealthTarget(cfg)
							redactedErr := redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)
							reporter.Add(doctor.CheckResult{Name: "provider health", Status: doctor.StatusFail, Summary: fmt.Sprintf("NOT reachable (%s): %s", target, redactedErr)})
							markFailure(1)
						} else {
							reporter.Add(doctor.CheckResult{Name: "provider health", Status: doctor.StatusPass, Summary: fmt.Sprintf("reachable (%s)", doctorProviderHealthTarget(cfg))})
						}
					}
				}
			} else {
				reporter.Add(doctor.CheckResult{Name: "provider health", Status: doctor.StatusSkip, Summary: "skipped (--offline)"})
			}

			reporter.Add(tuilocal.DoctorStatus())
			// Pure local FS inspection of the Gormes-owned home layout -
			// no network, identical under --offline. Auto-groups under the
			// ◆ Directory Structure header via sectionForCheck.
			reporter.Add(doctor.CheckDirectoryStructure(config.GormesHome()))
			if termuxRuntime := doctor.CheckTermuxRuntime(doctor.TermuxRuntimeOptions{}); termuxRuntime.Status != doctor.StatusSkip {
				reporter.Add(termuxRuntime)
			}

			// Toolbox section — inspect the built-in registry. Runs in both modes.
			reg := buildDefaultRegistry(context.Background(), cfg, nil, cfg.Hermes.Model)
			result := doctor.CheckTools(reg)
			reporter.Add(result)
			reporter.Add(doctorWebToolsStatus(cfg))
			reporter.Add(doctorBrowserRuntimeStatusWithDeps(browserRuntimeDoctorDeps{offline: offline}))
			reporter.Add(doctorACPBridgeStatus())
			reporter.Add(doctorGitHubAuthStatus(cmd.Context(), offline))
			reporter.Add(doctor.CheckSkillsHub(cmd.Context(), doctor.SkillsHubOptions{
				Home:            config.GormesHome(),
				Env:             doctorGitHubAuthEnv(),
				RunGHAuthStatus: doctorGitHubAuthRunner,
				Offline:         offline,
			}))
			reporter.Add(doctorGonchoConfig(cfg))
			// Pure local FS over the Gormes profile seam — identical under
			// --offline. Auto-groups under the ◆ Profiles header via
			// sectionForCheck.
			reporter.Add(doctorProfilesStatus())

			runtimeStatus := gateway.RuntimeStatus{}
			if snapshot, err := gateway.NewRuntimeStatusStore(config.GatewayRuntimeStatusPath()).ReadRuntimeStatusSnapshot(context.Background()); err == nil && !snapshot.Missing {
				runtimeStatus = snapshot.Status
			}
			reporter.Add(doctorSlackGatewayConfig(cfg, runtimeStatus))
			reporter.Add(doctorCustomEndpointReadiness(cfg))

			if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() && !cfg.Slack.Enabled {
				reporter.Add(doctor.CheckResult{Name: "gateway", Status: doctor.StatusWarn, Summary: "no channels configured ([telegram], [discord], or [slack])"})
			} else {
				if cfg.Telegram.BotToken != "" {
					if runtimeWarning, ok := doctorTelegramGatewayRuntimeWarning(cfg.Telegram, runtimeStatus); ok {
						reporter.Add(runtimeWarning)
					} else if offline {
						reporter.Add(doctor.CheckResult{Name: "gateway/telegram", Status: doctor.StatusPass, Summary: configuredTelegramGatewayStatusDetail(cfg.Telegram) + " (network validation skipped --offline)"})
					} else if err := doctorNewTelegramClient(cfg.Telegram.BotToken); err != nil {
						redactedErr := redactRuntimeSecretText(err.Error(), cfg.Telegram.BotToken)
						reporter.Add(doctor.CheckResult{Name: "gateway/telegram", Status: doctor.StatusFail, Summary: redactedErr})
						markFailure(2)
					} else {
						reporter.Add(doctor.CheckResult{Name: "gateway/telegram", Status: doctor.StatusPass, Summary: configuredTelegramGatewayStatusDetail(cfg.Telegram)})
					}
				} else {
					reporter.Add(doctor.CheckResult{Name: "gateway/telegram", Status: doctor.StatusSkip, Summary: "disabled"})
				}

				if cfg.Discord.Enabled() {
					if offline {
						reporter.Add(doctor.CheckResult{Name: "gateway/discord", Status: doctor.StatusPass, Summary: "allowed_channel_id=" + cfg.Discord.AllowedChannelID + " (network validation skipped --offline)"})
					} else if err := doctorNewDiscordSession(cfg.Discord.Token); err != nil {
						redactedErr := redactRuntimeSecretText(err.Error(), cfg.Discord.Token)
						reporter.Add(doctor.CheckResult{Name: "gateway/discord", Status: doctor.StatusFail, Summary: redactedErr})
						markFailure(2)
					} else {
						reporter.Add(doctor.CheckResult{Name: "gateway/discord", Status: doctor.StatusPass, Summary: "allowed_channel_id=" + cfg.Discord.AllowedChannelID})
					}
				} else {
					reporter.Add(doctor.CheckResult{Name: "gateway/discord", Status: doctor.StatusSkip, Summary: "disabled"})
				}
			}

			if result.Status == doctor.StatusFail {
				markFailure(2)
			}
			if exitCode != 0 {
				return newExitCodeError(exitCode, fmt.Errorf("doctor: checks failed"))
			}
			if reporter.failed {
				return newExitCodeError(1, fmt.Errorf("doctor: checks failed"))
			}
			return nil
		},
	}
}

func doctorUnsupportedTargetSummary(value string) string {
	return fmt.Sprintf("unsupported target %q (supported: terminal, telegram, whatsapp, discord, slack, navivox)", strings.TrimSpace(value))
}

func doctorSetupTargetFromString(value string) (cli.SetupTargetID, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(cli.SetupTargetTerminal):
		return cli.SetupTargetTerminal, true
	case string(cli.SetupTargetTelegram):
		return cli.SetupTargetTelegram, true
	case string(cli.SetupTargetWhatsApp):
		return cli.SetupTargetWhatsApp, true
	case string(cli.SetupTargetDiscord):
		return cli.SetupTargetDiscord, true
	case string(cli.SetupTargetSlack):
		return cli.SetupTargetSlack, true
	case string(cli.SetupTargetNavivox):
		return cli.SetupTargetNavivox, true
	default:
		return "", false
	}
}

func doctorTargetReadinessStatus(plan cli.FirstRunPlan) doctor.CheckResult {
	status := doctor.StatusPass
	if !plan.Ready {
		status = doctor.StatusWarn
	}
	summary := plan.Summary
	if command := firstRunGuidanceCommand(plan.NextCommand); command != "" {
		if summary == "" {
			summary = "next: " + command
		} else {
			summary += "; next: " + command
		}
	}
	items := make([]doctor.ItemInfo, 0, len(plan.MissingSteps))
	for _, step := range plan.MissingSteps {
		note := step.Detail
		if command := firstRunGuidanceCommand(step.Command); command != "" {
			note += "; run: " + command
		}
		items = append(items, doctor.ItemInfo{
			Name:   step.Label,
			Status: doctor.StatusWarn,
			Note:   note,
		})
	}
	return doctor.CheckResult{
		Name:    "target readiness",
		Status:  status,
		Summary: summary,
		Items:   items,
	}
}

func doctorTargetReadinessFromPlan(plan cli.FirstRunPlan) *doctorTargetReadinessJSON {
	missing := make([]string, 0, len(plan.MissingSteps))
	for _, step := range plan.MissingSteps {
		missing = append(missing, string(step.ID))
	}
	return &doctorTargetReadinessJSON{
		Name:        string(plan.Target),
		Ready:       plan.Ready,
		Summary:     plan.Summary,
		NextCommand: plan.NextCommand,
		Missing:     missing,
	}
}

// doctorSecurityAdvisoriesStatus renders the ◆ Security Advisories section
// from the Gormes-owned advisory catalog + the ~/.gormes ack store. Owned
// divergence: Gormes is a single Go binary with no Python venv, so the
// detector seam (security.NoInstalledPackages) reports nothing installed and
// the section is the faithful clean PASS "No active security advisories"
// (Hermes' "silent otherwise"), never a fabricated importlib/pip scan. Pure
// local FS — identical under --offline. A non-empty ackID is persisted to the
// ~/.gormes ack store before the section renders, and a Gormes-owned
// confirmation item is appended.
func doctorSecurityAdvisoriesStatus(ackID string) doctor.CheckResult {
	return gormescli.DoctorSecurityAdvisoriesStatus(ackID, config.GormesHome())
}

// doctorProfilesStatus renders the ◆ Profiles section content from the real
// Gormes profile seam (defaultProfileCommandSeams) as the source of truth.
// Pure local FS: it reads each profile's own config files and recorded
// gateway_state.json without env/flag overlays or live PID/process probes.
// Owned divergence: no Hermes shell-wrapper alias or orphan-alias checks.
func doctorProfilesStatus() doctor.CheckResult {
	seams := defaultProfileCommandSeams()
	inv := doctor.DoctorProfileInventory{}
	if seams.ListKnownProfiles != nil {
		if known, err := seams.ListKnownProfiles(); err == nil {
			inv.Known = known
		}
	}
	if seams.ReadActiveProfileName != nil {
		if active, err := seams.ReadActiveProfileName(); err == nil {
			inv.Active = active
		}
	}
	if seams.ResolveProfileRoot != nil {
		rootFor := func(name string) (string, bool) {
			root, err := seams.ResolveProfileRoot(name)
			if err != nil || strings.TrimSpace(root) == "" {
				return "", false
			}
			return root, true
		}
		inv.RootExists = func(name string) bool {
			root, ok := rootFor(name)
			if !ok {
				return false
			}
			info, err := os.Stat(root)
			return err == nil && info.IsDir()
		}
		inv.Config = func(name string) doctor.DoctorProfileConfig {
			root, ok := rootFor(name)
			if !ok {
				return doctor.DoctorProfileConfig{Error: "profile_root_unresolved"}
			}
			return doctorProfileConfigSummary(root)
		}
		inv.Gateway = func(name string) doctor.DoctorProfileGateway {
			root, ok := rootFor(name)
			if !ok {
				return doctor.DoctorProfileGateway{Error: "profile_root_unresolved"}
			}
			return doctorProfileGatewaySummary(root)
		}
		if seams.ReadDistributionManifest != nil {
			inv.Distribution = func(name string) doctor.DoctorProfileDistribution {
				root, ok := rootFor(name)
				if !ok {
					return doctor.DoctorProfileDistribution{Error: "profile_root_unresolved"}
				}
				manifest, present, err := seams.ReadDistributionManifest(root)
				if err != nil {
					return doctor.DoctorProfileDistribution{Present: present, Error: "profile_distribution_invalid"}
				}
				if !present {
					return doctor.DoctorProfileDistribution{}
				}
				return doctor.DoctorProfileDistribution{Present: true, Summary: manifest.Summary()}
			}
		}
	}
	return doctor.CheckProfiles(inv)
}

func doctorProfileConfigSummary(root string) doctor.DoctorProfileConfig {
	cfg := config.Config{Hermes: config.HermesCfg{Model: "hermes-agent"}}
	path := filepath.Join(root, "config.toml")
	data, err := os.ReadFile(path)
	if err == nil {
		if err := toml.NewDecoder(bytes.NewReader(data)).EnableUnmarshalerInterface().Decode(&cfg); err != nil {
			return doctor.DoctorProfileConfig{Present: true, Error: "config_toml_decode_failed"}
		}
		provider, model := doctorEffectiveProfileProviderModel(cfg.Hermes)
		return doctor.DoctorProfileConfig{Present: true, Provider: provider, Model: model}
	}
	if !os.IsNotExist(err) {
		return doctor.DoctorProfileConfig{Error: "config_toml_read_failed"}
	}

	path = filepath.Join(root, "config.yaml")
	data, err = os.ReadFile(path)
	if err == nil {
		if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&cfg); err != nil {
			return doctor.DoctorProfileConfig{Present: true, Error: "config_yaml_decode_failed"}
		}
		provider, model := doctorEffectiveProfileProviderModel(cfg.Hermes)
		return doctor.DoctorProfileConfig{Present: true, Provider: provider, Model: model}
	}
	if !os.IsNotExist(err) {
		return doctor.DoctorProfileConfig{Error: "config_yaml_read_failed"}
	}
	return doctor.DoctorProfileConfig{}
}

func doctorEffectiveProfileProviderModel(h config.HermesCfg) (string, string) {
	provider := strings.TrimSpace(h.Provider)
	model := strings.TrimSpace(h.Model)
	if provider == "" && strings.EqualFold(model, "hermes-agent") {
		model = ""
	}
	if provider != "" && (model == "" || strings.EqualFold(model, "hermes-agent")) {
		resolved := llm.ResolveProviderDefaultModel(provider, llm.ProviderDefaultModelOptions{})
		if strings.TrimSpace(resolved.Model) != "" {
			provider = strings.TrimSpace(resolved.Provider)
			model = strings.TrimSpace(resolved.Model)
		}
	}
	return provider, model
}

func doctorProfileGatewaySummary(root string) doctor.DoctorProfileGateway {
	snapshot, err := gateway.NewRuntimeStatusStore(filepath.Join(root, "runtime", "gateway_state.json")).ReadRuntimeStatusSnapshot(context.Background())
	if err != nil {
		return doctor.DoctorProfileGateway{Error: "gateway_state_unreadable"}
	}
	if snapshot.Missing {
		return doctor.DoctorProfileGateway{}
	}
	return doctor.DoctorProfileGateway{Present: true, State: string(snapshot.Status.GatewayState)}
}

// doctorBuildIdentityStatus reports the running binary's identity as
// the first doctor check. PASS when the binary was built from a clean
// source tree (the default release CI invariant); WARN when the
// `-X main.GitDirty=true` ldflag was injected, signalling the binary
// includes uncommitted local changes. Operators reading doctor output
// must know they are NOT running a clean release artifact, otherwise
// stale or unreviewed local work silently rides into production.
//
// When GitCommit is the literal default "unknown" (no ldflags injection
// happened, e.g., `go run` or `go build` without the Makefile/CI flags),
// the summary labels the binary as a "source build" rather than showing
// a bare `commit=unknown` — the sentinel value is accurate but cryptic.
func doctorBuildIdentityStatus() doctor.CheckResult {
	dirty := resolveGitDirty()
	short := resolveGitCommit()
	if len(short) > 12 {
		short = short[:12]
	}
	commitLabel := "commit=" + short
	if short == "unknown" {
		commitLabel = "source build (no commit metadata)"
	}
	if dirty {
		return doctor.CheckResult{
			Name:    "build identity",
			Status:  doctor.StatusWarn,
			Summary: fmt.Sprintf("dirty build: version=%s %s — uncommitted source at build time", Version, commitLabel),
		}
	}
	return doctor.CheckResult{
		Name:    "build identity",
		Status:  doctor.StatusPass,
		Summary: fmt.Sprintf("version=%s %s", Version, commitLabel),
	}
}

func doctorGitHubAuthEnv() map[string]string {
	return map[string]string{
		"GITHUB_TOKEN": os.Getenv("GITHUB_TOKEN"),
		"GH_TOKEN":     os.Getenv("GH_TOKEN"),
	}
}

func doctorGitHubAuthStatus(ctx context.Context, offline bool) doctor.CheckResult {
	env := doctorGitHubAuthEnv()
	if offline && strings.TrimSpace(env["GITHUB_TOKEN"]) == "" && strings.TrimSpace(env["GH_TOKEN"]) == "" {
		return doctor.CheckResult{
			Name:    "GitHub auth",
			Status:  doctor.StatusSkip,
			Summary: "skipped (--offline; set GITHUB_TOKEN/GH_TOKEN for local token readiness)",
		}
	}
	return doctor.CheckGitHubAuth(ctx, doctor.GitHubAuthOptions{
		Env:             env,
		RunGHAuthStatus: doctorGitHubAuthRunner,
	})
}

func doctorAuthProvidersStatus(ctx context.Context, cfg config.Config) doctor.CheckResult {
	codexStatus, codexErr := cli.ResolveAuthStatus(ctx, config.CodexOAuthProvider, cli.AuthStatusOptions{})
	nousStatus, nousErr := cli.ResolveAuthStatus(ctx, config.NousOAuthProvider, cli.AuthStatusOptions{})
	return doctor.CheckAuthProviders([]doctor.AuthProviderStatus{
		doctorAuthProviderFromCLI("OpenAI Codex", config.CodexOAuthProvider, codexStatus, codexErr),
		doctorAuthProviderFromCLI("Nous Portal", config.NousOAuthProvider, nousStatus, nousErr),
		doctorCustomEndpointAuthProviderStatus(cfg),
	})
}

func doctorAuthProviderFromCLI(label, provider string, status cli.ProviderAuthStatus, err error) doctor.AuthProviderStatus {
	out := doctor.AuthProviderStatus{
		Name:            label,
		Provider:        firstNonEmpty(strings.TrimSpace(status.Provider), provider),
		AuthType:        strings.TrimSpace(status.AuthType),
		Status:          strings.TrimSpace(status.Status),
		Reason:          strings.TrimSpace(status.Reason),
		Authenticated:   status.Authenticated,
		CredentialCount: len(status.Credentials),
		Redacted:        true,
	}
	if out.Status == "" {
		out.Status = doctor.AuthProviderLoggedOut
	}
	if err != nil {
		out.Status = doctor.AuthProviderError
		if out.Reason == "" {
			out.Reason = "auth_status_error"
		}
	}
	return out
}

func doctorCustomEndpointAuthProviderStatus(cfg config.Config) doctor.AuthProviderStatus {
	h := cfg.Hermes
	out := doctor.AuthProviderStatus{
		Name:     "Custom endpoint",
		Provider: "custom",
		AuthType: "api_key",
		Status:   doctor.AuthProviderSkipped,
		Reason:   "not configured",
		Redacted: true,
	}
	if strings.EqualFold(strings.TrimSpace(h.Provider), config.CodexOAuthProvider) {
		out.AuthType = "oauth_external"
		out.Reason = "handled by active provider openai-codex"
		return out
	}
	endpointSet := strings.TrimSpace(h.Endpoint) != ""
	authSet := strings.TrimSpace(h.APIKey) != "" || configuredProviderAPIKeyRefPresent(cfg)
	switch {
	case !endpointSet && !authSet:
		return out
	case endpointSet && authSet:
		out.Status = doctor.AuthProviderLoggedIn
		out.Authenticated = true
		out.CredentialCount = 1
		out.Reason = "configured"
	case endpointSet:
		out.Status = doctor.AuthProviderLoggedOut
		out.Reason = "api_key_missing"
	default:
		out.Status = doctor.AuthProviderSkipped
		out.Reason = "endpoint_missing"
	}
	return out
}

func doctorWebToolsStatus(cfg config.Config) doctor.CheckResult {
	status := tools.ResolveWebBackendStatus(nil, tools.WebBackendConfig{
		Backend:             cfg.Web.Backend,
		UseGateway:          cfg.Web.UseGateway,
		ManagedToolsEnabled: true,
		AuthStorePath:       filepath.Join(config.GormesHome(), "auth.json"),
	})
	checkStatus := doctor.StatusWarn
	if status.Available {
		checkStatus = doctor.StatusPass
	}
	summary := fmt.Sprintf("backend=%s route=%s source=%s evidence=%s", status.Backend, status.Route, status.Source, status.Evidence)
	backendNote := fmt.Sprintf("base_url=%s use_gateway=%t managed=%t", status.BaseURL, status.UseGateway, status.Managed)
	if status.Backend == tools.WebBackendGoscraplingBrowser {
		backendNote = "local browser extraction; extract only; " + backendNote
	}
	requiresEnvNote := strings.Join(status.RequiresEnv, ",")
	if status.Backend == tools.WebBackendGoscraplingBrowser {
		requiresEnvNote = "none (browser runtime checked separately)"
	}
	items := []doctor.ItemInfo{
		{Name: "backend", Status: checkStatus, Note: backendNote},
		{Name: "toolset", Status: doctor.StatusPass, Note: strings.Join(status.ToolNames, ",")},
		{Name: "requires_env", Status: doctor.StatusPass, Note: requiresEnvNote},
	}
	if !status.Available {
		items[0].Note = "provider unavailable; " + items[0].Note
	}
	return doctor.CheckResult{
		Name:    "Web tools",
		Status:  checkStatus,
		Summary: summary,
		Items:   items,
	}
}

func doctorProviderHealthTarget(cfg config.Config) string {
	if endpoint := strings.TrimSpace(cfg.Hermes.Endpoint); endpoint != "" {
		return endpoint
	}
	if provider := strings.TrimSpace(cfg.Hermes.Provider); provider != "" {
		return provider
	}
	return "configured provider"
}

func doctorProviderHealthUsesAuthReadiness(cfg config.Config) bool {
	return strings.EqualFold(strings.TrimSpace(cfg.Hermes.Provider), config.CodexOAuthProvider)
}

func doctorACPBridgeStatus() doctor.CheckResult {
	status := acp.DefaultBridgeStatus()
	checkStatus := doctor.StatusPass
	if !status.ServerReady || !status.ClientReady || status.RemoteStatus != acp.BridgeEndpointReady {
		checkStatus = doctor.StatusWarn
	}

	serverStatus := doctor.StatusPass
	if !status.ServerReady {
		serverStatus = doctor.StatusWarn
	}
	clientStatus := doctor.StatusPass
	if !status.ClientReady {
		clientStatus = doctor.StatusWarn
	}
	remoteStatus := doctor.StatusPass
	if status.RemoteStatus != acp.BridgeEndpointReady {
		remoteStatus = doctor.StatusWarn
	}

	return doctor.CheckResult{
		Name:   "ACP bridge",
		Status: checkStatus,
		Summary: fmt.Sprintf("server=%s client=%s remote=%s evidence=%s",
			readyWord(status.ServerReady),
			readyWord(status.ClientReady),
			status.RemoteStatus,
			status.RemoteEvidence,
		),
		Items: []doctor.ItemInfo{
			{
				Name:   "server",
				Status: serverStatus,
				Note:   fmt.Sprintf("evidence=%s surfaces=%d row_backed=%d", status.ServerEvidence, status.ServerSurfaces, status.ServerRowBacked),
			},
			{
				Name:   "client",
				Status: clientStatus,
				Note:   "evidence=" + status.ClientEvidence,
			},
			{
				Name:   "remote",
				Status: remoteStatus,
				Note:   fmt.Sprintf("evidence=%s reason=%s", status.RemoteEvidence, status.RemoteReason),
			},
		},
	}
}

func readyWord(ok bool) string {
	if ok {
		return "ready"
	}
	return "unavailable"
}

func doctorSecretRuntimeStatus(snapshot gormesruntime.SecretRuntimeSnapshot, activationErr error) doctor.CheckResult {
	resolved := 0
	inactive := 0
	unavailable := 0
	items := make([]doctor.ItemInfo, 0, len(snapshot.Entries))
	paths := make([]string, 0, len(snapshot.Entries))
	for path := range snapshot.Entries {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		entry := snapshot.Entries[path]
		state := "unavailable"
		status := doctor.StatusFail
		switch {
		case entry.Resolved:
			resolved++
			state = "resolved"
			status = doctor.StatusPass
		case !entry.Active:
			inactive++
			state = "inactive"
			status = doctor.StatusPass
		default:
			unavailable++
		}
		note := fmt.Sprintf("%s: %s source=%s provider=%s id=%s redacted=%t",
			path,
			state,
			entry.Evidence.Source,
			entry.Evidence.Provider,
			entry.Evidence.ID,
			entry.Evidence.Redacted,
		)
		if entry.Reason != "" {
			note += " reason=" + entry.Reason
		}
		items = append(items, doctor.ItemInfo{Name: "secret", Status: status, Note: note})
	}
	status := doctor.StatusPass
	if unavailable > 0 || activationErr != nil {
		status = doctor.StatusFail
	}
	return doctor.CheckResult{
		Name:    "SecretRef runtime",
		Status:  status,
		Summary: fmt.Sprintf("resolved=%d inactive=%d unavailable=%d", resolved, inactive, unavailable),
		Items:   items,
	}
}

func doctorTelegramGatewayRuntimeWarning(cfg config.TelegramCfg, runtime gateway.RuntimeStatus) (doctor.CheckResult, bool) {
	if strings.TrimSpace(cfg.BotToken) == "" {
		return doctor.CheckResult{}, false
	}
	if runtime.Kind == "" && runtime.GatewayState == "" && len(runtime.Platforms) == 0 {
		return doctor.CheckResult{}, false
	}
	detail := configuredTelegramGatewayStatusDetail(cfg)
	if runtime.GatewayState != "" && runtime.GatewayState != gateway.GatewayStateRunning {
		return doctor.CheckResult{
			Name:    "gateway/telegram",
			Status:  doctor.StatusWarn,
			Summary: fmt.Sprintf("%s; gateway runtime=%s; run `gormes gateway` or `gormes gateway restart` to activate Telegram", detail, runtime.GatewayState),
		}, true
	}
	if runtime.GatewayState != gateway.GatewayStateRunning {
		return doctor.CheckResult{}, false
	}
	platform, ok := runtime.Platforms["telegram"]
	if !ok {
		return doctor.CheckResult{
			Name:    "gateway/telegram",
			Status:  doctor.StatusWarn,
			Summary: detail + "; live gateway has not registered telegram; run `gormes gateway restart` to load new channel config",
		}, true
	}
	if platform.State == gateway.PlatformStateRunning {
		return doctor.CheckResult{}, false
	}
	note := "state=" + string(platform.State)
	if strings.TrimSpace(platform.ErrorMessage) != "" {
		note += " error=" + platform.ErrorMessage
	}
	items := []doctor.ItemInfo{{
		Name:   "runtime",
		Status: doctor.StatusWarn,
		Note:   note,
	}}
	return doctor.CheckResult{
		Name:    "gateway/telegram",
		Status:  doctor.StatusWarn,
		Summary: fmt.Sprintf("%s; live gateway telegram lifecycle=%s; run `gormes gateway restart` to activate channel", detail, platform.State),
		Items:   items,
	}, true
}

func doctorSlackGatewayConfig(cfg config.Config, runtime gateway.RuntimeStatus) doctor.CheckResult {
	slackCfg := cfg.Slack
	if !slackCfg.Enabled {
		return doctor.CheckResult{
			Name:    "Gateway Slack",
			Status:  doctor.StatusWarn,
			Summary: "disabled",
		}
	}

	items := []doctor.ItemInfo{{
		Name:   "config",
		Status: doctor.StatusPass,
		Note:   slackGatewayTargetDetail(slackCfg),
	}}
	if missing := missingSlackCredentials(slackCfg); len(missing) > 0 {
		return doctor.CheckResult{
			Name:    "Gateway Slack",
			Status:  doctor.StatusWarn,
			Summary: "missing_tokens=" + strings.Join(missing, ","),
			Items:   items,
		}
	}

	platform, ok := runtime.Platforms["slack"]
	switch {
	case ok && platform.State == gateway.PlatformStateRunning:
		return doctor.CheckResult{
			Name:    "Gateway Slack",
			Status:  doctor.StatusPass,
			Summary: "running",
			Items:   items,
		}
	case ok && platform.State == gateway.PlatformStateFailed:
		items = append(items, doctor.ItemInfo{
			Name:   "runtime",
			Status: doctor.StatusWarn,
			Note:   platform.ErrorMessage,
		})
		return doctor.CheckResult{
			Name:    "Gateway Slack",
			Status:  doctor.StatusWarn,
			Summary: "startup_failed",
			Items:   items,
		}
	default:
		return doctor.CheckResult{
			Name:    "Gateway Slack",
			Status:  doctor.StatusWarn,
			Summary: "configured_not_running",
			Items:   items,
		}
	}
}

func slackGatewayTargetDetail(cfg config.SlackCfg) string {
	detail := "first_run_discovery=" + strconv.FormatBool(cfg.FirstRunDiscovery)
	if cfg.AllowedChannelID != "" {
		detail = "allowed_channel_id=" + cfg.AllowedChannelID
	}
	return detail + " coalesce_ms=" + strconv.Itoa(cfg.CoalesceMs)
}

// doctorCustomEndpointReadiness reports whether the [hermes] custom endpoint
// triple (endpoint + api_key + model) is configured well enough for Gormes to
// route requests. It performs no network probe — `--offline` callers see the
// same verdict as live ones.
func doctorCustomEndpointReadiness(cfg config.Config) doctor.CheckResult {
	h := cfg.Hermes
	if h.Endpoint == "" && h.APIKey == "" && h.Model == "" {
		return doctor.CheckResult{
			Name:    "Custom endpoint",
			Status:  doctor.StatusWarn,
			Summary: "disabled",
		}
	}

	authItem := readinessItem("api_key", h.APIKey, doctor.StatusWarn)
	if strings.EqualFold(strings.TrimSpace(h.Provider), config.CodexOAuthProvider) {
		authItem = readinessBoolItem("auth", configuredProviderAuthPresent(cfg), doctor.StatusWarn)
		if authItem.Status != doctor.StatusPass {
			authItem.Note = "missing; run `gormes auth add openai-codex`"
		}
	}
	items := []doctor.ItemInfo{
		readinessItem("endpoint", h.Endpoint, doctor.StatusWarn),
		authItem,
		doctorModelReadinessItem(h),
	}

	status := doctor.StatusPass
	missing := 0
	for _, it := range items {
		if it.Status == doctor.StatusFail {
			status = doctor.StatusFail
			missing++
			continue
		}
		if it.Status == doctor.StatusWarn {
			if status != doctor.StatusFail {
				status = doctor.StatusWarn
			}
			missing++
		}
	}

	summary := fmt.Sprintf("configured endpoint=%s", h.Endpoint)
	if strings.TrimSpace(h.Provider) != "" {
		summary = fmt.Sprintf("configured provider=%s endpoint=%s", strings.TrimSpace(h.Provider), h.Endpoint)
	}
	if missing > 0 {
		missingNames := missingReadinessItemNames(items)
		if h.Endpoint == "" {
			summary = "setup incomplete: missing " + strings.Join(missingNames, ", ")
		} else {
			if strings.TrimSpace(h.Provider) != "" {
				summary = fmt.Sprintf("configured provider=%s endpoint=%s missing=%s", strings.TrimSpace(h.Provider), h.Endpoint, strings.Join(missingNames, ", "))
			} else {
				summary = fmt.Sprintf("configured endpoint=%s missing=%s", h.Endpoint, strings.Join(missingNames, ", "))
			}
		}
	}
	return doctor.CheckResult{
		Name:    "Custom endpoint",
		Status:  status,
		Summary: summary,
		Items:   items,
	}
}

func doctorModelReadinessItem(h config.HermesCfg) doctor.ItemInfo {
	item := readinessItem("model", h.Model, doctor.StatusFail)
	if item.Status == doctor.StatusPass && strings.TrimSpace(h.ModelResolutionSource) != "" {
		item.Note = fmt.Sprintf("set model=%s source=%s", strings.TrimSpace(h.Model), strings.TrimSpace(h.ModelResolutionSource))
	}
	return item
}

func missingReadinessItemNames(items []doctor.ItemInfo) []string {
	missing := make([]string, 0, len(items))
	for _, item := range items {
		if item.Status != doctor.StatusPass {
			missing = append(missing, item.Name)
		}
	}
	return missing
}

// readinessItem returns a Pass item with note "set" when value is non-empty,
// or an item at missingStatus with note "missing" when value is empty.
func readinessItem(name, value string, missingStatus doctor.Status) doctor.ItemInfo {
	if value == "" {
		return doctor.ItemInfo{Name: name, Status: missingStatus, Note: "missing"}
	}
	return doctor.ItemInfo{Name: name, Status: doctor.StatusPass, Note: "set"}
}

func readinessBoolItem(name string, present bool, missingStatus doctor.Status) doctor.ItemInfo {
	if !present {
		return doctor.ItemInfo{Name: name, Status: missingStatus, Note: "missing"}
	}
	return doctor.ItemInfo{Name: name, Status: doctor.StatusPass, Note: "set"}
}

func doctorGonchoConfig(cfg config.Config) doctor.CheckResult {
	g := cfg.Goncho
	profile := doctorGonchoProfileName(config.GormesHome())
	memoryDB := doctorGormesDisplayPath(config.MemoryDBPath())
	items := []doctor.ItemInfo{
		{
			Name:   "storage",
			Status: doctor.StatusPass,
			Note: fmt.Sprintf("profile=%s scope=profile_root memory_db=%s sessions_db=%s",
				profile, memoryDB, doctorGormesDisplayPath(config.SessionDBPath())),
		},
		{
			Name:   "runtime",
			Status: doctor.StatusPass,
			Note: fmt.Sprintf("recent_messages=%d max_message_size=%d max_file_size=%d get_context_max_tokens=%d",
				g.RecentMessages, g.MaxMessageSize, g.MaxFileSize, g.GetContextMaxTokens),
		},
		{
			Name:   "features",
			Status: doctor.StatusPass,
			Note: fmt.Sprintf("reasoning_enabled=%t peer_card_enabled=%t summary_enabled=%t dream_enabled=%t",
				g.ReasoningEnabled, g.PeerCardEnabled, g.SummaryEnabled, g.DreamEnabled),
		},
		{
			Name:   "deriver",
			Status: doctor.StatusPass,
			Note: fmt.Sprintf("deriver_workers=%d representation_batch_max_tokens=%d",
				g.DeriverWorkers, g.RepresentationBatchMaxTokens),
		},
		{
			Name:   "dialectic",
			Status: doctor.StatusPass,
			Note:   fmt.Sprintf("dialectic_default_level=%s", g.DialecticDefaultLevel),
		},
	}
	if !g.DreamEnabled {
		items = append(items, doctor.ItemInfo{
			Name:   "dream",
			Status: doctor.StatusWarn,
			Note:   "feature_disabled:dream dream_enabled=false reason=dream fixtures are not available yet",
		})
	}
	return doctor.CheckResult{
		Name:    "Goncho config",
		Status:  doctor.StatusPass,
		Summary: fmt.Sprintf("enabled=%t profile=%s memory_db=%s workspace=%s observer_peer=%s", g.Enabled, profile, memoryDB, g.Workspace, g.ObserverPeer),
		Items:   items,
	}
}

func doctorGonchoProfileName(home string) string {
	clean := filepath.Clean(strings.TrimSpace(home))
	if clean != "." && filepath.Base(filepath.Dir(clean)) == "profiles" {
		if name := filepath.Base(clean); name != "." && name != string(filepath.Separator) && strings.TrimSpace(name) != "" {
			return name
		}
	}
	return config.DefaultProfileID
}

func doctorGormesDisplayPath(path string) string {
	clean := filepath.Clean(strings.TrimSpace(path))
	base := filepath.Clean(strings.TrimSpace(config.GormesBaseHome()))
	if clean == "." || base == "." || base == string(filepath.Separator) {
		return filepath.ToSlash(clean)
	}
	if rel, err := filepath.Rel(base, clean); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		if rel == "." {
			return "~/.gormes"
		}
		return filepath.ToSlash(filepath.Join("~/.gormes", rel))
	}
	return filepath.ToSlash(clean)
}
