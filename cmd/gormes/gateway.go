package main

import (
	"context"
	"database/sql"
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
	"github.com/TrebuchetDynamics/gormes-agent/internal/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	gormesruntime "github.com/TrebuchetDynamics/gormes-agent/internal/runtime"
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

type gatewayReloadManager interface {
	Reload(context.Context) error
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
	secretController := gormesruntime.NewGatewaySecretRuntimeController(gormesruntime.GatewaySecretRuntimeOptions{})
	secretActivation, err := secretController.Activate(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("secret runtime activation: %w", err)
	}
	cfg = secretActivation.Config
	secretSnapshot := secretActivation.Snapshot
	securityReport := evaluateGatewayStartupSecurity(cfg, os.Getenv)
	cfg = securityReport.Config
	logGatewayStartupSecurityEvidence(securityReport.Evidence, slog.Default())
	if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() && !cfg.Slack.Enabled && !cfg.Yuanbao.Enabled {
		return fmt.Errorf("no channels configured — set at least one of [telegram], [discord], [slack], or [yuanbao] in config.toml")
	}
	if _, err := ensureGatewayAgentTemplates(cfg, slog.Default()); err != nil {
		return err
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

	baseHC, err := newGatewayHermesClient(cfg)
	if err != nil {
		return fmt.Errorf("provider setup: %w", err)
	}
	hc := newReloadableHermesClient(baseHC)
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	reg := buildDefaultRegistry(rootCtx, cfg, hc, cfg.Hermes.Model)
	toolAudit := audit.NewJSONLWriter(config.ToolAuditLogPath())

	// Initialize Goncho for cross-session memory persistence.
	// When available, every user + assistant turn is persisted and recent
	// context is injected into the system prompt on each turn.
	var gonchoStore kernel.GonchoStore
	if cfg.Goncho.Enabled {
		gonchoDB, err := sqlOpenGoncho(config.MemoryDBPath())
		if err != nil {
			slog.Warn("goncho db open failed; memory disabled", "err", err)
		} else {
			gc := goncho.NewService(gonchoDB, goncho.Config{
				PeerCardEnabled: cfg.Goncho.PeerCardEnabled,
				DreamEnabled:    cfg.Goncho.DreamEnabled,
			}, slog.Default())
			gonchoStore = newGonchoAdapter(gc)
			slog.Info("goncho initialized", "db", config.MemoryDBPath())
		}
	}

	k := kernel.New(kernel.Config{
		Model:             cfg.Hermes.Model,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             reg,
		MaxToolIterations: configuredMaxToolIterations(cfg),
		MaxToolDuration:   30 * time.Second,
		ToolAudit:         toolAudit,
		Goncho:            gonchoStore,
	}, hc, mstore, telemetry.New(), slog.Default())

	allowedChats, allowDiscovery := gatewayPolicyMaps(cfg)
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
		SelfRestart:             gateway.SelfRestartViaExec,
		DrainTimeout:            kernel.ShutdownBudget,
	}
	var reloadManagerConfig func(context.Context) (gateway.ManagerConfig, error)
	reloadManagerConfig = func(ctx context.Context) (gateway.ManagerConfig, error) {
		next, err := config.Load(os.Args[1:])
		if err != nil {
			return gateway.ManagerConfig{}, fmt.Errorf("config: %w", err)
		}
		activation, err := secretController.Reload(ctx, next)
		if err != nil {
			return gateway.ManagerConfig{}, fmt.Errorf("secret runtime activation: %w", err)
		}
		next = activation.Config
		securityReport := evaluateGatewayStartupSecurity(next, os.Getenv)
		next = securityReport.Config
		logGatewayStartupSecurityEvidence(securityReport.Evidence, slog.Default())
		if next.Telegram.BotToken == "" && !next.Discord.Enabled() && !next.Slack.Enabled && !next.Yuanbao.Enabled {
			return gateway.ManagerConfig{}, fmt.Errorf("no channels configured — set at least one of [telegram], [discord], [slack], or [yuanbao] in config.toml")
		}
		if _, err := ensureGatewayAgentTemplates(next, slog.Default()); err != nil {
			return gateway.ManagerConfig{}, err
		}
		nextBaseHC, err := newGatewayHermesClient(next)
		if err != nil {
			return gateway.ManagerConfig{}, fmt.Errorf("provider setup: %w", err)
		}
		nextAllowedChats, nextAllowDiscovery := gatewayPolicyMaps(next)
		nextCfg := gatewayManagerConfig(next, nextAllowedChats, nextAllowDiscovery, smap, hc, hooks, runtimeStatus, restartCfg)
		nextCfg.ToolRegistry = buildDefaultRegistry(rootCtx, next, hc, next.Hermes.Model)
		nextCfg.SkillRuntime = skills.NewRuntime(next.SkillsRoot(), next.Skills.MaxDocumentBytes, next.Skills.SelectionCap, next.SkillsUsageLogPath())
		if nextCfg.AgentRouting.Enabled {
			nextCfg.AgentRuntimeFactory = newGatewayAgentRuntimeFactory(rootCtx, next, mstore, gonchoStore)
		}
		nextCfg.ReloadConfig = reloadManagerConfig
		hc.Set(nextBaseHC)
		return nextCfg, nil
	}
	mgrCfg := gatewayManagerConfig(cfg, allowedChats, allowDiscovery, smap, hc, hooks, runtimeStatus, restartCfg)
	mgrCfg.ToolRegistry = reg
	mgrCfg.SkillRuntime = skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath())
	if mgrCfg.AgentRouting.Enabled {
		mgrCfg.AgentRuntimeFactory = newGatewayAgentRuntimeFactory(rootCtx, cfg, mstore, gonchoStore)
	}
	mgrCfg.ReloadConfig = reloadManagerConfig
	mgr := gateway.NewManager(mgrCfg, k, slog.Default())

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

	slog.Info("gormes gateway starting", "channels", mgr.ChannelCount(), "endpoint", cfg.Hermes.Endpoint, "hooks_root", hooksRoot, "loaded_hooks", len(loadedHooks), "boot_path", bootPath, "boot_queued", bootQueued, "secret_refs", len(secretSnapshot.Entries))
	return mgr.Run(rootCtx)
}

