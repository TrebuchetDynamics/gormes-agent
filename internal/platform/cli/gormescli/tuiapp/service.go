package tuiapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/tuiadapter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/kanban"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/tuiapp/firstrun"
	tuistartup "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/tuiapp/startup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	tuilocal "github.com/TrebuchetDynamics/gormes-agent/internal/tui/local"
)

var logsHTTPClient = gormescli.NewLogsHTTPClient(5 * time.Second)
var logsEndpointURL = "http://127.0.0.1:43827/api/logs"

// Program is the Bubble Tea program seam used by TUI startup tests and the
// cmd/gormes root shim.
type Program interface {
	Run() (tea.Model, error)
	Quit()
}

// ProgramFactory constructs the terminal program for a prepared TUI model.
type ProgramFactory func(tea.Model, ...tea.ProgramOption) Program

// Runtime carries root-level seams needed to start the native TUI without
// coupling startup behavior back to cmd/gormes.
type Runtime struct {
	ProgramFactory       ProgramFactory
	RunResolvedTUI       func(*cobra.Command, Invocation) error
	Version              string
	KanbanCommandOptions gormescli.KanbanCommandOptions
	GatewayLogTail       tui.GatewayLogTailFunc
	IsTTY                func() bool
	RunFirstRunSetup     func(*cobra.Command) error
	NewExitCodeError     func(int, error) error
}

// Invocation is the resolved root TUI startup request.
type Invocation struct {
	Inference config.TUIInferenceResolution
	Config    config.Config
	// ForcedSkills is a one-turn root CLI skill allowlist. The full TUI does
	// not currently inject it, but carrying the value keeps invocation parsing
	// symmetric with scripted chat startup.
	ForcedSkills []string
	// RemoteURL, when non-empty, switches startup to remote-TUI mode:
	// gormes connects to the gateway's SSE event stream instead of
	// instantiating a local kernel + provider client. Empty leaves local
	// Bubble Tea behavior intact.
	RemoteURL string
	// PromptTemplatePaths are explicit operator-provided Markdown template files
	// or directories for the native TUI. NoPromptTemplates disables discovery.
	PromptTemplatePaths []string
	NoPromptTemplates   bool
}

type exitCodeError struct {
	code int
	err  error
}

// NewExitCodeError wraps err with an ExitCode method for Cobra callers.
func NewExitCodeError(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("exit code %d", code)
	}
	return exitCodeError{code: code, err: err}
}

func (e exitCodeError) Error() string { return e.err.Error() }
func (e exitCodeError) Unwrap() error { return e.err }
func (e exitCodeError) ExitCode() int { return e.code }

// ExitCodeFromError returns the process exit code represented by err.
func ExitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func (runtime Runtime) exitCodeError(code int, err error) error {
	if runtime.NewExitCodeError != nil {
		return runtime.NewExitCodeError(code, err)
	}
	return NewExitCodeError(code, err)
}

// DefaultProgramFactory constructs the real Bubble Tea program.
func DefaultProgramFactory(model tea.Model, options ...tea.ProgramOption) Program {
	return tea.NewProgram(model, options...)
}

// RunRootCommand resolves and starts the default TUI path for a root command.
func RunRootCommand(cmd *cobra.Command, _ []string, runtime Runtime) error {
	restoreKanbanDB := pinCurrentKanbanBoardDBForChat()
	defer restoreKanbanDB()
	invocation, err := ResolveInvocation(cmd)
	if err != nil {
		return err
	}
	if handled, err := MaybeHandleFirstRun(cmd, invocation, runtime); handled || err != nil {
		return err
	}
	if runtime.RunResolvedTUI != nil {
		return runtime.RunResolvedTUI(cmd, invocation)
	}
	return RunResolved(cmd, invocation, runtime)
}

