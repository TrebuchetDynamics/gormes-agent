// Command gormes is the Go-native Hermes-compatible agent runtime.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			dumpCrash(r)
			os.Exit(2)
		}
	}()

	root := newRootCommand()
	if err := executeRootCommand(root, os.Args[1:]...); err != nil {
		os.Exit(exitCodeFromError(err))
	}
}

func executeRootCommand(root *cobra.Command, args ...string) error {
	if suggestion, ok := cli.TypoSuggestion(args); ok {
		fmt.Fprintf(root.ErrOrStderr(), "unknown command %q for %q\n%s\n", args[0], root.CommandPath(), suggestion)
		return newExitCodeError(1, fmt.Errorf("unknown command %q for %q; %s", args[0], root.CommandPath(), suggestion))
	}
	if len(args) > 0 {
		root.SetArgs(args)
	}
	return root.Execute()
}

func newRootCommand() *cobra.Command {
	return newRootCommandWithRuntime(rootRuntime{})
}

type rootRuntime struct {
	runTUI                 func(*cobra.Command, []string) error
	runResolvedTUI         func(*cobra.Command, tuiInvocation) error
	runOneshot             func(*cobra.Command, oneshotInvocation) error
	newOneshotClient       oneshotClientFactory
	configureOneshotKernel oneshotKernelConfigurer
	tuiProgramFactory      tuiProgramFactory
}

type tuiInvocation struct {
	Inference config.TUIInferenceResolution
	Config    config.Config
	// RemoteURL, when non-empty, switches startup to remote-TUI mode:
	// gormes connects to the gateway's SSE event stream instead of
	// instantiating a local kernel + provider client. Empty leaves local
	// Bubble Tea behavior intact.
	RemoteURL string
}

type oneshotInvocation struct {
	Prompt    string
	Inference config.OneshotInferenceResolution
	Config    config.Config
}