func activateGatewaySecretRuntime(ctx context.Context, cfg config.Config, resolver gormesruntime.SecretStringResolver) (config.Config, gormesruntime.SecretRuntimeSnapshot, error) {
	activation, err := gormesruntime.ActivateGatewaySecretRefs(ctx, cfg, gormesruntime.GatewaySecretRuntimeOptions{Resolver: resolver})
	return activation.Config, activation.Snapshot, err
}

func newGatewayHermesClient(cfg config.Config) (hermes.Client, error) {
	return newProviderHTTPClient(cfg, cfg.Hermes.Provider)
}

func newGatewayAgentRuntimeFactory(rootCtx context.Context, cfg config.Config, mstore *memory.SqliteStore, gonchoStore kernel.GonchoStore) gateway.AgentRuntimeFactory {
	return func(_ context.Context, req gateway.AgentRuntimeRequest) (gateway.KernelSubmitter, error) {
		agentCfg := cfg
		model := firstUsageString(req.Model, cfg.Hermes.Model)
		agentCfg.Hermes.Model = model
		hc, err := newProviderHTTPClientWithCredentialHome(agentCfg, agentCfg.Hermes.Provider, req.AuthHome)
		if err != nil {
			return nil, err
		}
		reg := req.Tools
		if reg == nil {
			reg = buildDefaultRegistry(rootCtx, agentCfg, hc, model).FilterPolicy(req.ToolPolicy.Allow, req.ToolPolicy.Deny)
		}
		k := kernel.New(kernel.Config{
			Model:             model,
			Endpoint:          agentCfg.Hermes.Endpoint,
			Admission:         kernel.Admission{MaxBytes: agentCfg.Input.MaxBytes, MaxLines: agentCfg.Input.MaxLines},
			Tools:             reg,
			Skills:            req.Skills,
			ToolSafety:        req.ToolSafety,
			MaxToolIterations: configuredMaxToolIterations(agentCfg),
			MaxToolDuration:   30 * time.Second,
			ChatKey:           req.SessionKey,
			ToolAudit:         audit.NewJSONLWriter(config.ToolAuditLogPath()),
			Goncho:            gonchoStore,
		}, hc, mstore, telemetry.New(), slog.Default())
		go k.Run(rootCtx)
		return k, nil
	}
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
				AllowedUserIDs:    cfg.Telegram.AllowedUserIDs,
				FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
				RequireMention:    cfg.Telegram.RequireMention,
				BotUsername:       cfg.Telegram.BotUsername,
				AudioTranscriber:  telegram.NewWhisperTranscriberFromEnv(),
				DynamicCommands:   gatewayTelegramDynamicCommands(context.Background(), cfg),
				TokenLockDir:      config.GatewayLockDir(),
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
		AllowedUsers:               gatewayAllowedUsers(cfg),
		AllowDiscovery:             allowDiscovery,
		CoalesceMs:                 gatewayCoalesceMs(cfg),
		FreshFinalAfter:            gatewayFreshFinalAfter(cfg),
		ToolProgressMode:           cfg.Display.ToolProgress,
		ToolProgressCommandEnabled: cfg.Display.ToolProgressCommand,
		PersistToolProgressMode:    config.SetGormesDisplayPlatformToolProgress,
		ToolProgressModes:          gatewayToolProgressModes(cfg),
		SessionMap:                 smap,
		AgentRouting:               gatewayAgentRoutingConfig(cfg),
		TitleStore:                 titleStore,
		TitleModel:                 titleModel,
		Hooks:                      hooks,
		RuntimeStatus:              runtimeStatus,
		Restart:                    restart,
		RememberedSourceStore:      gateway.NewChannelDirectorySourceStore(config.GormesHome()),
		ContextFilesCWD:            gatewayContextFilesCWD(cfg),
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

func gatewayContextFilesCWD(cfg config.Config) string {
	if cwd := strings.TrimSpace(cfg.Terminal.CWD); cwd != "" && cwd != "." {
		return cwd
	}
	if agent, ok := cfg.Agents.AgentByID(cfg.Agents.DefaultAgentID()); ok {
		return strings.TrimSpace(agent.Workspace)
	}
	return ""
}

func gatewayAllowedUsers(cfg config.Config) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	if len(cfg.Telegram.AllowedUserIDs) > 0 {
		users := make(map[string]bool, len(cfg.Telegram.AllowedUserIDs))
		for _, id := range cfg.Telegram.AllowedUserIDs {
			users[strconv.FormatInt(id, 10)] = true
		}
		out["telegram"] = users
	}
	return out
}

