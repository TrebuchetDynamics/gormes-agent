package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"

	dynamicagents "github.com/TrebuchetDynamics/goncho/dynamicagents"
	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	goncho "github.com/TrebuchetDynamics/goncho/service"
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

func newGatewayCommand() *cobra.Command {
	cmd := gatewaymodule.NewGatewayCommandWithSeams(gatewayCommandSeams(), gatewayCommandOptions())
	cmd.Flags().Bool("no-wakelock", false, "skip automatic termux-wake-lock acquisition on Termux (gateway foreground mode only)")
	return cmd
}

func gatewayCommandSeams() gatewaymodule.GatewayCommandSeams {
	return gatewaymodule.GatewayCommandSeams{
		Run:                        runGateway,
		StopCommand:                newGatewayStopCommand,
		RestartCommand:             newGatewayRestartCommand,
		ReloadCommand:              newGatewayReloadCommand,
		StatusCommand:              newGatewayStatusCommand,
		FleetCommand:               newGatewayFleetCommand,
		DiscoverCommand:            newGatewayDiscoverCommand,
		ProbeCommand:               newGatewayProbeCommand,
		UsageCostCommand:           newGatewayUsageCostCommand,
		MutatingUnavailableCommand: newGatewayMutatingUnavailableCommand,
		BootInstallCommand:         newGatewayBootInstallCommand,
		BootUninstallCommand:       newGatewayBootUninstallCommand,
	}
}

// gatewayMutatingUnavailableExitCode is the stable non-zero exit code surfaced
// by non-Windows lifecycle subcommands that still do not own a native service
// manager path.
const gatewayMutatingUnavailableExitCode = 2

var gatewayRuntimeGOOS = runtime.GOOS

const gatewayDetachedEnvName = "GORMES_GATEWAY_DETACHED"

var gatewayMutatingUnavailableSubcommands = []string{
	"start",
	"install",
	"uninstall",
}

var gatewayRowBackedUnavailableSubcommands = []string{
	"run",
	"setup",
	"migrate-legacy",
	"list",
}

func newGatewayMutatingUnavailableCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:          name,
		Short:        fmt.Sprintf("Manage gateway %s through the platform service helper", name),
		Long:         fmt.Sprintf("On Windows, the %s subcommand uses the native Scheduled Task gateway service. On Termux, run the foreground gateway in tmux with `gormes gateway`. On other platforms it remains unavailable; use the systemd/launchd helper exposed by internal/cli/service_restart.go to drive the live service manager.", name),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if gatewayRuntimeGOOS == "windows" {
				return runGatewayWindowsScheduledTaskCommand(cmd, name)
			}
			if gatewayTermuxDetected() {
				return newExitCodeError(gatewayMutatingUnavailableExitCode, gatewayTermuxLifecycleGuidanceError(name))
			}
			return newExitCodeError(gatewayMutatingUnavailableExitCode,
				fmt.Errorf("gateway: %s is not available; use the service_restart helper", name))
		},
	}
}

type gatewayWindowsScheduledTaskConfig struct {
	TaskName string
	Command  string
	Args     []string
}

type gatewayWindowsScheduledTaskRunner interface {
	Install(context.Context, gatewayWindowsScheduledTaskConfig) error
	Start(context.Context, gatewayWindowsScheduledTaskConfig) error
	Restart(context.Context, gatewayWindowsScheduledTaskConfig) error
	Uninstall(context.Context, gatewayWindowsScheduledTaskConfig) error
}

var gatewayWindowsTaskRunner gatewayWindowsScheduledTaskRunner = realGatewayWindowsScheduledTaskRunner{}

func runGatewayWindowsScheduledTaskCommand(cmd *cobra.Command, action string) error {
	cfg := defaultGatewayWindowsScheduledTaskConfig()
	out := cmd.OutOrStdout()
	ctx := cmd.Context()

	switch action {
	case "install":
		if err := gatewayWindowsTaskRunner.Install(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("install", err)
		}
		fmt.Fprintf(out, "gateway install: Scheduled Task service installed name=%q\n", cfg.TaskName)
		if err := gatewayWindowsTaskRunner.Start(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("install start", err)
		}
		fmt.Fprintf(out, "gateway install: Scheduled Task service started name=%q\n", cfg.TaskName)
	case "start":
		if err := gatewayWindowsTaskRunner.Start(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("start", err)
		}
		fmt.Fprintf(out, "gateway start: Scheduled Task service started name=%q\n", cfg.TaskName)
	case "restart":
		if err := gatewayWindowsTaskRunner.Restart(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("restart", err)
		}
		fmt.Fprintf(out, "gateway restart: Scheduled Task service restarted name=%q\n", cfg.TaskName)
	case "uninstall":
		if err := gatewayWindowsTaskRunner.Uninstall(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("uninstall", err)
		}
		fmt.Fprintf(out, "gateway uninstall: Scheduled Task service removed name=%q\n", cfg.TaskName)
	default:
		return newExitCodeError(gatewayMutatingUnavailableExitCode, fmt.Errorf("gateway: %s is not available; use the service_restart helper", action))
	}
	return nil
}