func newRootCommandWithRuntime(runtime rootRuntime) *cobra.Command {
	if runtime.tuiProgramFactory == nil {
		runtime.tuiProgramFactory = defaultTUIProgramFactory
	}
	if runtime.runResolvedTUI == nil {
		if runtime.runTUI != nil {
			runLegacyTUI := runtime.runTUI
			runtime.runResolvedTUI = func(cmd *cobra.Command, _ tuiInvocation) error {
				return runLegacyTUI(cmd, nil)
			}
		} else {
			runtime.runResolvedTUI = func(cmd *cobra.Command, invocation tuiInvocation) error {
				return runResolvedTUIWithRuntime(cmd, invocation, runtime)
			}
		}
	}
	if runtime.newOneshotClient == nil {
		runtime.newOneshotClient = newOneshotHTTPClient
	}
	if runtime.runOneshot == nil {
		newClient := runtime.newOneshotClient
		configureKernel := runtime.configureOneshotKernel
		runtime.runOneshot = func(cmd *cobra.Command, invocation oneshotInvocation) error {
			return runResolvedOneshotWithClient(cmd, invocation, newClient, configureKernel)
		}
	}
	resetGonchoDoctorFlags()
	root := &cobra.Command{
		Use:   "gormes",
		Short: "Go-native Hermes-compatible agent runtime",
		Long: `Gormes runs AI agents as a single static Go binary: no Python, no Docker, no Hermes process.

Getting started:
  gormes onboard                    show configured state and next steps
  gormes setup provider             configure endpoint, model, and API key
  gormes setup model                pick the default provider/model
  gormes --oneshot "hello"          test one provider-backed turn

Daily use:
  gormes                            open the TUI
  gormes --offline                  smoke test without provider calls
  gormes doctor --offline           check local readiness
  gormes dashboard                  start http://127.0.0.1:43827/dashboard

Configuration:
  gormes config edit                open config.toml in your editor
  gormes config set <key> <value>   set a supported config value
  gormes config show                show config with secrets redacted
  gormes secrets audit --plan file  audit SecretRef runtime plans
  gormes security audit --deep      inspect gateway, channel, tool, and state security
  gormes auth add <provider>        add provider credentials
  gormes logout <provider>          clear stored provider auth

Gateway:
  gormes acp client                connect a debug ACP client to Gormes
  gormes system event "note"        enqueue a system event and heartbeat wake
  gormes gateway                    start the configured gateway
  gormes gateway status             check gateway runtime state
  gormes gateway stop               stop a running gateway
  gormes telegram                   start Telegram-only mode
  gormes logs                       show recent gateway logs

Agents and profiles:
  gormes agent reset                seed default agent context templates
  gormes setup agent                print multi-agent setup guidance
  gormes setup workspace            print workspace setup guidance
  gormes setup bindings             print channel-to-agent binding guidance
  gormes profile list               list known profiles
  gormes profile set <name>         switch active profile

Memory and sessions:
  gormes memory status              inspect memory store
  gormes session list               list past sessions
  gormes session export <id>        export a session transcript
  gormes goncho doctor --json       inspect Goncho memory storage

Tools and skills:
  gormes skills list                list installed skills
  gormes skills install <url>       install a direct SKILL.md URL
  gormes mcp login <server>         refresh OAuth for one MCP server

Maintenance:
  gormes status                     show runtime and progress blockers
  gormes usage                      show provider account usage
  gormes migrate hermes             import state from Hermes (dry-run)
  gormes version                    print version
  gormes uninstall                  remove Gormes artifacts

Environment:
  GORMES_HOME                       runtime home (default ~/.gormes)
  GORMES_API_KEY                    provider API key
  GORMES_ENDPOINT                   provider endpoint URL
  GORMES_SKILLS_ROOT                custom skills directory

Docs: https://docs.gormes.ai`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRootCommand(cmd, args, runtime)
		},
	}
	root.Flags().StringP("oneshot", "z", "", "one-shot mode: send a single prompt and resolve model/provider selection without starting the TUI")
	root.Flags().StringP("model", "m", "", "model override for --oneshot or TUI startup; also settable via GORMES_INFERENCE_MODEL")
	root.Flags().String("provider", "", "provider override for --oneshot or TUI startup; also settable via GORMES_INFERENCE_PROVIDER")
	root.Flags().String("endpoint", "", "provider endpoint override for --oneshot or TUI startup; invocation-only and also settable via GORMES_ENDPOINT")
	root.Flags().String("api-key", "", "provider API key override for --oneshot or TUI startup; invocation-only and never persisted")
	root.Flags().Bool("offline", false, "run the TUI as a local smoke test without provider health checks or network submits")
	root.Flags().String("resume", "", "override persisted session_id for the TUI's default key")
	root.Flags().String("remote", "", "connect the TUI to a remote Gormes gateway over SSE (consumes /events; bypasses local kernel and provider setup)")
	root.AddCommand(doctorCmd, versionCmd, telegramCmd, gatewayCmd, sessionCmd, memoryCmd, gonchoCmd, newACPCommand(), newSystemCommand(), newAgentCommand(), newUsageCommand(), newStatusCommand(), newAuthCommand(), newLogoutCommand(), newConfigCommand(), newSecretsCommand(), newSecurityCommand(), newMigrateCommand(), newProfileCommand(), newModelCommand(), newSetupCommand(), newOnboardCommand(), newSkillsCommand(), newMCPCommand(), newDashboardCommand(), newUninstallCommand(), newLogsCommand())
	return root
}

func runRootCommand(cmd *cobra.Command, args []string, runtime rootRuntime) error {
	if cmd.Flags().Changed("oneshot") {
		invocation, err := resolveOneshotInvocation(cmd)
		if err != nil {
			return err
		}
		return runtime.runOneshot(cmd, invocation)
	}
	invocation, err := resolveTUIInvocation(cmd)
	if err != nil {
		return err
	}
	return runtime.runResolvedTUI(cmd, invocation)
}