// ResolveInvocation resolves root flags, env overrides, and persisted config
// for native or remote TUI startup.
func ResolveInvocation(cmd *cobra.Command) (Invocation, error) {
	modelFlag := tuistartup.CommandStringFlag(cmd, "model")
	providerFlag := tuistartup.CommandStringFlag(cmd, "provider")
	endpointFlag := tuistartup.CommandStringFlag(cmd, "endpoint")
	apiKeyFlag := tuistartup.CommandStringFlag(cmd, "api-key")
	remoteFlag := tuilocal.ResolveRemoteURL(tuistartup.CommandStringFlag(cmd, "remote"))

	cfg, err := config.Load(nil)
	if err != nil {
		return Invocation{RemoteURL: remoteFlag}, err
	}
	ApplyProviderStartupFlags(&cfg, endpointFlag, apiKeyFlag)
	resolution, err := config.ResolveTUIInference(config.TUIInferenceRequest{
		Config:       cfg,
		ModelFlag:    modelFlag,
		ProviderFlag: providerFlag,
	})
	resolution = ResolveStaticStartupInference(resolution)
	invocation := Invocation{
		Inference:           resolution,
		Config:              cfg,
		ForcedSkills:        tuistartup.ForcedSkillNames(cmd),
		RemoteURL:           remoteFlag,
		PromptTemplatePaths: tuistartup.CommandStringArrayFlag(cmd, "prompt-template"),
		NoPromptTemplates:   tuistartup.CommandBoolFlag(cmd, "no-prompt-templates"),
	}
	if err != nil {
		return invocation, NewExitCodeError(2, err)
	}
	return invocation, nil
}

// ApplyProviderStartupFlags applies invocation-only provider overrides to cfg.
func ApplyProviderStartupFlags(cfg *config.Config, endpointFlag, apiKeyFlag string) {
	tuistartup.ApplyProviderStartupFlags(cfg, endpointFlag, apiKeyFlag)
}

// ResolveStaticStartupInference normalizes known model aliases without making
// provider network calls during startup.
func ResolveStaticStartupInference(resolution config.InferenceResolution) config.InferenceResolution {
	return tuistartup.ResolveStaticStartupInference(resolution)
}

// MaybeHandleFirstRun prints non-interactive setup guidance or runs the setup
// flow before local TUI startup when the terminal target is not ready.
func MaybeHandleFirstRun(cmd *cobra.Command, invocation Invocation, runtime Runtime) (bool, error) {
	if RootFirstRunBypass(cmd, invocation) {
		return false, nil
	}
	interactive := runtime.IsTTY != nil && runtime.IsTTY()
	plan := BuildFirstRunPlanFromConfig(invocation.Config, cli.SetupTargetTerminal, interactive)
	if plan.Ready {
		return false, nil
	}
	if interactive {
		if runtime.RunFirstRunSetup == nil {
			return true, nil
		}
		return true, runtime.RunFirstRunSetup(cmd)
	}
	PrintFirstRunGuidance(cmd.OutOrStdout(), plan)
	return true, nil
}

// RootFirstRunBypass reports whether flags bypass setup-readiness prompting.
func RootFirstRunBypass(cmd *cobra.Command, invocation Invocation) bool {
	if tuistartup.CommandBoolFlag(cmd, "offline") {
		return true
	}
	return strings.TrimSpace(invocation.RemoteURL) != ""
}

// BuildFirstRunPlanFromConfig builds first-run readiness from a loaded config.
func BuildFirstRunPlanFromConfig(cfg config.Config, target cli.SetupTargetID, interactive bool) cli.FirstRunPlan {
	return firstrun.BuildPlanFromConfig(cfg, target, interactive)
}

// PrintFirstRunGuidance writes the root-command first-run guidance text.
func PrintFirstRunGuidance(out io.Writer, plan cli.FirstRunPlan) {
	firstrun.PrintGuidance(out, plan)
}

// FirstRunGuidanceCommand normalizes a setup command for guidance output.
func FirstRunGuidanceCommand(command string) string { return firstrun.GuidanceCommand(command) }

// DetectHermesMigrationSource returns a local Hermes source path when present.
func DetectHermesMigrationSource() string { return firstrun.DetectHermesMigrationSource() }

// DetectOpenClawMigrationSource returns a local OpenClaw source path when present.
func DetectOpenClawMigrationSource() string { return firstrun.DetectOpenClawMigrationSource() }

// ExistingDir returns path when it exists and is a directory.
func ExistingDir(path string) string { return firstrun.ExistingDir(path) }

