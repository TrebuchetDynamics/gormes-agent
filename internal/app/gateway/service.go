// Package gateway provides the core gateway runtime behavior extracted from
// cmd/gormes/gateway.go. The RunGateway function orchestrates session, memory,
// kernel, cron, and channel startup — the same contract as the root-level
// runGateway, but without cobra Command or main-package import dependencies.
package gateway

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"

	dynamicagents "github.com/TrebuchetDynamics/goncho/dynamicagents"
	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	gonchoadapter "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/channelmemory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
	gatewaymodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	gormesruntime "github.com/TrebuchetDynamics/gormes-agent/internal/runtime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// RunOptions contains the dependencies injected from the root main package
// that cannot be imported by this internal package.
type RunOptions struct {
	// RegisterChannels is called to register all configured channels
	// against the gateway manager. It is injected from the root package
	// because the channel factory logic (defaultGatewayChannelFactories)
	// lives in the root package alongside gateway_channels.go.
	RegisterChannels func(mgr *gateway.Manager, cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, log *slog.Logger) (int, error)

	// OnNoWakeLock is called when the --no-wakelock flag is set.
	OnNoWakeLock func(cmd *cobra.Command) (func(), error)

	// NoWakeLock returns whether --no-wakelock was set.
	NoWakeLock func(cmd *cobra.Command) bool

	// NewExitCodeError wraps an exit code into an error.
	NewExitCodeError func(code int, err error) error

	// GatewayManagerConfig builds the ManagerConfig from config and runtime state.
	// Injected because it depends on types from the gateway module that would
	// create a cycle if imported here.
	GatewayManagerConfig func(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, allowedWhitelists map[string]gateway.WhitelistConfig, smap session.Map, hc llm.Client, hooks *gateway.Hooks, runtimeStatus gateway.RuntimeStatusWriter, restart gateway.RestartConfig) gateway.ManagerConfig
}