func resolveOneshotInvocation(cmd *cobra.Command) (oneshotInvocation, error) {
	prompt, _ := cmd.Flags().GetString("oneshot")
	modelFlag, _ := cmd.Flags().GetString("model")
	providerFlag, _ := cmd.Flags().GetString("provider")
	endpointFlag, _ := cmd.Flags().GetString("endpoint")
	apiKeyFlag, _ := cmd.Flags().GetString("api-key")

	cfg, err := config.Load(nil)
	if err != nil {
		return oneshotInvocation{Prompt: prompt}, err
	}
	applyProviderStartupFlags(&cfg, endpointFlag, apiKeyFlag)
	resolution, err := config.ResolveOneshotInference(config.OneshotInferenceRequest{
		Config:       cfg,
		ModelFlag:    modelFlag,
		ProviderFlag: providerFlag,
	})
	resolution = resolveStaticStartupInference(resolution)
	invocation := oneshotInvocation{
		Prompt:    prompt,
		Inference: resolution,
		Config:    cfg,
	}
	if err != nil {
		return invocation, newExitCodeError(2, err)
	}
	return invocation, nil
}

func resolveTUIInvocation(cmd *cobra.Command) (tuiInvocation, error) {
	modelFlag, _ := cmd.Flags().GetString("model")
	providerFlag, _ := cmd.Flags().GetString("provider")
	endpointFlag, _ := cmd.Flags().GetString("endpoint")
	apiKeyFlag, _ := cmd.Flags().GetString("api-key")
	remoteFlag, _ := cmd.Flags().GetString("remote")

	cfg, err := config.Load(nil)
	if err != nil {
		return tuiInvocation{RemoteURL: remoteFlag}, err
	}
	applyProviderStartupFlags(&cfg, endpointFlag, apiKeyFlag)
	resolution, err := config.ResolveTUIInference(config.TUIInferenceRequest{
		Config:       cfg,
		ModelFlag:    modelFlag,
		ProviderFlag: providerFlag,
	})
	resolution = resolveStaticStartupInference(resolution)
	invocation := tuiInvocation{
		Inference: resolution,
		Config:    cfg,
		RemoteURL: remoteFlag,
	}
	if err != nil {
		return invocation, newExitCodeError(2, err)
	}
	return invocation, nil
}

func applyProviderStartupFlags(cfg *config.Config, endpointFlag, apiKeyFlag string) {
	if endpoint := strings.TrimSpace(endpointFlag); endpoint != "" {
		cfg.Hermes.Endpoint = endpoint
	}
	if apiKey := strings.TrimSpace(apiKeyFlag); apiKey != "" {
		cfg.Hermes.APIKey = apiKey
	}
}

func resolveStaticStartupInference(resolution config.InferenceResolution) config.InferenceResolution {
	if resolution.Model == "" {
		return resolution
	}
	metadata := hermes.LookupModelMetadata(hermes.ModelRegistryQuery{
		Provider: resolution.Provider,
		Model:    resolution.Model,
	})
	if !metadata.Found {
		return resolution
	}
	resolution.Model = metadata.Model
	if resolution.Provider == "" {
		resolution.Provider = metadata.Provider
		resolution.ProviderAutoDetectRequired = false
	}
	return resolution
}

type oneshotClientFactory func(context.Context, config.Config, oneshotInvocation) (hermes.Client, error)
type oneshotKernelConfigurer func(*kernel.Config)

func newOneshotHTTPClient(_ context.Context, cfg config.Config, invocation oneshotInvocation) (hermes.Client, error) {
	return newProviderHTTPClient(cfg, invocation.Inference.Provider)
}

func runResolvedOneshot(cmd *cobra.Command, invocation oneshotInvocation) error {
	return runResolvedOneshotWithClient(cmd, invocation, newOneshotHTTPClient)
}