func gatewayPolicyMaps(cfg config.Config) (map[string]string, map[string]bool) {
	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}
	if cfg.Telegram.BotToken != "" {
		if cfg.Telegram.AllowedChatID != 0 {
			allowedChats["telegram"] = strconv.FormatInt(cfg.Telegram.AllowedChatID, 10)
		}
		allowDiscovery["telegram"] = cfg.Telegram.FirstRunDiscovery
	}
	if cfg.Discord.Enabled() {
		if cfg.Discord.AllowedChannelID != "" {
			allowedChats["discord"] = cfg.Discord.AllowedChannelID
		}
		allowDiscovery["discord"] = cfg.Discord.FirstRunDiscovery
	}
	if cfg.Slack.Enabled {
		if cfg.Slack.AllowedChannelID != "" {
			allowedChats["slack"] = cfg.Slack.AllowedChannelID
		}
		allowDiscovery["slack"] = cfg.Slack.FirstRunDiscovery
	}
	if cfg.Yuanbao.Enabled {
		if cfg.Yuanbao.AllowedConversationID != "" {
			allowedChats["yuanbao"] = cfg.Yuanbao.AllowedConversationID
		}
		allowDiscovery["yuanbao"] = cfg.Yuanbao.FirstRunDiscovery
	}
	return allowedChats, allowDiscovery
}