func gatewayWindowsScheduledTaskError(action string, err error) error {
	return newExitCodeError(gatewayMutatingUnavailableExitCode,
		fmt.Errorf("gateway %s scheduled_task_unavailable: %w", action, err))
}

func defaultGatewayWindowsScheduledTaskConfig() gatewayWindowsScheduledTaskConfig {
	command, err := os.Executable()
	if err != nil || strings.TrimSpace(command) == "" {
		command = "gormes.exe"
	}
	return gatewayWindowsScheduledTaskConfig{
		TaskName: "Gormes Gateway",
		Command:  command,
		Args:     []string{"gateway"},
	}
}

type realGatewayWindowsScheduledTaskRunner struct{}

func (realGatewayWindowsScheduledTaskRunner) Install(ctx context.Context, cfg gatewayWindowsScheduledTaskConfig) error {
	return runGatewayWindowsScheduledTask(ctx, "/Create", "/TN", cfg.TaskName, "/SC", "ONLOGON", "/TR", windowsScheduledTaskCommandLine(cfg), "/F")
}

func (realGatewayWindowsScheduledTaskRunner) Start(ctx context.Context, cfg gatewayWindowsScheduledTaskConfig) error {
	return runGatewayWindowsScheduledTask(ctx, "/Run", "/TN", cfg.TaskName)
}

func (realGatewayWindowsScheduledTaskRunner) Restart(ctx context.Context, cfg gatewayWindowsScheduledTaskConfig) error {
	_ = runGatewayWindowsScheduledTask(ctx, "/End", "/TN", cfg.TaskName)
	return runGatewayWindowsScheduledTask(ctx, "/Run", "/TN", cfg.TaskName)
}

func (realGatewayWindowsScheduledTaskRunner) Uninstall(ctx context.Context, cfg gatewayWindowsScheduledTaskConfig) error {
	return runGatewayWindowsScheduledTask(ctx, "/Delete", "/TN", cfg.TaskName, "/F")
}

