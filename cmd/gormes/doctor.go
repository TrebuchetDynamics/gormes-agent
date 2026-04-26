package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/discord"
	telegram "github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func init() {
	doctorCmd.Flags().Bool("offline", false, "skip the api_server health check and only validate the local tool registry")
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify Gormes runtime: api_server reachability + built-in tools",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load(nil)
		if err != nil {
			return err
		}

		offline, _ := cmd.Flags().GetBool("offline")

		if !offline {
			c := hermes.NewHTTPClient(cfg.Hermes.Endpoint, cfg.Hermes.APIKey)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := c.Health(ctx); err != nil {
				fmt.Fprintf(os.Stderr,
					"[FAIL] api_server: NOT reachable at %s: %v\n\nStart it with:\n  API_SERVER_ENABLED=true hermes gateway start\n\nOr pass --offline to validate only the local tool registry.\n",
					cfg.Hermes.Endpoint, err)
				os.Exit(1)
			}
			fmt.Printf("[PASS] api_server: reachable at %s\n", cfg.Hermes.Endpoint)
		} else {
			fmt.Println("[SKIP] api_server: skipped (--offline)")
		}

		fmt.Print(doctorTUIStatus().Format())

		// Toolbox section — inspect the built-in registry. Runs in both modes.
		reg := buildDefaultRegistry(context.Background(), cfg.Delegation, cfg.SkillsRoot(), nil, cfg.Hermes.Model)
		result := doctor.CheckTools(reg)
		fmt.Print(result.Format())
		fmt.Print(doctorGonchoConfig(cfg).Format())

		runtimeStatus := gateway.RuntimeStatus{}
		if snapshot, err := gateway.NewRuntimeStatusStore(config.GatewayRuntimeStatusPath()).ReadRuntimeStatusSnapshot(context.Background()); err == nil && !snapshot.Missing {
			runtimeStatus = snapshot.Status
		}
		fmt.Print(doctorSlackGatewayConfig(cfg, runtimeStatus).Format())
		fmt.Print(doctorCustomEndpointReadiness(cfg).Format())

		if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() && !cfg.Slack.Enabled {
			fmt.Println("[WARN] gateway: no channels configured ([telegram], [discord], or [slack])")
		} else {
			if cfg.Telegram.BotToken != "" {
				if _, err := telegram.NewRealClient(cfg.Telegram.BotToken); err != nil {
					fmt.Printf("[FAIL] gateway/telegram: %v\n", err)
					os.Exit(2)
				}
				fmt.Printf("[PASS] gateway/telegram: allowed_chat_id=%d\n", cfg.Telegram.AllowedChatID)
			} else {
				fmt.Println("[SKIP] gateway/telegram: disabled")
			}

			if cfg.Discord.Enabled() {
				if _, err := discord.NewRealSession(cfg.Discord.Token); err != nil {
					fmt.Printf("[FAIL] gateway/discord: %v\n", err)
					os.Exit(2)
				}
				fmt.Printf("[PASS] gateway/discord: allowed_channel_id=%s\n", cfg.Discord.AllowedChannelID)
			} else {
				fmt.Println("[SKIP] gateway/discord: disabled")
			}
		}

		if result.Status == doctor.StatusFail {
			os.Exit(2)
		}
		return nil
	},
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

	items := []doctor.ItemInfo{
		readinessItem("endpoint", h.Endpoint, doctor.StatusWarn),
		readinessItem("api_key", h.APIKey, doctor.StatusWarn),
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
	if missing > 0 {
		summary = fmt.Sprintf("configured endpoint=%s missing=%d", h.Endpoint, missing)
	}
	return doctor.CheckResult{
		Name:    "Custom endpoint",
		Status:  status,
		Summary: summary,
		Items:   items,
	}
}

// readinessItem returns a Pass item with note "set" when value is non-empty,
// or an item at missingStatus with note "missing" when value is empty.
func readinessItem(name, value string, missingStatus doctor.Status) doctor.ItemInfo {
	if value == "" {
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