// RunGateway is the core gateway runtime entry point. It was extracted from
// cmd/gormes/gateway.go's runGateway function without behavior changes.
func RunGateway(cmd *cobra.Command, _ []string, opts RunOptions) error {
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
	securityReport := gatewaymodule.EvaluateStartupSecurity(cfg, os.Getenv)
	cfg = securityReport.Config
	gatewaymodule.LogStartupSecurityEvidence(securityReport.Evidence, slog.Default())
	if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() && !cfg.Slack.Enabled && !cfg.Teams.Enabled && !cfg.Yuanbao.Enabled && !cfg.Navivox.Enabled && !gormescli.SimpleXEnv(os.LookupEnv).Enabled {
		return fmt.Errorf("no channels configured — set at least one of [telegram], [discord], [slack], [teams], [yuanbao], [navivox], or SIMPLEX_WS_URL")
	}
	if _, err := gatewaymodule.EnsureAgentTemplates(cfg, slog.Default()); err != nil {
		return err
	}

	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()
	sessionMirror := gormescli.StartSessionIndexMirror(smap, slog.Default())
	defer sessionMirror.Stop()

	memorySettings := channelmemory.SettingsFromConfig(cfg)
	mstore, err := memory.OpenSqlite(config.MemoryDBPath(), memorySettings.QueueCap, slog.Default())
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
	dynamicAgentRegistry, err := dynamicagents.NewDynamicAgentRegistry(mstore.DB())
	if err != nil {
		return fmt.Errorf("dynamic agent registry: %w", err)
	}

	baseHC, err := NewGatewayHermesClient(cfg)
	if err != nil {
		return fmt.Errorf("provider setup: %w", err)
	}
	hc := gateway.NewReloadableHermesClient(baseHC)
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 2)
	shutdownSignals, absorbInterrupt := gateway.ShutdownSignalPlan(runtime.GOOS, os.Getenv)
	if absorbInterrupt {
		signal.Ignore(os.Interrupt)
	}
	signal.Notify(signals, shutdownSignals...)
	defer signal.Stop(signals)
	reg := gormescli.BuildDefaultRegistry(rootCtx, cfg, hc, cfg.Hermes.Model, gormescli.WithSessionSearch(mstore.DB(), smap))
	toolAudit := audit.NewJSONLWriter(config.ToolAuditLogPath())

	// Initialize Goncho for cross-session memory persistence through the public
	// github.com/TrebuchetDynamics/goncho/integration/gormes adapter.
	var gonchoStore kernel.GonchoStore
	var gonchoRuntime *gormesgoncho.Runtime
	if cfg.Goncho.Enabled {
		gonchoRuntime, err = gormesgoncho.Open(rootCtx, gormesgoncho.Config{
			DatabasePath:   config.MemoryDBPath(),
			WorkspaceID:    cfg.Goncho.Workspace,
			ObserverID:     cfg.Goncho.ObserverPeer,
			RecentMessages: cfg.Goncho.RecentMessages,
			Logger:         slog.Default(),
		})
		if err != nil {
			slog.Warn("goncho runtime open failed; memory disabled", "err", err)
		} else {
			gonchoStore = gonchoadapter.NewStore(gonchoRuntime.Service)
			gormescli.RegisterGormesGonchoTools(reg, gonchoRuntime)
			slog.Info(gormescli.FormatGormesGonchoStatus(gonchoRuntime.Status()))
			defer func() {
				shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
				defer cancelShutdown()
				if err := gonchoRuntime.Close(shutdownCtx); err != nil {
					slog.Warn("goncho runtime close", "err", err)
				}
			}()
		}
	}

	k := kernel.New(kernel.Config{
		Model:             cfg.Hermes.Model,
		Provider:          cfg.Hermes.Provider,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             reg,
		MaxToolIterations: gormescli.ConfiguredMaxToolIterations(cfg),
		MaxToolDuration:   30 * time.Second,
		ToolAudit:         toolAudit,
		Goncho:            gonchoStore,
		PrefillMessages:   gormescli.ConfiguredPrefillMessages(cfg),
	}, hc, mstore, telemetry.New(), slog.Default())

	allowedChats, allowDiscovery, allowedWhitelists := GatewayPolicyMaps(cfg)
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
		securityReport := gatewaymodule.EvaluateStartupSecurity(next, os.Getenv)
		next = securityReport.Config
		gatewaymodule.LogStartupSecurityEvidence(securityReport.Evidence, slog.Default())
		if next.Telegram.BotToken == "" && !next.Discord.Enabled() && !next.Slack.Enabled && !next.Teams.Enabled && !next.Yuanbao.Enabled && !next.Navivox.Enabled && !gormescli.SimpleXEnv(os.LookupEnv).Enabled {
			return gateway.ManagerConfig{}, fmt.Errorf("no channels configured — set at least one of [telegram], [discord], [slack], [teams], [yuanbao], [navivox], or SIMPLEX_WS_URL")
		}
		if _, err := gatewaymodule.EnsureAgentTemplates(next, slog.Default()); err != nil {
			return gateway.ManagerConfig{}, err
		}
		nextBaseHC, err := NewGatewayHermesClient(next)
		if err != nil {
			return gateway.ManagerConfig{}, fmt.Errorf("provider setup: %w", err)
		}
		nextAllowedChats, nextAllowDiscovery, nextAllowedWhitelists := GatewayPolicyMaps(next)
		nextCfg := opts.GatewayManagerConfig(next, nextAllowedChats, nextAllowDiscovery, nextAllowedWhitelists, smap, hc, hooks, runtimeStatus, restartCfg)
		nextCfg.DynamicAgentRegistry = dynamicAgentRegistry
		nextReg := gormescli.BuildDefaultRegistry(rootCtx, next, hc, next.Hermes.Model, gormescli.WithSessionSearch(mstore.DB(), smap))
		if gonchoRuntime != nil {
			gormescli.RegisterGormesGonchoTools(nextReg, gonchoRuntime)
		}
		nextCfg.ToolRegistry = nextReg
		nextCfg.SkillRuntime = skills.NewRuntime(next.SkillsRoot(), next.Skills.MaxDocumentBytes, next.Skills.SelectionCap, next.SkillsUsageLogPath())
		if nextCfg.AgentRouting.Enabled {
			nextCfg.AgentRuntimeFactory = NewGatewayAgentRuntimeFactory(rootCtx, next, mstore, gonchoStore)
		}
		nextCfg.ReloadConfig = reloadManagerConfig
		hc.Set(nextBaseHC)
		return nextCfg, nil
	}
	mgrCfg := opts.GatewayManagerConfig(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap, hc, hooks, runtimeStatus, restartCfg)
	mgrCfg.DynamicAgentRegistry = dynamicAgentRegistry
	mgrCfg.ToolRegistry = reg
	mgrCfg.SkillRuntime = skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath())
	if mgrCfg.AgentRouting.Enabled {
		mgrCfg.AgentRuntimeFactory = NewGatewayAgentRuntimeFactory(rootCtx, cfg, mstore, gonchoStore)
	}
	ext := startGatewayExtractor(rootCtx, mstore, hc, cfg, memorySettings, slog.Default())
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
		defer cancelShutdown()
		if err := ext.Close(shutdownCtx); err != nil {
			slog.Warn("extractor close", "err", err)
		}
	}()
	mgrCfg.ReloadConfig = reloadManagerConfig
	mgr := gateway.NewManager(mgrCfg, k, slog.Default())

	registeredChannels, err := opts.RegisterChannels(mgr, cfg, allowedChats, allowDiscovery, slog.Default())
	if err != nil {
		return err
	}
	if registeredChannels == 0 {
		return fmt.Errorf("no runnable channels configured — complete at least one of [telegram], [discord], [slack], [yuanbao], [navivox], or SIMPLEX_WS_URL")
	}

	memoryMonitor := gateway.NewMemoryMonitor(gateway.MemoryMonitorConfig{
		Status: runtimeStatus,
	})
	if memoryMonitor.Start(rootCtx) {
		defer func() {
			stopCtx, cancelStop := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancelStop()
			if err := memoryMonitor.Stop(stopCtx); err != nil {
				slog.Debug("gateway memory monitor stop", "err", err)
			}
		}()
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

	noWakeLock := opts.NoWakeLock(cmd)
	wakeLockMgr := tools.TermuxWakeLockManager{}
	if !noWakeLock {
		if err := wakeLockMgr.Acquire(cmd.Context()); err != nil {
			slog.Warn("termux-wake-lock acquire failed; continuing without wake lock", "err", err)
		} else {
			slog.Info("termux-wake-lock acquired")
		}
	}

	go gateway.RunSignalLoop(gateway.SignalLoopOptions{
		Signals:                  signals,
		Budget:                   kernel.ShutdownBudget,
		Manager:                  mgr,
		Cancel:                   cancel,
		Log:                      slog.Default(),
		ForceExit:                os.Exit,
		WakeLockManager:          wakeLockMgr,
		ConsumePlannedStopMarker: ConsumeGatewayPlannedStopMarker,
	})

	// Phase 2.D — cron scheduler + executor + mirror (opt-in via cfg.Cron.Enabled).
	if cfg.Cron.Enabled {
		InitGatewayCron(cfg, smap.DB(), mstore.DB(), k, rootCtx)
	}

	slog.Info("gormes gateway starting", "channels", mgr.ChannelCount(), "endpoint", cfg.Hermes.Endpoint, "hooks_root", hooksRoot, "loaded_hooks", len(loadedHooks), "boot_path", bootPath, "boot_queued", bootQueued, "secret_refs", len(secretSnapshot.Entries), "wakelock", !noWakeLock)
	return mgr.Run(rootCtx)
}