func runGatewayWindowsScheduledTask(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "schtasks", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func windowsScheduledTaskCommandLine(cfg gatewayWindowsScheduledTaskConfig) string {
	parts := []string{quoteWindowsScheduledTaskArg(cfg.Command)}
	for _, arg := range cfg.Args {
		parts = append(parts, quoteWindowsScheduledTaskArg(arg))
	}
	return `cmd.exe /d /c set "` + gatewayDetachedEnvName + `=1"&& ` + strings.Join(parts, " ")
}

func quoteWindowsScheduledTaskArg(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

type gracefulShutdownManager interface {
	Shutdown(context.Context) error
}

type gatewayReloadManager interface {
	Reload(context.Context) error
}

var consumeGatewayPlannedStopMarkerForSelf = func(ctx context.Context) (gateway.PlannedStopConsumeResult, error) {
	store := gateway.NewPlannedStopStore(gateway.DefaultPlannedStopMarkerPath(config.GatewayRuntimeStatusPath()))
	return store.ConsumeForSelf(ctx)
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
	if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() && !cfg.Slack.Enabled && !cfg.Teams.Enabled && !cfg.Yuanbao.Enabled && !cfg.Navivox.Enabled && !gormescli.SimpleXEnv(os.LookupEnv).Enabled {
		return fmt.Errorf("no channels configured — set at least one of [telegram], [discord], [slack], [teams], [yuanbao], [navivox], or SIMPLEX_WS_URL")
	}
	if _, err := ensureGatewayAgentTemplates(cfg, slog.Default()); err != nil {
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

	baseHC, err := newGatewayHermesClient(cfg)
	if err != nil {
		return fmt.Errorf("provider setup: %w", err)
	}
	hc := gateway.NewReloadableHermesClient(baseHC)
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 2)
	shutdownSignals, absorbInterrupt := gatewayShutdownSignalPlan()
	if absorbInterrupt {
		signal.Ignore(os.Interrupt)
	}
	signal.Notify(signals, shutdownSignals...)
	defer signal.Stop(signals)
	reg := gormescli.BuildDefaultRegistry(rootCtx, cfg, hc, cfg.Hermes.Model, gormescli.WithSessionSearch(mstore.DB(), smap))
	toolAudit := audit.NewJSONLWriter(config.ToolAuditLogPath())

	// Initialize Goncho for cross-session memory persistence through the public
	// github.com/TrebuchetDynamics/goncho/integration/gormes adapter. When
	// available, every user + assistant turn is persisted, recent context is
	// injected into the system prompt, and public goncho_* tools are registered.
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

	// Phase 2.D — cron scheduler is initialized after channel registration below.

	allowedChats, allowDiscovery, allowedWhitelists := gatewayPolicyMaps(cfg)
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
		if next.Telegram.BotToken == "" && !next.Discord.Enabled() && !next.Slack.Enabled && !next.Teams.Enabled && !next.Yuanbao.Enabled && !next.Navivox.Enabled && !gormescli.SimpleXEnv(os.LookupEnv).Enabled {
			return gateway.ManagerConfig{}, fmt.Errorf("no channels configured — set at least one of [telegram], [discord], [slack], [teams], [yuanbao], [navivox], or SIMPLEX_WS_URL")
		}
		if _, err := ensureGatewayAgentTemplates(next, slog.Default()); err != nil {
			return gateway.ManagerConfig{}, err
		}
		nextBaseHC, err := newGatewayHermesClient(next)
		if err != nil {
			return gateway.ManagerConfig{}, fmt.Errorf("provider setup: %w", err)
		}
		nextAllowedChats, nextAllowDiscovery, nextAllowedWhitelists := gatewayPolicyMaps(next)
		nextCfg := gatewayManagerConfig(next, nextAllowedChats, nextAllowDiscovery, nextAllowedWhitelists, smap, hc, hooks, runtimeStatus, restartCfg)
		nextCfg.DynamicAgentRegistry = dynamicAgentRegistry
		nextReg := gormescli.BuildDefaultRegistry(rootCtx, next, hc, next.Hermes.Model, gormescli.WithSessionSearch(mstore.DB(), smap))
		if gonchoRuntime != nil {
			gormescli.RegisterGormesGonchoTools(nextReg, gonchoRuntime)
		}
		nextCfg.ToolRegistry = nextReg
		nextCfg.SkillRuntime = skills.NewRuntime(next.SkillsRoot(), next.Skills.MaxDocumentBytes, next.Skills.SelectionCap, next.SkillsUsageLogPath())
		if nextCfg.AgentRouting.Enabled {
			nextCfg.AgentRuntimeFactory = newGatewayAgentRuntimeFactory(rootCtx, next, mstore, gonchoStore)
		}
		nextCfg.ReloadConfig = reloadManagerConfig
		hc.Set(nextBaseHC)
		return nextCfg, nil
	}
	mgrCfg := gatewayManagerConfig(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap, hc, hooks, runtimeStatus, restartCfg)
	mgrCfg.DynamicAgentRegistry = dynamicAgentRegistry
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

	noWakeLock, _ := cmd.Flags().GetBool("no-wakelock")
	wakeLockMgr := tools.TermuxWakeLockManager{}
	if !noWakeLock {
		if err := wakeLockMgr.Acquire(cmd.Context()); err != nil {
			slog.Warn("termux-wake-lock acquire failed; continuing without wake lock", "err", err)
		} else {
			slog.Info("termux-wake-lock acquired")
		}
	}

	go runGatewaySignalLoop(signals, kernel.ShutdownBudget, mgr, cancel, slog.Default(), os.Exit, wakeLockMgr)

	// Phase 2.D — cron scheduler + executor + mirror (opt-in via cfg.Cron.Enabled).
	// Initialized after channel registration so delivery adapters are available.
	if cfg.Cron.Enabled {
		initGatewayCron(cfg, smap.DB(), mstore.DB(), k, rootCtx)
	}

	slog.Info("gormes gateway starting", "channels", mgr.ChannelCount(), "endpoint", cfg.Hermes.Endpoint, "hooks_root", hooksRoot, "loaded_hooks", len(loadedHooks), "boot_path", bootPath, "boot_queued", bootQueued, "secret_refs", len(secretSnapshot.Entries), "wakelock", !noWakeLock)
	return mgr.Run(rootCtx)
}

// initGatewayCron starts the cron scheduler with multi-channel delivery.
// Must be called after channels are registered so Telegram credentials are
// available in the config.
func initGatewayCron(cfg config.Config, smapDB *bbolt.DB, mstoreDB *sql.DB, k *kernel.Kernel, rootCtx context.Context) {
	// Build a multi-protocol delivery sink.
	var sink cron.DeliverySink

	// Telegram delivery (most common).
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

	// Fallback: log to slog when no channel sink is configured.
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

func activateGatewaySecretRuntime(ctx context.Context, cfg config.Config, resolver gormesruntime.SecretStringResolver) (config.Config, gormesruntime.SecretRuntimeSnapshot, error) {
	activation, err := gormesruntime.ActivateGatewaySecretRefs(ctx, cfg, gormesruntime.GatewaySecretRuntimeOptions{Resolver: resolver})
	return activation.Config, activation.Snapshot, err
}

func newGatewayHermesClient(cfg config.Config) (llm.Client, error) {
	return gormescli.NewProviderHTTPClient(cfg, cfg.Hermes.Provider)
}

func newGatewayAgentRuntimeFactory(rootCtx context.Context, cfg config.Config, mstore *memory.SqliteStore, gonchoStore kernel.GonchoStore) gateway.AgentRuntimeFactory {
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

func gatewayManagerConfig(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, allowedWhitelists map[string]gateway.WhitelistConfig, smap session.Map, hc llm.Client, hooks *gateway.Hooks, runtimeStatus gateway.RuntimeStatusWriter, restart gateway.RestartConfig) gateway.ManagerConfig {
	titleStore, titleModel := gormescli.BuildGatewayTitleSeam(context.Background(), smap, hc, cfg.Hermes.Model)
	return gateway.ManagerConfig{
		AllowedChats:               allowedChats,
		AllowedUsers:               gatewayAllowedUsers(cfg),
		AllowedChatWhitelists:      allowedWhitelists,
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
		RestartNotifications:       cfg.GatewayRestartNotifications(),
		KanbanSlashRunner: func(ctx context.Context, input string) (string, error) {
			return gormescli.RunTUIKanbanSlashCommand(ctx, input, kanbanCommandOptions())
		},
		SkillsCommandOptions:  skillsCommandOptionsForConfig(cfg),
		RememberedSourceStore: gateway.NewChannelDirectorySourceStore(config.GormesHome()),
		ContextFilesCWD:       gatewayContextFilesCWD(cfg),
		LiveTurnNow:           func() time.Time { return time.Now() },
		LiveTurnActiveModel: func() string {
			resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes gateway live-turn metadata"})
			return providermodule.FirstUsageString(resolution.Model, cfg.Hermes.Model)
		},
		LiveTurnActiveProvider: func() string {
			resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes gateway live-turn metadata"})
			return providermodule.FirstUsageString(resolution.Provider, cfg.Hermes.Provider)
		},
		ImageInputMode: llm.ImageInputMode(cfg.Agent.ImageInputMode),
		AuxiliaryVision: llm.AuxiliaryVisionConfig{
			Provider: cfg.Auxiliary.Vision.Provider,
			Model:    cfg.Auxiliary.Vision.Model,
			BaseURL:  cfg.Auxiliary.Vision.BaseURL,
		},
		SessionResetPolicy:      cfg.Runtime.SessionResetPolicy,
		SessionResetIdleMinutes: cfg.Runtime.SessionResetAfterMinutes,
		SessionResetDailyHour:   cfg.Runtime.SessionResetDailyHour,
		AccountUsage: func(ctx context.Context, ev gateway.InboundEvent) (llm.AccountUsageSnapshot, error) {
			resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes gateway /usage"})
			provider := providermodule.InferUsageProvider(resolution.Provider, providermodule.FirstUsageString(resolution.Model, cfg.Hermes.Model))
			if provider == "" {
				provider = "openai-codex"
			}
			fetcher := llm.NewAccountUsageFetcher(providermodule.AccountUsageHTTPClient{Client: providermodule.UsageHTTPClient}, func() time.Time { return time.Now().UTC() })
			return fetcher.Fetch(ctx, llm.AccountUsageFetchRequest{Provider: provider, BaseURL: cfg.Hermes.Endpoint, APIKey: cfg.Hermes.APIKey})
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

func gatewayPolicyMaps(cfg config.Config) (map[string]string, map[string]bool, map[string]gateway.WhitelistConfig) {
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
	if len(cfg.Teams.AllowedUserIDs()) > 0 || cfg.Teams.AllowAllUsers {
		return true
	}
	if strings.TrimSpace(cfg.Yuanbao.AllowedConversationID) != "" {
		return true
	}
	if cfg.Navivox.Enabled {
		return true
	}
	if gormescli.SimpleXStartupAllowlistConfigured(lookupEnv) {
		return true
	}
	for _, key := range []string{
		"SIGNAL_GROUP_ALLOWED_USERS",
		"GORMES_TELEGRAM_ALLOWED_USERS",
		"TELEGRAM_ALLOWED_USERS",
		"GORMES_DISCORD_CHANNEL_ID",
		"GORMES_SLACK_CHANNEL_ID",
		"TEAMS_ALLOWED_USERS",
	} {
		if strings.TrimSpace(lookupEnv(key)) != "" {
			return true
		}
	}
	return false
}

func gatewayStartupAllowAllConfigured(lookupEnv func(string) string) bool {
	for _, key := range []string{"GATEWAY_ALLOW_ALL_USERS", "TELEGRAM_ALLOW_ALL_USERS", "TEAMS_ALLOW_ALL_USERS", "SIMPLEX_ALLOW_ALL_USERS"} {
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

func runGatewaySignalLoop(signals <-chan os.Signal, budget time.Duration, mgr gracefulShutdownManager, cancel context.CancelFunc, log *slog.Logger, forceExit func(int), wakeLockMgr tools.TermuxWakeLockManager) {
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
		plannedStop, plannedStopStatus := classifyGatewayShutdownSignal(sig)
		if plannedStop {
			log.Info("gateway shutdown requested", "signal", sig.String(), "planned_stop", true, "planned_stop_status", plannedStopStatus)
		} else {
			log.Warn("gateway shutdown requested", "signal", sig.String(), "planned_stop", false, "exit_class", "unexpected_signal_restartable", "planned_stop_status", plannedStopStatus)
		}

		timer := time.AfterFunc(budget, func() {
			log.Error("shutdown budget exceeded; forcing exit")
			forceExit(3)
		})
		defer timer.Stop()

		if err := wakeLockMgr.Release(context.Background()); err != nil {
			log.Warn("termux-wake-lock release failed", "err", err)
		} else {
			log.Info("termux-wake-lock released")
		}
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

func gatewayShutdownSignalPlan() ([]os.Signal, bool) {
	absorbInterrupt := gatewayRuntimeGOOS == "windows" && gatewayTruthyEnv(os.Getenv(gatewayDetachedEnvName))
	signals := []os.Signal{syscall.SIGTERM, syscall.SIGHUP}
	if !absorbInterrupt {
		signals = append([]os.Signal{os.Interrupt}, signals...)
	}
	return signals, absorbInterrupt
}

func gatewayTruthyEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func classifyGatewayShutdownSignal(sig os.Signal) (bool, gateway.PlannedStopConsumeStatus) {
	if sig == os.Interrupt {
		return true, gateway.PlannedStopConsumeMatched
	}
	if sig != syscall.SIGTERM {
		return false, ""
	}
	result, err := consumeGatewayPlannedStopMarkerForSelf(context.Background())
	if err != nil {
		return false, gateway.PlannedStopConsumeInvalid
	}
	return result.Matched, result.Status
}

func sqlOpenGoncho(path string) (*sql.DB, error) {
	db, err := sqlOpenGonchoRaw(path)
	if err == nil {
		return db, nil
	}
	if !memory.IsSQLiteCorruptionError(err) {
		return nil, err
	}
	if _, healErr := memory.SelfHealCorruptGonchoSQLite(path); healErr != nil {
		return nil, fmt.Errorf("%w; self-heal failed: %v", err, healErr)
	}
	return sqlOpenGonchoRaw(path)
}

func sqlOpenGonchoUnmigrated(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func sqlOpenGonchoRaw(path string) (*sql.DB, error) {
	db, err := sqlOpenGonchoUnmigrated(path)
	if err != nil {
		return nil, err
	}
	if err := memory.EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := goncho.RunMigrations(db); err != nil {
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