// RunResolved starts either the remote SSE-backed TUI or the local Bubble Tea
// model wired to a kernel and session persistence.
func RunResolved(cmd *cobra.Command, invocation Invocation, runtime Runtime) error {
	tuilocal.RunNativeStartupPreflight(context.Background(), tuilocal.StartupPreflightOptions{})
	if runtime.ProgramFactory == nil {
		runtime.ProgramFactory = DefaultProgramFactory
	}
	if strings.TrimSpace(runtime.Version) == "" {
		runtime.Version = "unknown"
	}
	if runtime.GatewayLogTail == nil {
		runtime.GatewayLogTail = readLogsTail
	}
	if runtime.KanbanCommandOptions.BuildProvenance == nil && runtime.KanbanCommandOptions.ExitCodeError == nil {
		runtime.KanbanCommandOptions = defaultKanbanCommandOptions()
	}

	// --remote <url> bypasses the local kernel and provider setup entirely
	// and runs the SSE-backed remote TUI instead. Local Bubble Tea behaviour
	// is preserved when --remote is empty.
	if invocation.RemoteURL != "" {
		return gormescli.RunRemoteTUI(context.Background(), cmd.ErrOrStderr(), gormescli.RemoteTUIOptions{
			RemoteURL:     invocation.RemoteURL,
			SidecarURL:    tuilocal.ResolveRemoteSidecarURL(),
			MouseTracking: invocation.Config.TUI.MouseTracking,
			ProgramFactory: func(model tea.Model, options ...tea.ProgramOption) gormescli.TUIProgram {
				return runtime.ProgramFactory(model, options...)
			},
			ModelOptions: func(ctx context.Context) tui.Options {
				return tui.Options{
					MouseTracking:      invocation.Config.TUI.MouseTracking,
					VoiceRecordKey:     invocation.Config.Voice.RecordKey,
					SkillSlashCommands: gormescli.TUISkillSlashCommands(ctx, invocation.Config),
					SkillSlashReload:   gormescli.TUISkillSlashReloadFunc(invocation.Config),
				}
			},
		})
	}

	cfg := invocation.Config
	modelName := invocation.Inference.Model
	if modelName == "" {
		modelName = cfg.Hermes.Model
	}
	providerName := firstNonEmpty(invocation.Inference.Provider, cfg.Hermes.Provider)

	offline := tuistartup.CommandBoolFlag(cmd, "offline")
	c := llm.NewHTTPClientWithProvider(cfg.Hermes.Endpoint, cfg.Hermes.APIKey, providerName)
	if !offline {
		var err error
		c, err = gormescli.NewProviderHTTPClient(cfg, providerName)
		if err != nil {
			redactedErr := redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)
			return runtime.exitCodeError(1, errors.New(formatTUIProviderSetupError(redactedErr, cfg, providerName, modelName)))
		}
	}

	// Phase 2.C — open the session map; honor --resume.
	smap, boltMap, startupNotice, err := openTUISessionMap(cmd)
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()
	if sessionMirror := gormescli.StartSessionIndexMirror(boltMap, slog.Default()); sessionMirror != nil {
		defer sessionMirror.Stop()
	}

	resumeFlag, _ := cmd.Flags().GetString("resume")
	continueFlag, _ := cmd.Flags().GetString("continue")
	if resumeFlag == "" && continueFlag != "" {
		resolved, err := gormescli.ResolveContinueSessionFlag(continueFlag)
		if err != nil {
			return runtime.exitCodeError(1, err)
		}
		resumeFlag = resolved
	}
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
	registry := gormescli.BuildDefaultRegistry(rootCtx, cfg, c, modelName)
	k := kernel.New(kernel.Config{
		Model:             modelName,
		Provider:          cfg.Hermes.Provider,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             registry,
		MaxToolIterations: gormescli.ConfiguredMaxToolIterations(cfg),
		MaxToolDuration:   30 * time.Second,
		InitialSessionID:  initialSID,
		ToolAudit:         toolAudit,
		PrefillMessages:   gormescli.ConfiguredPrefillMessages(cfg),
	}, c, store.NewNoop(), tm, tuiKernelLogger())

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
	steerTurn := func(text string) {
		_ = k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSteer, Text: text})
	}

	welcomeVersion, welcomeToolCount, welcomeToolsets := welcomeStartupSeed(runtime.Version, registry)
	tuiOptions := tui.Options{
		MouseTracking: cfg.TUI.MouseTracking,
		KanbanSlash: func(input string) (string, error) {
			return gormescli.RunTUIKanbanSlashCommand(rootCtx, input, runtime.KanbanCommandOptions)
		},
		PromptTemplates:  gormescli.PromptTemplateCatalog(cfg, "", gormescli.PromptTemplateCatalogOptions{Paths: invocation.PromptTemplatePaths, Disabled: invocation.NoPromptTemplates}),
		GatewayLogTail:   runtime.GatewayLogTail,
		AccountUsage:     tuilocal.NewAccountUsageFunc(cfg),
		OfflineSmoke:     offline,
		StartupNotice:    startupNotice,
		BusyInputMode:    tui.HermesBusyInputMode(cfg.Display.BusyInputMode),
		Steer:            steerTurn,
		WelcomeVersion:   welcomeVersion,
		WelcomeToolCount: welcomeToolCount,
		WelcomeToolsets:  welcomeToolsets,
	}
	tuiadapter.RuntimeBundle{
		Presentation: tuiadapter.PresentationBundle{
			VoiceRecordKey: cfg.Voice.RecordKey,
			VoiceToggle:    tuilocal.NewVoiceToggleFunc(cfg),
			SkinName:       cfg.TUI.Theme,
			SkinConfig:     tuilocal.NewSkinConfigFunc(cfg),
		},
		Model: tuiadapter.ModelBundle{
			SetSessionModel: k.SetSessionModel,
			Catalog:         tui.DefaultModelPickerCatalog,
			Provider:        providerName,
			Name:            modelName,
		},
		ToolSkill: tuiadapter.ToolSkillBundle{
			ToolsConfigure: tuilocal.NewToolsConfigureFunc(),
			SkillsCommand: func(input string) string {
				return gateway.HandleSkillsCommandWithOptions(rootCtx, input, gormescli.SkillsCommandOptionsForConfig(cfg))
			},
			SkillSlashCommands: gormescli.TUISkillSlashCommands(rootCtx, cfg),
			SkillSlashReload:   gormescli.TUISkillSlashReloadFunc(cfg),
		},
		Session: tuiadapter.NewLocalSessionBundle(tuiadapter.LocalSessionBundleOptions{
			RootContext: rootCtx,
			Metadata:    boltMap,
			Resume:      k.ResumeSession,
			Reset:       k.ResetSession,
		}),
	}.Apply(&tuiOptions)
	model := tui.NewModelWithOptions(hookedFrames, submit, cancelTurn, tuiOptions)
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
	prog := runtime.ProgramFactory(model, programOptions...)

	// Signal → shutdown-budget force-exit watcher.
	programDone := make(chan struct{}, 1)
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

