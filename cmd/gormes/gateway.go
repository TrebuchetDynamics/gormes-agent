package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/discord"
	telegram "github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/slack"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

var gatewayCmd = &cobra.Command{
	Use:          "gateway",
	Short:        "Run Gormes as a multi-channel messaging gateway",
	Long:         "Runs every configured channel through one gateway.Manager that drives the same kernel + tool loop as the TUI.",
	SilenceUsage: true,
	RunE:         runGateway,
}

// gatewayMutatingUnavailableExitCode is the stable non-zero exit code surfaced
// by the mutating gateway subcommands (start/stop/restart/install/uninstall).
// They intentionally never shell out to systemd/launchd from the Go binary;
// operators are pointed at the internal/cli/service_restart.go helper instead.
const gatewayMutatingUnavailableExitCode = 2

var gatewayMutatingUnavailableSubcommands = []string{
	"start",
	"restart",
	"install",
	"uninstall",
}

func init() {
	for _, name := range gatewayMutatingUnavailableSubcommands {
		gatewayCmd.AddCommand(newGatewayMutatingUnavailableCommand(name))
	}
}

func newGatewayMutatingUnavailableCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:          name,
		Short:        fmt.Sprintf("Unavailable: %s the gateway via the service_restart helper", name),
		Long:         fmt.Sprintf("The %s subcommand is intentionally unavailable in gormes; use the systemd/launchd helper exposed by internal/cli/service_restart.go to drive the live service manager.", name),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return newExitCodeError(gatewayMutatingUnavailableExitCode,
				fmt.Errorf("gateway: %s is not available; use the service_restart helper", name))
		},
	}
}

type gracefulShutdownManager interface {
	Shutdown(context.Context) error
}

type gatewayChannelFactory func(config.Config, *slog.Logger) (gateway.Channel, error)

type gatewayChannelFactories struct {
	Telegram gatewayChannelFactory
	Discord  gatewayChannelFactory
	Slack    gatewayChannelFactory
	Yuanbao  gatewayChannelFactory
}

func runGateway(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() && !cfg.Slack.Enabled && !cfg.Yuanbao.Enabled {
		return fmt.Errorf("no channels configured — set at least one of [telegram], [discord], [slack], or [yuanbao] in config.toml")
	}

	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()
	sessionMirror := startSessionIndexMirror(smap, slog.Default())
	defer sessionMirror.Stop()

	mstore, err := memory.OpenSqlite(config.MemoryDBPath(), cfg.Telegram.MemoryQueueCap, slog.Default())
	if err != nil {
		return fmt.Errorf("memory store: %w", err)
	}
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
		defer cancelShutdown()
		if err := mstore.Close(shutdownCtx); err != nil {
			slog.Warn("memory store close", "err", err)
		}
	}()

	hc, err := newGatewayHermesClient(cfg)
	if err != nil {
		return fmt.Errorf("provider setup: %w", err)
	}
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	reg := buildDefaultRegistry(rootCtx, cfg, hc, cfg.Hermes.Model)
	toolAudit := audit.NewJSONLWriter(config.ToolAuditLogPath())

	k := kernel.New(kernel.Config{
		Model:             cfg.Hermes.Model,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             reg,
		MaxToolIterations: kernel.DefaultMaxToolIterations,
		MaxToolDuration:   30 * time.Second,
		ToolAudit:         toolAudit,
	}, hc, mstore, telemetry.New(), slog.Default())

	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}
	runtimeStatusPath := config.GatewayRuntimeStatusPath()
	runtimeStatus := gateway.NewRuntimeStatusStore(runtimeStatusPath)
	restartMarkerStore := gateway.NewRestartTakeoverStore(gateway.DefaultRestartTakeoverMarkerPath(runtimeStatusPath))

	hooksRoot := config.HooksRoot()
	hooks, loadedHooks, err := gateway.LoadHookScripts(hooksRoot, slog.Default())
	if err != nil {
		slog.Warn("gateway hooks unavailable", "root", hooksRoot, "err", err)
		hooks = gateway.NewHooks()
	}

	restartCfg := gateway.RestartConfig{
		MarkerStore:             restartMarkerStore,
		ServiceManagerAvailable: gateway.EnvironmentServiceManagerAvailable,
		DrainTimeout:            kernel.ShutdownBudget,
	}
	mgr := gateway.NewManager(gatewayManagerConfig(cfg, allowedChats, allowDiscovery, smap, hc, hooks, runtimeStatus, restartCfg), k, slog.Default())

	registeredChannels, err := registerConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, defaultGatewayChannelFactories(), runtimeStatus, slog.Default())
	if err != nil {
		return err
	}
	if registeredChannels == 0 {
		return fmt.Errorf("no runnable channels configured — complete at least one of [telegram], [discord], [slack], or [yuanbao] in config.toml")
	}

	go k.Run(rootCtx)
	bootPath := config.BootPath()
	bootQueued := gateway.StartBootHook(rootCtx, gateway.BootHookConfig{
		Path:   bootPath,
		Model:  cfg.Hermes.Model,
		Client: hc,
		Tools:  reg,
		Log:    slog.Default(),
	})
	go runGatewaySignalLoop(signals, kernel.ShutdownBudget, mgr, cancel, slog.Default(), os.Exit)

	slog.Info("gormes gateway starting", "channels", mgr.ChannelCount(), "endpoint", cfg.Hermes.Endpoint, "hooks_root", hooksRoot, "loaded_hooks", len(loadedHooks), "boot_path", bootPath, "boot_queued", bootQueued)
	return mgr.Run(rootCtx)
}

