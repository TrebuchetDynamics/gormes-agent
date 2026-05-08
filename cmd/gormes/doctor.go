package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/acp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/discord"
	telegram "github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	gormesruntime "github.com/TrebuchetDynamics/gormes-agent/internal/runtime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func init() {
	doctorCmd.Flags().Bool("offline", false, "skip the provider health check and validate local runtime checks")
}

var doctorGitHubAuthRunner = doctor.DefaultGitHubAuthStatusRunner

var doctorCmd = &cobra.Command{
	Use:           "doctor",
	Short:         "Verify Gormes runtime: provider readiness + built-in tools",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()
		errOut := cmd.ErrOrStderr()

		cfg, err := config.Load(nil)
		if err != nil {
			return err
		}

		offline, _ := cmd.Flags().GetBool("offline")
		activatedCfg, secretSnapshot, secretActivationErr := activateGatewaySecretRuntime(cmd.Context(), cfg, nil)
		cfg = activatedCfg
		secretRuntimeResult := doctorSecretRuntimeStatus(secretSnapshot, secretActivationErr)
		fmt.Fprint(out, secretRuntimeResult.Format())
		if secretRuntimeResult.Status == doctor.StatusFail {
			return newExitCodeError(2, fmt.Errorf("doctor: secret runtime failed"))
		}

		if !offline {
			providerName := cfg.Hermes.Provider
			if doctorProviderHealthUsesAuthReadiness(cfg) {
				if !configuredProviderAuthPresent(cfg) {
					fmt.Fprintf(errOut,
						"[FAIL] provider health: auth missing (%s)\n\nConfigure Gormes provider credentials/endpoint, or pass --offline to validate local runtime checks only.\n",
						doctorProviderHealthTarget(cfg))
					return newExitCodeError(1, fmt.Errorf("doctor: provider auth missing"))
				}
				fmt.Fprintf(out, "[PASS] provider health: auth-ready (%s)\n", doctorProviderHealthTarget(cfg))
			} else {
				c, err := newProviderHTTPClient(cfg, providerName)
				if err != nil {
					redactedErr := redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)
					fmt.Fprintf(errOut,
						"[FAIL] provider setup: %s\n\nConfigure Gormes provider credentials/endpoint, or pass --offline to validate local runtime checks only.\n",
						redactedErr)
					return newExitCodeError(1, fmt.Errorf("doctor: provider setup failed: %s", redactedErr))
				}
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				if err := c.Health(ctx); err != nil {
					target := doctorProviderHealthTarget(cfg)
					redactedErr := redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)
					fmt.Fprintf(errOut,
						"[FAIL] provider health: NOT reachable (%s): %v\n\nConfigure Gormes provider credentials/endpoint, or pass --offline to validate local runtime checks only.\n",
						target, redactedErr)
					return newExitCodeError(1, fmt.Errorf("doctor: provider unreachable"))
				}
				fmt.Fprintf(out, "[PASS] provider health: reachable (%s)\n", doctorProviderHealthTarget(cfg))
			}
		} else {
			fmt.Fprintln(out, "[SKIP] provider health: skipped (--offline)")
		}

		fmt.Fprint(out, doctorTUIStatus().Format())

		// Toolbox section — inspect the built-in registry. Runs in both modes.
		reg := buildDefaultRegistry(context.Background(), cfg, nil, cfg.Hermes.Model)
		result := doctor.CheckTools(reg)
		fmt.Fprint(out, result.Format())
		fmt.Fprint(out, doctorWebToolsStatus(cfg).Format())
		fmt.Fprint(out, doctorBrowserRuntimeStatus().Format())
		fmt.Fprint(out, doctorACPBridgeStatus().Format())
		fmt.Fprint(out, doctor.CheckGitHubAuth(cmd.Context(), doctor.GitHubAuthOptions{
			Env:             doctorGitHubAuthEnv(),
			RunGHAuthStatus: doctorGitHubAuthRunner,
		}).Format())
		fmt.Fprint(out, doctorGonchoConfig(cfg).Format())

		runtimeStatus := gateway.RuntimeStatus{}
		if snapshot, err := gateway.NewRuntimeStatusStore(config.GatewayRuntimeStatusPath()).ReadRuntimeStatusSnapshot(context.Background()); err == nil && !snapshot.Missing {
			runtimeStatus = snapshot.Status
		}
		fmt.Fprint(out, doctorSlackGatewayConfig(cfg, runtimeStatus).Format())
		fmt.Fprint(out, doctorCustomEndpointReadiness(cfg).Format())

		if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() && !cfg.Slack.Enabled {
			fmt.Fprintln(out, "[WARN] gateway: no channels configured ([telegram], [discord], or [slack])")
		} else {
			if cfg.Telegram.BotToken != "" {
				if _, err := telegram.NewRealClient(cfg.Telegram.BotToken); err != nil {
					fmt.Fprintf(out, "[FAIL] gateway/telegram: %v\n", err)
					return newExitCodeError(2, fmt.Errorf("doctor: telegram client init failed: %w", err))
				}
				fmt.Fprintf(out, "[PASS] gateway/telegram: %s\n", configuredTelegramGatewayStatusDetail(cfg.Telegram))
			} else {
				fmt.Fprintln(out, "[SKIP] gateway/telegram: disabled")
			}

			if cfg.Discord.Enabled() {
				if _, err := discord.NewRealSession(cfg.Discord.Token); err != nil {
					fmt.Fprintf(out, "[FAIL] gateway/discord: %v\n", err)
					return newExitCodeError(2, fmt.Errorf("doctor: discord session init failed: %w", err))
				}
				fmt.Fprintf(out, "[PASS] gateway/discord: allowed_channel_id=%s\n", cfg.Discord.AllowedChannelID)
			} else {
				fmt.Fprintln(out, "[SKIP] gateway/discord: disabled")
			}
		}

		if result.Status == doctor.StatusFail {
			return newExitCodeError(2, fmt.Errorf("doctor: toolbox check failed"))
		}
		return nil
	},
}

func doctorGitHubAuthEnv() map[string]string {
	return map[string]string{
		"GITHUB_TOKEN": os.Getenv("GITHUB_TOKEN"),
		"GH_TOKEN":     os.Getenv("GH_TOKEN"),
	}
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
	items := []doctor.ItemInfo{
		{Name: "backend", Status: checkStatus, Note: fmt.Sprintf("base_url=%s use_gateway=%t managed=%t", status.BaseURL, status.UseGateway, status.Managed)},
		{Name: "toolset", Status: doctor.StatusPass, Note: strings.Join(status.ToolNames, ",")},
		{Name: "requires_env", Status: doctor.StatusPass, Note: strings.Join(status.RequiresEnv, ",")},
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
		readinessItem("model", h.Model, doctor.StatusFail),
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
	items := []doctor.ItemInfo{
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
		Summary: fmt.Sprintf("enabled=%t workspace=%s observer_peer=%s", g.Enabled, g.Workspace, g.ObserverPeer),
		Items:   items,
	}
}