func runResolvedOneshotWithClient(cmd *cobra.Command, invocation oneshotInvocation, newClient oneshotClientFactory, configureKernel ...oneshotKernelConfigurer) error {
	if newClient == nil {
		newClient = newOneshotHTTPClient
	}
	cfg := invocation.Config
	model := invocation.Inference.Model
	if model == "" {
		model = cfg.Hermes.Model
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := newClient(rootCtx, cfg, invocation)
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes -z: provider setup failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)))
	}
	if client == nil {
		return newExitCodeError(1, fmt.Errorf("gormes -z: provider setup failed: %w", errors.New("nil hermes client")))
	}

	toolSafety, err := kernel.NewOneshotToolSafetyPolicy(kernel.OneshotToolSafetyOptions{
		TrustClass: kernel.TrustClassOperator,
	})
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes -z: safety policy setup failed: %w", err))
	}
	kernelCfg := kernel.Config{
		Model:             model,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		MaxToolIterations: configuredMaxToolIterations(cfg),
		ToolAudit:         audit.NewJSONLWriter(config.ToolAuditLogPath()),
		ToolSafety:        toolSafety,
	}
	if len(configureKernel) > 0 && configureKernel[0] != nil {
		configureKernel[0](&kernelCfg)
	}
	k := kernel.New(kernelCfg, client, store.NewNoop(), telemetry.New(), slog.New(slog.NewTextHandler(io.Discard, nil)))

	runDone := make(chan error, 1)
	go func() {
		runDone <- k.Run(rootCtx)
	}()
	defer func() {
		stop()
		select {
		case <-runDone:
		case <-time.After(kernel.ShutdownBudget):
		}
	}()

	initial, err := readOneshotFrame(rootCtx, k.Render())
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes -z: kernel startup failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)))
	}
	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: invocation.Prompt}); err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes -z: submit failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)))
	}
	final, err := waitForOneshotFinalFrame(rootCtx, k.Render(), initial.Seq)
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes -z: kernel turn failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)))
	}
	if final.LastError != "" {
		return newExitCodeError(1, fmt.Errorf("gormes -z: %s", redactRuntimeSecretText(final.LastError, cfg.Hermes.APIKey)))
	}
	content, ok := finalAssistantContent(final.History)
	if !ok {
		return newExitCodeError(1, errors.New("gormes -z: no final assistant content"))
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), content); err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes -z: write stdout: %w", err))
	}
	return nil
}

func readOneshotFrame(ctx context.Context, frames <-chan kernel.RenderFrame) (kernel.RenderFrame, error) {
	select {
	case frame, ok := <-frames:
		if !ok {
			return kernel.RenderFrame{}, errors.New("render stream closed")
		}
		return frame, nil
	case <-ctx.Done():
		return kernel.RenderFrame{}, ctx.Err()
	}
}

func waitForOneshotFinalFrame(ctx context.Context, frames <-chan kernel.RenderFrame, initialSeq uint64) (kernel.RenderFrame, error) {
	for {
		frame, err := readOneshotFrame(ctx, frames)
		if err != nil {
			return kernel.RenderFrame{}, err
		}
		if frame.LastError != "" || frame.Phase == kernel.PhaseFailed {
			return frame, nil
		}
		if frame.Phase == kernel.PhaseIdle && frame.Seq > initialSeq {
			return frame, nil
		}
	}
}

func finalAssistantContent(history []hermes.Message) (string, bool) {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			return history[i].Content, true
		}
	}
	return "", false
}

func runTUI(cmd *cobra.Command, _ []string) error {
	invocation, err := resolveTUIInvocation(cmd)
	if err != nil {
		return err
	}
	return runResolvedTUI(cmd, invocation)
}

func runResolvedTUI(cmd *cobra.Command, invocation tuiInvocation) error {
	return runResolvedTUIWithRuntime(cmd, invocation, rootRuntime{})
}

type tuiProgram interface {
	Run() (tea.Model, error)
	Quit()
}

type tuiProgramFactory func(tea.Model, ...tea.ProgramOption) tuiProgram