func newGatewayHermesClient(cfg config.Config) (hermes.Client, error) {
	return newProviderHTTPClient(cfg, cfg.Hermes.Provider)
}

func defaultGatewayChannelFactories() gatewayChannelFactories {
	return gatewayChannelFactories{
		Telegram: func(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
			tc, err := telegram.NewRealClient(cfg.Telegram.BotToken)
			if err != nil {
				return nil, err
			}
			return telegram.New(telegram.Config{
				AllowedChatID:     cfg.Telegram.AllowedChatID,
				FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
				RequireMention:    cfg.Telegram.RequireMention,
				BotUsername:       cfg.Telegram.BotUsername,
				AudioTranscriber:  telegram.NewWhisperTranscriberFromEnv(),
				DynamicCommands:   gatewayTelegramDynamicCommands(context.Background(), cfg),
			}, tc, log), nil
		},
		Discord: func(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
			ds, err := discord.NewRealSession(cfg.Discord.Token)
			if err != nil {
				return nil, err
			}
			return discord.New(discord.Config{
				AllowedChannelID:  cfg.Discord.AllowedChannelID,
				FirstRunDiscovery: cfg.Discord.FirstRunDiscovery,
			}, ds, log), nil
		},
		Slack: func(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
			return slack.NewChannel(slack.NewRealClient(cfg.Slack.BotToken, cfg.Slack.AppToken), log), nil
		},
		Yuanbao: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			return nil, errors.New("yuanbao_runtime_unavailable: live Yuanbao transport is not implemented; the runtime slice binds fake clients only")
		},
	}
}

func gatewayTelegramDynamicCommands(ctx context.Context, cfg config.Config) []gateway.PlatformCommand {
	runtime := skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath())
	skillCommands, _, err := runtime.SkillSlashCommands(ctx, skills.RuntimeOptions{})
	if err != nil || len(skillCommands) == 0 {
		return nil
	}
	commands := make([]gateway.PlatformCommand, 0, len(skillCommands))
	for _, cmd := range skillCommands {
		commands = append(commands, gateway.PlatformCommand{
			Name:        strings.TrimPrefix(cmd.Command, "/"),
			Description: cmd.Description,
		})
	}
	return commands
}

func gatewayCoalesceMs(cfg config.Config) int {
	coalesceMs := 1000
	if cfg.Telegram.CoalesceMs > 0 {
		coalesceMs = cfg.Telegram.CoalesceMs
	}
	if cfg.Discord.CoalesceMs > 0 && (cfg.Telegram.BotToken == "" || cfg.Telegram.CoalesceMs <= 0) {
		coalesceMs = cfg.Discord.CoalesceMs
	}
	if cfg.Slack.Enabled && cfg.Slack.CoalesceMs > 0 && cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() {
		coalesceMs = cfg.Slack.CoalesceMs
	}
	return coalesceMs
}

func gatewayFreshFinalAfter(cfg config.Config) time.Duration {
	if cfg.Telegram.BotToken == "" || cfg.Telegram.FreshFinalAfterSeconds <= 0 {
		return 0
	}
	return time.Duration(cfg.Telegram.FreshFinalAfterSeconds * float64(time.Second))
}