func defaultKanbanCommandOptions() gormescli.KanbanCommandOptions {
	return gormescli.KanbanCommandOptions{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "unknown", GitCommit: "unknown"}
		},
		ExitCodeError: NewExitCodeError,
	}
}

func logsCommandOptions() gormescli.LogsCommandOptions {
	return gormescli.LogsCommandOptions{
		Client:      logsHTTPClient,
		EndpointURL: logsEndpointURL,
		LogPath:     config.LogPath(),
	}
}

func readLogsTail(limit int) (string, error) {
	opts := logsCommandOptions()
	content, err := gormescli.ReadLogsContent(opts.Client, opts.EndpointURL, opts.LogPath)
	if err != nil {
		return "", err
	}
	return gormescli.ReadLogsTail(content, limit), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func tuiKernelLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// welcomeStartupSeed returns the operator-facing release version and the
// agent tool count used to seed the session-aware welcome panel.
func welcomeStartupSeed(version string, reg *tools.Registry) (string, int, []string) {
	descs := gormescli.RegistryDescriptors(reg)
	return version, len(descs), welcomeToolsets(descs)
}

func welcomeToolsets(descs []llm.ToolDescriptor) []string {
	seen := map[string]struct{}{}
	for _, desc := range descs {
		for _, toolset := range toolsetsForToolName(desc.Name) {
			seen[toolset] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for toolset := range seen {
		out = append(out, toolset)
	}
	sort.Strings(out)
	return out
}

func toolsetsForToolName(name string) []string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case name == "":
		return nil
	case strings.Contains(name, "browser"), strings.HasPrefix(name, "web_"):
		return []string{"browser"}
	case strings.Contains(name, "clarify"):
		return []string{"clarify"}
	case strings.Contains(name, "execute_code"):
		return []string{"code_execution"}
	case strings.Contains(name, "cron"):
		return []string{"cronjob"}
	case strings.Contains(name, "delegate"):
		return []string{"delegation"}
	case strings.Contains(name, "file"), strings.Contains(name, "patch"):
		return []string{"file"}
	case strings.Contains(name, "homeassistant"):
		return []string{"homeassistant"}
	case strings.Contains(name, "image"):
		return []string{"image_gen"}
	case strings.Contains(name, "kanban"):
		return []string{"kanban"}
	case strings.Contains(name, "memory"):
		return []string{"memory"}
	case strings.Contains(name, "message"):
		return []string{"messaging"}
	case strings.Contains(name, "session_search"):
		return []string{"session_search"}
	case strings.Contains(name, "skill"):
		return []string{"skills"}
	case strings.Contains(name, "terminal"):
		return []string{"terminal"}
	case strings.Contains(name, "todo"):
		return []string{"todo"}
	case strings.Contains(name, "speech"), strings.Contains(name, "tts"), strings.Contains(name, "transcribe"):
		return []string{"tts"}
	case strings.Contains(name, "vision"), strings.Contains(name, "video"):
		return []string{"vision"}
	default:
		return nil
	}
}

func openTUISessionMap(cmd *cobra.Command) (session.Map, *session.BoltMap, string, error) {
	path := config.SessionDBPath()
	smap, err := session.OpenBolt(path)
	if err == nil {
		return smap, smap, "", nil
	}
	if errors.Is(err, session.ErrDBLocked) {
		notice := "session state: in-memory (sessions.db locked; gateway status/stop)"
		return session.NewMemMap(), nil, notice, nil
	}
	if errors.Is(err, session.ErrDBCorrupt) {
		backup, healErr := memory.QuarantineCorruptStateFile(path, nil)
		if healErr != nil {
			return nil, nil, "", fmt.Errorf("%w; self-heal failed: %v", err, healErr)
		}
		smap, retryErr := session.OpenBolt(path)
		if retryErr != nil {
			return nil, nil, "", fmt.Errorf("%w; self-heal backup=%s retry failed: %v", err, backup, retryErr)
		}
		fmt.Fprintf(cmd.ErrOrStderr(),
			"session persistence self-healed: corrupt sessions.db quarantined at %s; recreated persisted session DB at %s\n",
			backup, path)
		return smap, smap, "", nil
	}
	return nil, nil, "", err
}

func redactRuntimeSecretText(text string, secrets ...string) string {
	return tuistartup.RedactRuntimeSecretText(text, secrets...)
}

func formatTUIProviderSetupError(detail string, cfg config.Config, providerName, modelName string) string {
	return tuistartup.FormatProviderSetupError(detail, cfg, providerName, modelName)
}

func setupDisplayValue(value string) string { return tuistartup.SetupDisplayValue(value) }

func friendlyProviderSetupDetail(detail string) string {
	return tuistartup.FriendlyProviderSetupDetail(detail)
}

func pinCurrentKanbanBoardDBForChat() func() {
	current, hadCurrent := os.LookupEnv("GORMES_KANBAN_DB")
	if strings.TrimSpace(current) != "" {
		return func() {}
	}
	board, err := kanban.NewBoardRegistry(config.KanbanHome()).Current()
	if err != nil {
		slog.Debug("chat: kanban board pin unavailable", "err", err)
		return func() {}
	}
	if strings.TrimSpace(board.Path) == "" {
		return func() {}
	}
	if err := os.Setenv("GORMES_KANBAN_DB", board.Path); err != nil {
		slog.Debug("chat: kanban board pin failed", "err", err)
		return func() {}
	}
	return func() {
		if hadCurrent {
			if err := os.Setenv("GORMES_KANBAN_DB", current); err != nil {
				slog.Debug("chat: kanban board pin restore failed", "err", err)
			}
			return
		}
		if err := os.Unsetenv("GORMES_KANBAN_DB"); err != nil {
			slog.Debug("chat: kanban board pin restore failed", "err", err)
		}
	}
}