func defaultTUIProgramFactory(model tea.Model, options ...tea.ProgramOption) tuiProgram {
	return tea.NewProgram(model, options...)
}

func runResolvedTUIWithRuntime(cmd *cobra.Command, invocation tuiInvocation, runtime rootRuntime) error {
	runNativeTUIStartupPreflight(context.Background(), tuiStartupPreflightOptions{})
	if runtime.tuiProgramFactory == nil {
		runtime.tuiProgramFactory = defaultTUIProgramFactory
	}

	// --remote <url> bypasses the local kernel and provider setup entirely
	// and runs the SSE-backed remote TUI instead. Local Bubble Tea behaviour
	// is preserved when --remote is empty.
	if invocation.RemoteURL != "" {
		return runRemoteTUIWithRuntime(cmd, invocation, runtime)
	}

	cfg := invocation.Config
	modelName := invocation.Inference.Model
	if modelName == "" {
		modelName = cfg.Hermes.Model
	}
	providerName := firstNonEmpty(invocation.Inference.Provider, cfg.Hermes.Provider)

	offline, _ := cmd.Flags().GetBool("offline")
	c := hermes.NewHTTPClientWithProvider(cfg.Hermes.Endpoint, cfg.Hermes.APIKey, providerName)
	if !offline {
		var err error
		c, err = newProviderHTTPClient(cfg, providerName)
		if err != nil {
			redactedErr := redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)
			return newExitCodeError(1, errors.New(formatTUIProviderSetupError(redactedErr, cfg, providerName, modelName)))
		}
	}

	// Phase 2.C — open the session map; honor --resume.
	smap, boltMap, err := openTUISessionMap(cmd)
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()
	if sessionMirror := startSessionIndexMirror(boltMap, slog.Default()); sessionMirror != nil {
		defer sessionMirror.Stop()
	}

	resumeFlag, _ := cmd.Flags().GetString("resume")
	pctx := context.Background()
	key := session.TUIKey()
	if resumeFlag != "" {
		if err := smap.Put(pctx, key, resumeFlag); err != nil {
			slog.Warn("failed to apply --resume override", "err", err)
		}
	}
	var initialSID string
	if sid, err := smap.Get(pctx, key); err != nil {
		slog.Warn("could not load initial session_id", "key", key, "err", err)
	} else {
		initialSID = sid
		if sid != "" {
			slog.Info("resuming persisted session", "key", key, "session_id", sid)
		}
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	tm := telemetry.New()
	toolAudit := audit.NewJSONLWriter(config.ToolAuditLogPath())
	registry := buildDefaultRegistry(rootCtx, cfg, c, modelName)
	k := kernel.New(kernel.Config{
		Model:             modelName,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             registry,
		MaxToolIterations: configuredMaxToolIterations(cfg),
		MaxToolDuration:   30 * time.Second,
		InitialSessionID:  initialSID,
		ToolAudit:         toolAudit,
	}, c, store.NewNoop(), tm, slog.Default())

	go k.Run(rootCtx)

	// Fan-through: read every frame from the kernel, persist its SessionID
	// when it changes, then forward to the TUI. Single consumer invariant
	// preserved — internal/tui's Model remains the only reader of the
	// downstream channel. Buffered cap 1 matches kernel.RenderMailboxCap.
	hookedFrames := make(chan kernel.RenderFrame, 1)
	go func() {
		defer close(hookedFrames)
		var lastSID string
		raw := k.Render()
		for f := range raw {
			if f.SessionID != lastSID {
				if err := smap.Put(rootCtx, key, f.SessionID); err != nil {
					slog.Warn("tui: failed to persist session_id", "key", key, "err", err)
				} else {
					lastSID = f.SessionID
				}
			}
			select {
			case hookedFrames <- f:
			case <-rootCtx.Done():
				return
			}
		}
	}()

	submit := func(text string) {
		_ = k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: text})
	}
	cancelTurn := func() {
		_ = k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel})
	}

	model := tui.NewModelWithOptions(hookedFrames, submit, cancelTurn, tui.Options{
		MouseTracking: cfg.TUI.MouseTracking,
		SessionExport: newTUISaveExportFunc(),
		OfflineSmoke:  offline,
	})
	// Hermes' current Ink TUI runs in an alternate screen by default. The
	// Bubble Tea port mirrors that for the full-screen dashboard so repeated
	// render ticks do not leave stale frame fragments in normal scrollback.
	var programOptions []tea.ProgramOption
	if tui.HermesChromeUseAltScreen() {
		programOptions = append(programOptions, tea.WithAltScreen())
	}
	if cfg.TUI.MouseTracking {
		programOptions = append(programOptions, tea.WithMouseAllMotion())
	}
	prog := runtime.tuiProgramFactory(model, programOptions...)

	// Signal → shutdown-budget force-exit watcher.
	programDone := make(chan struct{})
	go func() {
		<-rootCtx.Done()
		prog.Quit()
		select {
		case <-programDone:
		case <-time.After(kernel.ShutdownBudget):
			slog.Error("shutdown budget exceeded; forcing exit")
			os.Exit(3)
		}
	}()

	_, err = prog.Run()
	close(programDone)
	return err
}