func gatewayManagerConfig(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, smap session.Map, hc hermes.Client, hooks *gateway.Hooks, runtimeStatus gateway.RuntimeStatusWriter, restart gateway.RestartConfig) gateway.ManagerConfig {
	titleStore, titleModel := buildGatewayTitleSeam(context.Background(), smap, hc, cfg.Hermes.Model)
	return gateway.ManagerConfig{
		AllowedChats:               allowedChats,
		AllowDiscovery:             allowDiscovery,
		CoalesceMs:                 gatewayCoalesceMs(cfg),
		FreshFinalAfter:            gatewayFreshFinalAfter(cfg),
		ToolProgressMode:           cfg.Display.ToolProgress,
		ToolProgressCommandEnabled: cfg.Display.ToolProgressCommand,
		PersistToolProgressMode:    config.SetHermesDisplayPlatformToolProgress,
		ToolProgressModes:          gatewayToolProgressModes(cfg),
		SessionMap:                 smap,
		TitleStore:                 titleStore,
		TitleModel:                 titleModel,
		Hooks:                      hooks,
		RuntimeStatus:              runtimeStatus,
		Restart:                    restart,
		RememberedSourceStore:      gateway.NewChannelDirectorySourceStore(config.GormesHome()),
		LiveTurnNow:                func() time.Time { return time.Now() },
		LiveTurnActiveModel: func() string {
			resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes gateway live-turn metadata"})
			return firstUsageString(resolution.Model, cfg.Hermes.Model)
		},
		LiveTurnActiveProvider: func() string {
			resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes gateway live-turn metadata"})
			return firstUsageString(resolution.Provider, cfg.Hermes.Provider)
		},
		AccountUsage: func(ctx context.Context, ev gateway.InboundEvent) (hermes.AccountUsageSnapshot, error) {
			resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes gateway /usage"})
			provider := inferUsageProvider(resolution.Provider, firstUsageString(resolution.Model, cfg.Hermes.Model))
			if provider == "" {
				provider = "openai-codex"
			}
			fetcher := hermes.NewAccountUsageFetcher(accountUsageHTTPClient{client: http.DefaultClient}, func() time.Time { return time.Now().UTC() })
			return fetcher.Fetch(ctx, hermes.AccountUsageFetchRequest{Provider: provider, BaseURL: cfg.Hermes.Endpoint, APIKey: cfg.Hermes.APIKey})
		},
	}
}

func gatewayToolProgressModes(cfg config.Config) map[string]string {
	if len(cfg.Display.Platforms) == 0 {
		return nil
	}
	modes := map[string]string{}
	for platform, display := range cfg.Display.Platforms {
		key := strings.ToLower(strings.TrimSpace(platform))
		mode := strings.TrimSpace(display.ToolProgress)
		if key == "" || mode == "" {
			continue
		}
		modes[key] = mode
	}
	if len(modes) == 0 {
		return nil
	}
	return modes
}