func startGatewayExtractor(rootCtx context.Context, mstore *memory.SqliteStore, hc llm.Client, cfg config.Config, settings channelmemory.Settings, log *slog.Logger) *memory.Extractor {
	ext := memory.NewExtractor(mstore, hc, memory.ExtractorConfig{
		Model:        cfg.Hermes.Model,
		BatchSize:    settings.ExtractorBatchSize,
		PollInterval: settings.ExtractorPollInterval,
	}, log)
	go ext.Run(rootCtx)
	return ext
}

// InitGatewayCron starts the cron scheduler with multi-channel delivery.
// Must be called after channels are registered so Telegram credentials are
// available in the config.
func InitGatewayCron(cfg config.Config, smapDB *bbolt.DB, mstoreDB *sql.DB, k *kernel.Kernel, rootCtx context.Context) {
	var sink cron.DeliverySink

	tgToken := strings.TrimSpace(cfg.Telegram.BotToken)
	tgChatID := cfg.Telegram.AllowedChatID
	if tgToken != "" && tgChatID != 0 {
		tgAPI, tgErr := tgbotapi.NewBotAPI(tgToken)
		if tgErr == nil {
			sink = cron.FuncSink(func(ctx context.Context, text string) error {
				msg := tgbotapi.NewMessage(tgChatID, text)
				msg.ParseMode = tgbotapi.ModeHTML
				_, err := tgAPI.Send(msg)
				return err
			})
			slog.Info("cron: telegram delivery ready", "chat_id", tgChatID)
		} else {
			slog.Warn("cron: telegram bot init failed", "err", tgErr)
		}
	}

	if sink == nil {
		sink = cron.FuncSink(func(_ context.Context, text string) error {
			slog.Info("cron delivery", "msg", text)
			return nil
		})
	}

	cronStore, cronErr := cron.NewStore(smapDB)
	if cronErr != nil {
		slog.Error("cron: init store", "err", cronErr)
		return
	}
	cronRunStore := cron.NewRunStore(mstoreDB)

	cronExec := cron.NewExecutor(cron.ExecutorConfig{
		Kernel:           k,
		JobStore:         cronStore,
		RunStore:         cronRunStore,
		Sink:             sink,
		CallTimeout:      cfg.Cron.CallTimeout,
		CronApprovalMode: cfg.Approvals.CronMode,
	}, slog.Default())

	cronSched := cron.NewScheduler(cron.SchedulerConfig{
		Store:    cronStore,
		Executor: cronExec,
	}, slog.Default())

	if err := cronSched.Start(rootCtx); err != nil {
		slog.Error("cron: start scheduler", "err", err)
		return
	}
	slog.Info("cron: scheduler started")

	go func() {
		<-rootCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
		defer cancel()
		cronSched.Stop(shutdownCtx)
	}()

	cronMirror := cron.NewMirror(cron.MirrorConfig{
		JobStore: cronStore,
		RunStore: cronRunStore,
		Path:     cfg.CronMirrorPath(),
		Interval: cfg.Cron.MirrorInterval,
	}, slog.Default())
	go cronMirror.Run(rootCtx)
	slog.Info("cron: mirror started")
}