func formatTUIProviderSetupError(detail string, cfg config.Config, providerName, modelName string) string {
	providerName = strings.TrimSpace(providerName)
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		modelName = strings.TrimSpace(cfg.Hermes.Model)
	}
	return strings.Join([]string{
		"Gormes provider setup needed",
		"",
		"Startup cannot contact a model because provider settings are incomplete.",
		"",
		"Detected:",
		"  home:     " + config.GormesHome(),
		"  provider: " + setupDisplayValue(providerName),
		"  model:    " + setupDisplayValue(modelName),
		"",
		"Fix:",
		"  gormes setup model        choose provider/model defaults",
		"  gormes setup provider     add endpoint and API key",
		"  gormes auth add <provider>  add OAuth/API credentials when supported",
		"",
		"Smoke test without a provider:",
		"  gormes --offline",
		"",
		"Advanced config/env:",
		"  hermes.endpoint, hermes.provider, GORMES_ENDPOINT, GORMES_API_KEY",
		"",
		"Details:",
		"  " + friendlyProviderSetupDetail(detail),
	}, "\n")
}

func setupDisplayValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "missing"
	}
	return value
}

func friendlyProviderSetupDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "endpoint unconfigured and no provider declared"):
		return "No provider endpoint or credential-backed provider is configured."
	case strings.Contains(lower, "endpoint unconfigured for provider"):
		return strings.ReplaceAll(detail, "hermes endpoint", "provider endpoint")
	default:
		return strings.ReplaceAll(detail, "hermes endpoint", "provider endpoint")
	}
}

func openTUISessionMap(cmd *cobra.Command) (session.Map, *session.BoltMap, error) {
	path := config.SessionDBPath()
	smap, err := session.OpenBolt(path)
	if err == nil {
		return smap, smap, nil
	}
	if errors.Is(err, session.ErrDBLocked) {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"session persistence unavailable: %v\nrunning TUI with in-memory session state; run `gormes gateway stop` to release the persisted session DB, or `gormes gateway status` to inspect the owner. persisted_path=%s\n",
			err, path)
		return session.NewMemMap(), nil, nil
	}
	return nil, nil, err
}

func redactRuntimeSecretText(text string, secrets ...string) string {
	redacted := text
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "[REDACTED]")
	}
	return redacted
}

func dumpCrash(r any) {
	dir := config.CrashLogDir()
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, fmt.Sprintf("crash-%d.log", time.Now().Unix()))
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "panic:", r)
		fmt.Fprintln(os.Stderr, string(debug.Stack()))
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "panic: %v\n\n%s\n", r, debug.Stack())
	fmt.Fprintln(os.Stderr, "gormes crashed — log at "+path)
}