func gatewayAgentRoutingConfig(cfg config.Config) gateway.AgentRoutingConfig {
	enabled := len(cfg.Bindings) > 0 || len(cfg.Agents.List) > 1
	if !enabled {
		return gateway.AgentRoutingConfig{}
	}
	return gateway.AgentRoutingConfig{
		Enabled:  true,
		Agents:   cfg.Agents,
		Bindings: cfg.Bindings,
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
		log.Info("gateway: telegram channel enabled", "allowed_chat_id", cfg.Telegram.AllowedChatID, "allowed_user_count", len(cfg.Telegram.AllowedUserIDs))
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

type gatewayStartupSecurityReport struct {
	Config   config.Config
	Evidence []gateway.AdmissionEvidence
}

func evaluateGatewayStartupSecurity(cfg config.Config, lookupEnv func(string) string) gatewayStartupSecurityReport {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}
	report := gatewayStartupSecurityReport{Config: cfg}
	report.Evidence = append(report.Evidence, gateway.CheckStartupAllowlist(gateway.StartupAdmissionInput{
		AllowlistConfigured: gatewayStartupAllowlistConfigured(cfg, lookupEnv),
		AllowAll:            gatewayStartupAllowAllConfigured(lookupEnv),
	})...)
	credentialReport := gateway.CheckWeakCredentialPlatforms([]gateway.CredentialGuardPlatform{
		{
			Name:    "telegram",
			Enabled: strings.TrimSpace(cfg.Telegram.BotToken) != "",
			Credentials: []gateway.CredentialGuardValue{{
				Field: "bot_token",
				Value: cfg.Telegram.BotToken,
			}},
		},
		{
			Name:    "discord",
			Enabled: cfg.Discord.Enabled(),
			Credentials: []gateway.CredentialGuardValue{{
				Field: "token",
				Value: cfg.Discord.Token,
			}},
		},
		{
			Name:    "slack",
			Enabled: cfg.Slack.Enabled,
			Credentials: []gateway.CredentialGuardValue{
				{Field: "bot_token", Value: cfg.Slack.BotToken},
				{Field: "app_token", Value: cfg.Slack.AppToken},
			},
		},
	})
	report.Evidence = append(report.Evidence, credentialReport.Evidence...)
	for _, platform := range credentialReport.DisabledPlatforms {
		switch platform {
		case "telegram":
			report.Config.Telegram.BotToken = ""
		case "discord":
			report.Config.Discord.Token = ""
		case "slack":
			report.Config.Slack.Enabled = false
			report.Config.Slack.BotToken = ""
			report.Config.Slack.AppToken = ""
		}
	}
	return report
}

func gatewayStartupAllowlistConfigured(cfg config.Config, lookupEnv func(string) string) bool {
	if cfg.Telegram.AllowedChatID != 0 || len(cfg.Telegram.AllowedUserIDs) > 0 {
		return true
	}
	if strings.TrimSpace(cfg.Discord.AllowedChannelID) != "" {
		return true
	}
	if strings.TrimSpace(cfg.Slack.AllowedChannelID) != "" {
		return true
	}
	if strings.TrimSpace(cfg.Yuanbao.AllowedConversationID) != "" {
		return true
	}
	for _, key := range []string{
		"SIGNAL_GROUP_ALLOWED_USERS",
		"GORMES_TELEGRAM_ALLOWED_USERS",
		"TELEGRAM_ALLOWED_USERS",
		"GORMES_DISCORD_CHANNEL_ID",
		"GORMES_SLACK_CHANNEL_ID",
	} {
		if strings.TrimSpace(lookupEnv(key)) != "" {
			return true
		}
	}
	return false
}

func gatewayStartupAllowAllConfigured(lookupEnv func(string) string) bool {
	for _, key := range []string{"GATEWAY_ALLOW_ALL_USERS", "TELEGRAM_ALLOW_ALL_USERS"} {
		if parseGatewayStartupBool(lookupEnv(key)) {
			return true
		}
	}
	return false
}

func parseGatewayStartupBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}

func logGatewayStartupSecurityEvidence(evidence []gateway.AdmissionEvidence, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	for _, item := range evidence {
		if item.Code == "" {
			continue
		}
		log.Warn("gateway startup admission", "code", item.Code, "platform", item.Platform, "field", item.Field, "message", item.Message)
	}
}

func runGatewaySignalLoop(signals <-chan os.Signal, budget time.Duration, mgr gracefulShutdownManager, cancel context.CancelFunc, log *slog.Logger, forceExit func(int)) {
	if log == nil {
		log = slog.Default()
	}
	if forceExit == nil {
		forceExit = os.Exit
	}

	for {
		sig, ok := <-signals
		if !ok {
			return
		}
		if sig == syscall.SIGHUP {
			reloader, ok := mgr.(gatewayReloadManager)
			if !ok {
				log.Warn("gateway config reload unavailable", "signal", sig.String())
				continue
			}
			if err := reloader.Reload(context.Background()); err != nil {
				log.Warn("gateway config reload failed; continuing with last good config")
			} else {
				log.Info("gateway config reloaded", "signal", sig.String())
			}
			continue
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
		return
	}
}

func sqlOpenGoncho(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, stmt := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func newGonchoAdapter(svc *goncho.Service) kernel.GonchoStore {
	return &gonchoAdapter{svc: svc}
}

type gonchoAdapter struct{ svc *goncho.Service }

func (a *gonchoAdapter) AppendTurn(ctx context.Context, peer, sessionKey, role, content string) error {
	if a.svc == nil || sessionKey == "" || content == "" {
		return nil
	}
	_, err := a.svc.CreateMessages(ctx, goncho.CreateMessagesParams{
		SessionKey: sessionKey,
		Messages:   []goncho.CreateMessage{{Peer: peer, Role: role, Content: content}},
	})
	return err
}

func (a *gonchoAdapter) GetContext(ctx context.Context, sessionKey string, maxTokens int) (string, error) {
	if a.svc == nil || sessionKey == "" {
		return "", nil
	}
	result, err := a.svc.Context(ctx, goncho.ContextParams{
		Peer:       "gormes",
		SessionKey: sessionKey,
		MaxTokens:  maxTokens,
	})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, m := range result.RecentMessages {
		role := "User"
		if m.Role == "assistant" {
			role = "Gormes"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String(), nil
}