// NewGatewayHermesClient creates an LLM client from the config.
func NewGatewayHermesClient(cfg config.Config) (llm.Client, error) {
	return gormescli.NewProviderHTTPClient(cfg, cfg.Hermes.Provider)
}

// NewGatewayAgentRuntimeFactory creates a factory for dynamic agent sub-kernels.
func NewGatewayAgentRuntimeFactory(rootCtx context.Context, cfg config.Config, mstore *memory.SqliteStore, gonchoStore kernel.GonchoStore) gateway.AgentRuntimeFactory {
	return func(_ context.Context, req gateway.AgentRuntimeRequest) (gateway.KernelSubmitter, error) {
		agentCfg := cfg
		model := providermodule.FirstUsageString(req.Model, cfg.Hermes.Model)
		agentCfg.Hermes.Model = model
		hc, err := gormescli.NewProviderHTTPClientWithCredentialHome(agentCfg, agentCfg.Hermes.Provider, req.AuthHome)
		if err != nil {
			return nil, err
		}
		reg := req.Tools
		if reg == nil {
			reg = gormescli.BuildDefaultRegistry(rootCtx, agentCfg, hc, model).FilterPolicy(req.ToolPolicy.Allow, req.ToolPolicy.Deny)
		}
		k := kernel.New(kernel.Config{
			Model:             model,
			Endpoint:          agentCfg.Hermes.Endpoint,
			Admission:         kernel.Admission{MaxBytes: agentCfg.Input.MaxBytes, MaxLines: agentCfg.Input.MaxLines},
			Tools:             reg,
			Skills:            req.Skills,
			ToolSafety:        req.ToolSafety,
			MaxToolIterations: gormescli.ConfiguredMaxToolIterations(agentCfg),
			MaxToolDuration:   30 * time.Second,
			ChatKey:           req.SessionKey,
			ToolAudit:         audit.NewJSONLWriter(config.ToolAuditLogPath()),
			Goncho:            gonchoStore,
			PrefillMessages:   gormescli.ConfiguredPrefillMessages(agentCfg),
		}, hc, mstore, telemetry.New(), slog.Default())
		go k.Run(rootCtx)
		return k, nil
	}
}