func registerConfiguredGatewayChannels(mgr *gateway.Manager, cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, factories gatewayChannelFactories, status gateway.RuntimeStatusWriter, log *slog.Logger) (int, error) {
	if log == nil {
		log = slog.Default()
	}
	registered := 0

	if cfg.Telegram.BotToken != "" {
		if factories.Telegram == nil {
			return registered, fmt.Errorf("register telegram: missing channel factory")
		}
		ch, err := factories.Telegram(cfg, log)
		if err != nil {
			return registered, err
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register telegram: %w", err)
		}
		if cfg.Telegram.AllowedChatID != 0 {
			allowedChats["telegram"] = strconv.FormatInt(cfg.Telegram.AllowedChatID, 10)
		}
		allowDiscovery["telegram"] = cfg.Telegram.FirstRunDiscovery
		registered++
		log.Info("gateway: telegram channel enabled", "allowed_chat_id", cfg.Telegram.AllowedChatID)
	}

	if cfg.Discord.Enabled() {
		if factories.Discord == nil {
			return registered, fmt.Errorf("register discord: missing channel factory")
		}
		ch, err := factories.Discord(cfg, log)
		if err != nil {
			return registered, err
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register discord: %w", err)
		}
		if cfg.Discord.AllowedChannelID != "" {
			allowedChats["discord"] = cfg.Discord.AllowedChannelID
		}
		allowDiscovery["discord"] = cfg.Discord.FirstRunDiscovery
		registered++
		log.Info("gateway: discord channel enabled", "allowed_channel_id", cfg.Discord.AllowedChannelID)
	}

	if cfg.Slack.Enabled {
		if cfg.Slack.AllowedChannelID != "" {
			allowedChats["slack"] = cfg.Slack.AllowedChannelID
		}
		allowDiscovery["slack"] = cfg.Slack.FirstRunDiscovery

		if missing := missingSlackCredentials(cfg.Slack); len(missing) > 0 {
			errText := "slack: missing " + strings.Join(missing, ",")
			writeGatewayChannelDegraded(status, "slack", errText)
			log.Warn("gateway: slack channel disabled by missing credentials", "missing", strings.Join(missing, ","))
		} else {
			if factories.Slack == nil {
				return registered, fmt.Errorf("register slack: missing channel factory")
			}
			ch, err := factories.Slack(cfg, log)
			if err != nil {
				errText := "slack: startup failed: " + err.Error()
				writeGatewayChannelDegraded(status, "slack", errText)
				log.Warn("gateway: slack channel startup failed", "err", err)
			} else {
				if err := mgr.Register(ch); err != nil {
					return registered, fmt.Errorf("register slack: %w", err)
				}
				registered++
				log.Info("gateway: slack channel enabled", "allowed_channel_id", cfg.Slack.AllowedChannelID)
			}
		}
	}

	if cfg.Yuanbao.Enabled {
		if cfg.Yuanbao.AllowedConversationID != "" {
			allowedChats["yuanbao"] = cfg.Yuanbao.AllowedConversationID
		}
		allowDiscovery["yuanbao"] = cfg.Yuanbao.FirstRunDiscovery

		if missing := cfg.Yuanbao.MissingCredentials(); len(missing) > 0 {
			errText := "yuanbao: missing " + strings.Join(missing, ",")
			writeGatewayChannelDegraded(status, "yuanbao", errText)
			log.Warn("gateway: yuanbao channel disabled by missing credentials", "missing", strings.Join(missing, ","))
			return registered, nil
		}
		if factories.Yuanbao == nil {
			return registered, fmt.Errorf("register yuanbao: missing channel factory")
		}
		ch, err := factories.Yuanbao(cfg, log)
		if err != nil {
			errText := "yuanbao: startup failed: " + err.Error()
			writeGatewayChannelDegraded(status, "yuanbao", errText)
			log.Warn("gateway: yuanbao channel startup failed", "err", err)
			return registered, nil
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register yuanbao: %w", err)
		}
		registered++
		log.Info("gateway: yuanbao channel enabled", "allowed_conversation_id", cfg.Yuanbao.AllowedConversationID)
	}

	return registered, nil
}

func writeGatewayChannelDegraded(status gateway.RuntimeStatusWriter, platform, errText string) {
	if status == nil {
		return
	}
	_ = status.UpdateRuntimeStatus(context.Background(), gateway.RuntimeStatusUpdate{
		Platform:      platform,
		PlatformState: gateway.PlatformStateFailed,
		ErrorMessage:  errText,
	})
}

func missingSlackCredentials(cfg config.SlackCfg) []string {
	missing := []string{}
	if strings.TrimSpace(cfg.BotToken) == "" {
		missing = append(missing, "bot_token")
	}
	if strings.TrimSpace(cfg.AppToken) == "" {
		missing = append(missing, "app_token")
	}
	return missing
}

func runGatewaySignalLoop(signals <-chan os.Signal, budget time.Duration, mgr gracefulShutdownManager, cancel context.CancelFunc, log *slog.Logger, forceExit func(int)) {
	if log == nil {
		log = slog.Default()
	}
	if forceExit == nil {
		forceExit = os.Exit
	}

	sig, ok := <-signals
	if !ok {
		return
	}
	log.Info("gateway shutdown requested", "signal", sig.String())

	timer := time.AfterFunc(budget, func() {
		log.Error("shutdown budget exceeded; forcing exit")
		forceExit(3)
	})
	defer timer.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), budget)
	err := mgr.Shutdown(shutdownCtx)
	shutdownCancel()
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		log.Warn("gateway shutdown drain", "err", err)
	} else if err != nil {
		log.Warn("gateway shutdown drain", "err", err)
	}

	cancel()
}