// GatewayCoalesceMs returns the coalesce interval in milliseconds for the
// first configured channel that has a non-zero value.
func GatewayCoalesceMs(cfg config.Config) int {
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

// GatewayFreshFinalAfter returns the fresh-final duration for Telegram.
func GatewayFreshFinalAfter(cfg config.Config) time.Duration {
	if cfg.Telegram.BotToken == "" || cfg.Telegram.FreshFinalAfterSeconds <= 0 {
		return 0
	}
	return time.Duration(cfg.Telegram.FreshFinalAfterSeconds * float64(time.Second))
}

// GatewayPolicyMaps extracts allowed-chat, discovery, and whitelist maps from
// the canonical config. Each map is keyed by channel platform name.
func GatewayPolicyMaps(cfg config.Config) (map[string]string, map[string]bool, map[string]gateway.WhitelistConfig) {
	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}
	whitelists := map[string]gateway.WhitelistConfig{}
	if cfg.Telegram.BotToken != "" {
		if cfg.Telegram.AllowedChatID != 0 {
			allowedChats["telegram"] = strconv.FormatInt(cfg.Telegram.AllowedChatID, 10)
		}
		if wl := gateway.ParseWhitelistConfig(cfg.Telegram.AllowedChatIDs()); wl.Enabled {
			whitelists["telegram"] = wl
		}
		allowDiscovery["telegram"] = cfg.Telegram.FirstRunDiscovery
	}
	if cfg.Discord.Enabled() {
		if cfg.Discord.AllowedChannelID != "" {
			allowedChats["discord"] = cfg.Discord.AllowedChannelID
		}
		if wl := gateway.ParseWhitelistConfig(cfg.Discord.AllowedChannelIDs()); wl.Enabled {
			whitelists["discord"] = wl
		}
		allowDiscovery["discord"] = cfg.Discord.FirstRunDiscovery
	}
	if cfg.Slack.Enabled {
		if cfg.Slack.AllowedChannelID != "" {
			allowedChats["slack"] = cfg.Slack.AllowedChannelID
		}
		if wl := gateway.ParseWhitelistConfig(cfg.Slack.AllowedChannelIDs()); wl.Enabled {
			whitelists["slack"] = wl
		}
		allowDiscovery["slack"] = cfg.Slack.FirstRunDiscovery
	}
	if cfg.Teams.Enabled {
		allowDiscovery["teams"] = false
	}
	if cfg.Yuanbao.Enabled {
		if cfg.Yuanbao.AllowedConversationID != "" {
			allowedChats["yuanbao"] = cfg.Yuanbao.AllowedConversationID
		}
		allowDiscovery["yuanbao"] = cfg.Yuanbao.FirstRunDiscovery
	}
	if cfg.Navivox.Enabled {
		allowDiscovery[channelsmodule.NavivoxPlatformName] = false
	}
	if simplexInfo := gormescli.SimpleXEnv(os.LookupEnv); simplexInfo.Enabled {
		if simplexInfo.HomeChannel != "" {
			allowedChats[simplexInfo.Platform] = simplexInfo.HomeChannel
		}
		allowDiscovery[simplexInfo.Platform] = false
	}
	return allowedChats, allowDiscovery, whitelists
}

// GatewayAllowedUsers extracts the allowed-users map per platform from config.
func GatewayAllowedUsers(cfg config.Config) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	if len(cfg.Telegram.AllowedUserIDs) > 0 {
		users := make(map[string]bool, len(cfg.Telegram.AllowedUserIDs))
		for _, id := range cfg.Telegram.AllowedUserIDs {
			users[strconv.FormatInt(id, 10)] = true
		}
		out["telegram"] = users
	}
	if teamsUsers := cfg.Teams.AllowedUserIDs(); len(teamsUsers) > 0 {
		users := make(map[string]bool, len(teamsUsers))
		for _, id := range teamsUsers {
			users[id] = true
		}
		out["teams"] = users
	}
	if cfg.Navivox.Enabled {
		out[channelsmodule.NavivoxPlatformName] = map[string]bool{"navivox": true}
	}
	if simplexInfo := gormescli.SimpleXEnv(os.LookupEnv); simplexInfo.Enabled {
		if len(simplexInfo.AllowedUsers) > 0 {
			out[simplexInfo.Platform] = simplexInfo.AllowedUsers
		} else if simplexInfo.AllowAllUsers {
			out[simplexInfo.Platform] = map[string]bool{"*": true}
		}
	}
	return out
}

// GatewayAgentRoutingConfig builds the agent routing config from bindings/agents.
func GatewayAgentRoutingConfig(cfg config.Config) gateway.AgentRoutingConfig {
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

// GatewayToolProgressModes extracts tool progress display modes per platform.
func GatewayToolProgressModes(cfg config.Config) map[string]string {
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

// ConsumeGatewayPlannedStopMarker is the default planned-stop consumer for
// the gateway runtime.
var ConsumeGatewayPlannedStopMarker gateway.PlannedStopConsumer = func(ctx context.Context) (gateway.PlannedStopConsumeResult, error) {
	store := gateway.NewPlannedStopStore(gateway.DefaultPlannedStopMarkerPath(config.GatewayRuntimeStatusPath()))
	return store.ConsumeForSelf(ctx)
}
