// Command gormes is the Go-native Hermes-compatible agent runtime.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	navivoxapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/kanban"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/profileapp"
	tuiapp "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/tuiapp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

var logsHTTPClient = gormescli.NewLogsHTTPClient(5 * time.Second)
var logsEndpointURL = "http://127.0.0.1:43827/api/logs"
var providerPool = gormescli.NewProviderClientPool()

type tuiPickChoice = gormescli.TUIPickChoice

func isStdinTTY() bool {
	return stdinIsTerminal(os.Stdin)
}

func stdinIsTerminal(file *os.File) bool {
	return gormescli.StdinIsTerminal(file)
}

type exitCodeError struct {
	code int
	err  error
}

func newExitCodeError(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("exit code %d", code)
	}
	return exitCodeError{code: code, err: err}
}

func (e exitCodeError) Error() string {
	return e.err.Error()
}

func (e exitCodeError) Unwrap() error {
	return e.err
}

func (e exitCodeError) ExitCode() int {
	return e.code
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var coded interface {
		ExitCode() int
	}
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func runBubbleTeaPick(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, defaultID string) (string, error) {
	return gormescli.RunTUIPick(ctx, stdin, out, prompt, choices, defaultID)
}

func runBubbleTeaPickWithOptions(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, defaultID string, extraOptions ...setupwizard.StepOption) (string, error) {
	return gormescli.RunTUIPickWithOptions(ctx, stdin, out, prompt, choices, defaultID, extraOptions...)
}

func runBubbleTeaChecklist(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, selectedIDs []string) ([]string, error) {
	return gormescli.RunTUIChecklist(ctx, stdin, out, prompt, choices, selectedIDs)
}

func bubbleTeaPickShouldFallback(err error) bool {
	return gormescli.TUIPickShouldFallback(err)
}

// Version marks the current operator-facing release line.
//
// Gormes adopts the Hermes-style dual taxonomy: the canonical semver tag (the
// value below) is paired with a date alias of the form vYYYY.M.D in release
// notes, release.json, and the GitHub release title. The git tag remains
// v<Version> until the release workflow learns to extract version from this
// file independently of the tag string.
var Version = "0.2.24"

// VersionDateAlias is the Hermes-style vYYYY.M.D paired alias for the
// current release. Bumped together with Version on every release. Fleet
// automation (whose own version IS the date) consumes this through
// `gormes version --json`.
var VersionDateAlias = "v2026.6.5"

// GitCommit is the source SHA the binary was built from. Defaults to
// "unknown" in dev/source builds; release CI is expected to inject the
// real value via `-ldflags="-X main.GitCommit=<sha>"`. Fleet automation
// verifying binaries against a specific commit reads this field
// through `gormes version --json`.
var GitCommit = "unknown"

// GitDirty marks whether the source tree had uncommitted changes when
// the binary was built. Stored as a string so the value is settable
// via ldflags; parsed to bool for JSON output. Accepts "true"/"1" as
// dirty, anything else (including the default empty/false) as clean.
// Release CI is expected to inject the actual flag — dev/source
// builds default to clean.
var GitDirty = "false"

// BuildDate is the UTC timestamp when the binary was built. Defaults to
// "unknown" in dev/source builds; release CI injects the real value via
// `-ldflags="-X main.BuildDate=<RFC3339 UTC>"`.
var BuildDate = "unknown"

// buildProvenanceJSON is the `{version, git_commit}` block prepended to
// every `--json` document that reports captured runtime/operator state.
type buildProvenanceJSON = gormescli.VersionBuildProvenance

func versionInfo() gormescli.VersionInfo {
	return gormescli.VersionInfo{
		Version:   Version,
		DateAlias: VersionDateAlias,
		GitCommit: GitCommit,
		GitDirty:  GitDirty,
		BuildDate: BuildDate,
	}
}

func newBuildProvenance() buildProvenanceJSON {
	return gormescli.NewVersionBuildProvenance(versionInfo())
}

func newCommandBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func parseGitDirty(value string) bool {
	return gormescli.ParseGitDirty(value)
}

func resolveGitCommit() string {
	return gormescli.ResolveGitCommit(GitCommit)
}

func resolveGitCommitFrom(injected string, settings []debug.BuildSetting) string {
	return gormescli.ResolveGitCommitFrom(injected, settings)
}

func resolveGitDirty() bool {
	return gormescli.ResolveGitDirty(GitDirty)
}

func resolveGitDirtyFrom(injected string, settings []debug.BuildSetting) bool {
	return gormescli.ResolveGitDirtyFrom(injected, settings)
}

func resolveBuildDate() string {
	return gormescli.ResolveBuildDate(BuildDate)
}

func resolveBuildDateFrom(injected string, settings []debug.BuildSetting) string {
	return gormescli.ResolveBuildDateFrom(injected, settings)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			dumpCrash(r)
			os.Exit(2)
		}
	}()

	root := newRootCommand()
	args := gormescli.SanitizeTermuxExecArgs(os.Args[1:])
	if err := executeRootCommand(root, args...); err != nil {
		os.Exit(exitCodeFromError(err))
	}
}

func executeRootCommand(root *cobra.Command, args ...string) error {
	return gormescli.ExecuteRootCommand(root, args, gormescli.RootExecutionOptions{
		BuildProvenance: newCommandBuildProvenance,
		ExitCodeError:   newExitCodeError,
		HandledExitError: func(err error) bool {
			return errors.As(err, new(exitCodeError))
		},
	})
}

func newRootCommand() *cobra.Command {
	return newRootCommandWithRuntime(rootRuntime{})
}

type rootRuntime struct {
	runTUI                 func(*cobra.Command, []string) error
	runResolvedTUI         func(*cobra.Command, tuiInvocation) error
	runOneshot             func(*cobra.Command, oneshotInvocation) error
	runRPC                 func(*cobra.Command, rpcInvocation) error
	newOneshotClient       oneshotClientFactory
	configureOneshotKernel oneshotKernelConfigurer
	tuiProgramFactory      tuiProgramFactory
	isTTY                  func() bool
	runFirstRunSetup       func(*cobra.Command) error
	sendMessage            gormescli.SendDeliveryFunc
}

type tuiInvocation = tuiapp.Invocation

type oneshotInvocation struct {
	Prompt       string
	Inference    config.OneshotInferenceResolution
	Config       config.Config
	ForcedSkills []string
}

type rpcInvocation struct {
	Inference    config.TUIInferenceResolution
	Config       config.Config
	ForcedSkills []string
	NoSession    bool
}

func newRootCommandWithRuntime(runtime rootRuntime) *cobra.Command {
	if runtime.isTTY == nil {
		runtime.isTTY = isStdinTTY
	}
	if runtime.runFirstRunSetup == nil {
		runtime.runFirstRunSetup = runFirstRunSetupCommand
	}
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
	if runtime.runRPC == nil {
		runtime.runRPC = runResolvedRPC
	}
	root := gormescli.NewRootCommand(gormescli.RootOptions{
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return applyProfileStartupFlag(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRootCommand(cmd, args, runtime)
		},
		Finalizers: []func(*cobra.Command){
			func(cmd *cobra.Command) {
				gormescli.InstallParentUnknownSubcommandGuards(cmd, gormescli.ParentUnknownSubcommandGuardOptions{
					BuildProvenance: newCommandBuildProvenance,
					ExitCodeError:   newExitCodeError,
				})
			},
			func(cmd *cobra.Command) {
				gormescli.InstallVisibleHelpCommand(cmd, gormescli.VisibleHelpCommandOptions{
					ExitCodeError: newExitCodeError,
				})
			},
			gormescli.InstallRootHelpRenderer,
		},
	}, rootCommandFactories(runtime))
	gormescli.InstallRootRPCModeFlags(root)
	return root
}

func tuiAppRuntime(runtime rootRuntime) tuiapp.Runtime {
	return tuiapp.Runtime{
		ProgramFactory:       runtime.tuiProgramFactory,
		Version:              Version,
		KanbanCommandOptions: kanbanCommandOptions(),
		GatewayLogTail:       readLogsTail,
		IsTTY:                runtime.isTTY,
		RunFirstRunSetup:     runtime.runFirstRunSetup,
		NewExitCodeError:     newExitCodeError,
	}
}

func runFirstRunSetupCommand(cmd *cobra.Command) error {
	setup := newSetupCommand()
	setup.SetOut(cmd.OutOrStdout())
	setup.SetErr(cmd.ErrOrStderr())
	setup.SetIn(cmd.InOrStdin())
	setup.SetArgs([]string{})
	return setup.ExecuteContext(cmd.Context())
}

func rootCommandFactories(runtime rootRuntime) gormescli.CommandFactories {
	return gormescli.CommandFactories{
		"doctor":   newDoctorCommand,
		"version":  func() *cobra.Command { return gormescli.NewVersionCommand(versionInfo()) },
		"telegram": newTelegramCommand,
		"gateway":  newGatewayCommand,
		"channels": newChannelsCommand,
		"whatsapp": newWhatsAppCommand,
		"slack":    gormescli.NewSlackCommand,
		"session":  newSessionCommand,
		"memory":   newMemoryCommand,
		"goncho":   newGonchoCommand,
		"kanban": func() *cobra.Command {
			return gormescli.NewKanbanCommand(kanbanCommandOptions())
		},
		"chat": func() *cobra.Command { return newChatCommand(runtime) },
		"send": func() *cobra.Command {
			isTTY := isStdinTTY
			if runtime.isTTY != nil {
				isTTY = runtime.isTTY
			}
			return gormescli.NewSendCommand(gormescli.SendOptions{Deliver: runtime.sendMessage, IsStdinTTY: isTTY})
		},
		"curator": func() *cobra.Command {
			return gormescli.NewCuratorCommandWithDeps(gormescli.CuratorCommandDeps{}, func() gormescli.BuildProvenance {
				build := newBuildProvenance()
				return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
			})
		},
		"acp": newACPCommand,
		"system": func() *cobra.Command {
			return gormescli.NewSystemCommand(func() gormescli.BuildProvenance {
				provenance := newBuildProvenance()
				return gormescli.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
			})
		},
		"agent":   newAgentCommand,
		"navivox": func() *cobra.Command { return navivoxapp.NewCommand(navivoxapp.CommandOptions{}) },
		"usage":   func() *cobra.Command { return providermodule.NewUsageCommand(providerCommandOptions()) },
		"tts":     gormescli.NewTTSCommand,
		"status": func() *cobra.Command {
			return gormescli.NewStatusCommand(gormescli.StatusCommandOptions{
				BuildProvenance: func() gormescli.BuildProvenance {
					build := newBuildProvenance()
					return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
				},
				SystemSnapshot: func(ctx context.Context) (tools.SystemEventsSnapshot, error) {
					return gormescli.DefaultSystemEventsManager().Snapshot(ctx)
				},
			})
		},
		"auth":      func() *cobra.Command { return providermodule.NewAuthCommand(providerCommandOptions()) },
		"providers": func() *cobra.Command { return providermodule.NewProvidersCommand(providerCommandOptions()) },
		"logout": func() *cobra.Command {
			return gormescli.NewLogoutCommand(gormescli.LogoutSeams{
				NormalizeAuthProvider: providermodule.NormalizeAuthProvider,
				ConfiguredProvider: func() (string, error) {
					return gormescli.ConfiguredLogoutProvider(providermodule.NormalizeAuthProvider)
				},
				RunAuthLogout: providermodule.RunAuthLogoutCommand,
				ResetProviderIfMatching: func(provider string) error {
					return gormescli.ResetLogoutProviderIfMatching(provider, providermodule.NormalizeAuthProvider)
				},
			}, gormescli.LogoutOptions{
				BuildProvenance: func() gormescli.BuildProvenance {
					build := newBuildProvenance()
					return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
				},
			})
		},
		"config": func() *cobra.Command {
			return gormescli.NewConfigCommand(func() gormescli.BuildProvenance {
				build := newBuildProvenance()
				return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
			})
		},
		"fallback": func() *cobra.Command {
			return providermodule.NewFallbackCommandWithSeams(gormescli.DefaultModelCommandSeams())
		},
		"router": gormescli.NewRouterCommand,
		"fidelity": func() *cobra.Command {
			return gormescli.NewFidelityCommand(gormescli.FidelityCommandOptions{
				Build: func() any {
					return newBuildProvenance()
				},
				ExitCodeError: newExitCodeError,
			})
		},
		"secrets": func() *cobra.Command {
			return gormescli.NewSecretsCommand(gormescli.SecretsOptions{
				BuildProvenance: func() gormescli.SecretsBuildProvenance {
					build := newBuildProvenance()
					return gormescli.SecretsBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
				},
			})
		},
		"security": func() *cobra.Command {
			return gormescli.NewSecurityCommand(gormescli.SecurityCommandOptions{
				BuildProvenance: func() gormescli.SecurityBuildProvenance {
					build := newBuildProvenance()
					return gormescli.SecurityBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
				},
				ExitCodeError: newExitCodeError,
			})
		},
		"migrate": newMigrateCommand,
		"claw":    newClawCommand,
		"profile": newProfileCommand,
		"model":   gormescli.NewModelCommand,
		"setup":   newSetupCommand,
		"skills":  newSkillsCommand,
		"plugins": func() *cobra.Command {
			return gormescli.NewPluginsCommand(gormescli.PluginsOptions{
				BuildProvenance: func() gormescli.PluginsBuildProvenance {
					build := newBuildProvenance()
					return gormescli.PluginsBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
				},
			})
		},
		"mcp": func() *cobra.Command {
			return gormescli.NewMCPCommand(gormescli.MCPCommandOptions{
				BuildProvenance: func() gormescli.BuildProvenance {
					build := newBuildProvenance()
					return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
				},
				ExitCodeError: newExitCodeError,
			})
		},
		"dashboard": func() *cobra.Command {
			return gormescli.NewDashboardCommand(gormescli.DefaultDashboardCommandOptions(Version, resolveGitCommit(), resolveGitDirty()))
		},
		"update": func() *cobra.Command {
			return gormescli.NewUpdateCommand(gormescli.UpdateCommandOptions{BuildProvenance: func() gormescli.BuildProvenance {
				build := newBuildProvenance()
				return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
			}})
		},
		"restore": func() *cobra.Command {
			return gormescli.NewRestoreCommand(gormescli.RestoreCommandOptions{BuildProvenance: func() gormescli.BuildProvenance {
				build := newBuildProvenance()
				return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
			}})
		},
		"uninstall": func() *cobra.Command {
			return gormescli.NewUninstallCommand(func() gormescli.BuildProvenance {
				build := newBuildProvenance()
				return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
			})
		},
		"logs": func() *cobra.Command {
			return gormescli.NewLogsCommand(func() gormescli.BuildProvenance {
				build := newBuildProvenance()
				return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
			}, logsCommandOptions())
		},
		"checkpoints": func() *cobra.Command {
			return gormescli.NewCheckpointsCommand(func() gormescli.BuildProvenance {
				provenance := newBuildProvenance()
				return gormescli.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
			})
		},
		"completion": gormescli.NewShellCompletionCommand,
		"cron":       newCronCommand,
		"webhook":    newWebhookCommand,
		"hooks":      newHooksCommand,
		"dump":       func() *cobra.Command { return gormescli.NewDumpCommand(hermesUnavailableOptions()) },
		"debug":      func() *cobra.Command { return gormescli.NewDebugCommand(hermesUnavailableOptions()) },
		"backup":     func() *cobra.Command { return gormescli.NewBackupCommand(hermesUnavailableOptions()) },
		"import":     func() *cobra.Command { return gormescli.NewImportCommand(hermesUnavailableOptions()) },
		"pairing":    newPairingCommand,
		"tools":      func() *cobra.Command { return gormescli.NewToolsCommand(gormescli.ToolsCommandOptions{}) },
		"insights":   func() *cobra.Command { return providermodule.NewInsightsCommand(providerCommandOptions()) },
		"admin": func() *cobra.Command {
			return gormescli.NewAdminCommand(gormescli.AdminCommandOptions{
				Runner: gormescli.AdminCommandRunner(gormescli.AdminRunnerOptions{
					ExecuteCommand: func(args []string) (string, string, error) {
						root := newRootCommandWithRuntime(rootRuntime{})
						var stdout, stderr bytes.Buffer
						root.SetOut(&stdout)
						root.SetErr(&stderr)
						root.SetIn(strings.NewReader(""))
						err := executeRootCommand(root, args...)
						return stdout.String(), stderr.String(), err
					},
					ExitCode: exitCodeFromError,
				}),
			})
		},
		"bridge":    gormescli.NewBridgeCommand,
		"bootstrap": gormescli.NewBootstrapCommand,
	}
}

func newWhatsAppCommand() *cobra.Command {
	return channelsmodule.NewWhatsAppAppCommand(whatsappCommandOptions())
}

func newACPCommand() *cobra.Command {
	return gormescli.NewACPCommand(gormescli.ACPCommandOptions{
		BuildProvenance: acpBuildProvenance,
		ExitError:       newExitCodeError,
	})
}

func acpBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func newAgentCommand() *cobra.Command {
	return gormescli.NewAgentCommand(agentCommandOptions())
}

func agentCommandOptions() gormescli.AgentOptions {
	return gormescli.AgentOptions{
		DefaultResetTarget: config.GormesHome(),
		BuildProvenance: func() gormescli.AgentBuildProvenance {
			build := newBuildProvenance()
			return gormescli.AgentBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
		},
		OpenRegistry: openDynamicAgentRegistry,
	}
}

func openDynamicAgentRegistry() (gormescli.AgentRegistry, func(), error) {
	path := config.MemoryDBPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, func() {}, fmt.Errorf("gormes agent: create memory dir: %w", err)
	}
	db, err := sqlOpenGoncho(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("gormes agent: open registry db: %w", err)
	}
	reg, err := goncho.NewDynamicAgentRegistry(db)
	if err != nil {
		_ = db.Close()
		return nil, func() {}, err
	}
	return reg, func() { _ = db.Close() }, nil
}

func newChannelsCommand() *cobra.Command {
	return channelsmodule.NewCommandWithSeams(channelsCommandSeams(), channelsCommandOptions())
}

type skillsProfileSyncSeams = gormescli.SkillsProfileSyncSeams

func newSkillsCommand() *cobra.Command {
	return newSkillsCommandWithProfileSync(skillsProfileSyncSeams{})
}

func newSkillsCommandWithProfileSync(syncSeams skillsProfileSyncSeams) *cobra.Command {
	return gormescli.NewSkillsCommand(gormescli.SkillsCLICommandOptions{
		SyncSeams:       syncSeams,
		BuildProvenance: skillsBuildProvenance,
		Row:             gormescli.HermesSkillsRow,
		UnavailableCommand: func(spec gormescli.RowBackedCommandSpec) *cobra.Command {
			return newHermesUnavailableCommand(spec)
		},
	})
}

func skillsCommandOptionsForConfig(cfg config.Config) gateway.SkillsCommandOptions {
	return gormescli.SkillsCommandOptionsForConfig(cfg)
}

func defaultSkillSyncProfiles() ([]skills.SkillProfileRoot, error) {
	return gormescli.DefaultSkillProfileRoots()
}

func skillsBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

type hermesUnavailableCommandSpec = gormescli.RowBackedCommandSpec

func newHermesUnavailableCommand(spec hermesUnavailableCommandSpec, children ...*cobra.Command) *cobra.Command {
	return gormescli.NewRowBackedCommand(spec, hermesUnavailableOptions(), children...)
}

func hermesUnavailableOptions() gormescli.RowBackedCommandOptions {
	return gormescli.RowBackedCommandOptions{BuildProvenance: func() gormescli.BuildProvenance {
		build := newBuildProvenance()
		return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
	}}
}

func hermesUnavailableYesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
}

type profileCommandSeams = profileapp.CommandSeams

func newProfileCommand() *cobra.Command {
	return profileapp.NewCommand(profileBuildProvenance)
}

func newProfileCommandWithSeams(seams profileCommandSeams) *cobra.Command {
	return profileapp.NewCommandWithSeams(seams, profileCommandOptions())
}

func defaultProfileCommandSeams() profileCommandSeams {
	return profileapp.DefaultSeams()
}

func defaultListKnownProfiles() ([]string, error) {
	return profileapp.DefaultListKnownProfiles()
}

func profileCommandOptions() profileapp.CommandOptions {
	return profileapp.CommandOptions{BuildProvenance: profileBuildProvenance}
}

func profileBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func newMigrateCommand() *cobra.Command {
	return gormescli.NewMigrateCommand(migrateCommandOptions())
}

func newClawCommand() *cobra.Command {
	return gormescli.NewClawCommand(migrateCommandOptions())
}

func migrateCommandOptions() gormescli.MigrateCommandOptions {
	return gormescli.MigrateCommandOptions{
		BuildProvenance: migrateBuildProvenance,
		ExitCodeError:   newExitCodeError,
	}
}

func migrateBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func newCronCommand() *cobra.Command {
	return gormescli.NewCronCommand(gormescli.CronCommandOptions{
		Rows: gormescli.CronCommandRows{
			Create: gormescli.RowBackedCommandSpec{
				Use:     "create",
				Aliases: []string{"add"},
				Short:   "Create a scheduled cron job",
				Row:     gormescli.HermesGatewayCronRow,
			},
			Edit: gormescli.RowBackedCommandSpec{
				Use:   "edit <job-id>",
				Short: "Edit a scheduled cron job",
				Row:   gormescli.HermesGatewayCronRow,
			},
			Pause: gormescli.RowBackedCommandSpec{
				Use:   "pause <job-id>",
				Short: "Pause a scheduled cron job",
				Row:   gormescli.HermesGatewayCronRow,
			},
			Resume: gormescli.RowBackedCommandSpec{
				Use:   "resume <job-id>",
				Short: "Resume a scheduled cron job",
				Row:   gormescli.HermesGatewayCronRow,
			},
			Run: gormescli.RowBackedCommandSpec{
				Use:   "run <job-id>",
				Short: "Run a scheduled cron job now",
				Row:   gormescli.HermesGatewayCronRow,
			},
			Tick: gormescli.RowBackedCommandSpec{
				Use:   "tick",
				Short: "Run one scheduler tick",
				Row:   gormescli.HermesGatewayCronRow,
			},
		},
		UnavailableCommand: func(spec gormescli.RowBackedCommandSpec) *cobra.Command {
			return newHermesUnavailableCommand(spec)
		},
	})
}

func newMemoryCommand() *cobra.Command {
	return gormescli.NewMemoryCommand(gormescli.MemoryCommandOptions{
		Status: memoryCommandOptions(),
		Rows: gormescli.MemoryCommandRows{
			Setup: gormescli.RowBackedCommandSpec{
				Use:   "setup",
				Short: "Configure Hermes-compatible memory",
				Row:   gormescli.HermesMemoryRow,
			},
			Off: gormescli.RowBackedCommandSpec{
				Use:   "off",
				Short: "Disable Hermes-compatible memory",
				Row:   gormescli.HermesMemoryRow,
			},
			Reset: gormescli.RowBackedCommandSpec{
				Use:         "reset",
				Short:       "Reset Hermes-compatible memory state",
				Row:         gormescli.HermesMemoryRow,
				Destructive: true,
				FlagSet:     hermesUnavailableYesFlag,
			},
		},
		UnavailableCommand: func(spec gormescli.RowBackedCommandSpec) *cobra.Command {
			return newHermesUnavailableCommand(spec)
		},
	})
}

func memoryCommandOptions() gormescli.MemoryCommandStatusOptions {
	return gormescli.MemoryCommandStatusOptions{
		BuildProvenance: memoryBuildProvenance,
		OpenDB:          sqlOpenGoncho,
	}
}

func memoryBuildProvenance() gormescli.MemoryBuildProvenance {
	build := newBuildProvenance()
	return gormescli.MemoryBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func newSessionCommand() *cobra.Command {
	return gormescli.NewSessionCommand(gormescli.SessionCommandOptions{
		Build: func() gormescli.SessionBuildProvenance {
			build := newBuildProvenance()
			return gormescli.SessionBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
		},
		UnavailableCommand: func(spec gormescli.SessionUnavailableCommandSpec) *cobra.Command {
			return newHermesUnavailableCommand(hermesUnavailableCommandSpec{Use: spec.Use, Short: spec.Short, Row: spec.Row})
		},
	})
}

func channelsCommandSeams() channelsmodule.Seams {
	return channelsmodule.Seams{
		LoadConfig:        func() (config.Config, error) { return config.Load(nil) },
		ConfiguredDetails: gormescli.ConfiguredChannelCapabilityDetails,
	}
}

func channelsCommandOptions() channelsmodule.Options {
	return channelsmodule.Options{BuildProvenance: func() gormescli.BuildProvenance {
		build := newBuildProvenance()
		return gormescli.BuildProvenance{
			Version:   build.Version,
			GitCommit: build.GitCommit,
		}
	}}
}

func whatsappCommandOptions() channelsmodule.WhatsAppAppOptions {
	return channelsmodule.WhatsAppAppOptions{BuildProvenance: func() channelsmodule.WhatsAppBuildProvenance {
		build := newBuildProvenance()
		return channelsmodule.WhatsAppBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
	}}
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

func kanbanCommandOptions() gormescli.KanbanCommandOptions {
	return gormescli.KanbanCommandOptions{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
		},
		ExitCodeError: newExitCodeError,
	}
}

func providerCommandOptions() providermodule.Options {
	return providermodule.Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
	}
}

func applyProfileStartupFlag(cmd *cobra.Command) error {
	if err := gormescli.RejectGatewayProfileStartupFlag(cmd, gormescli.GatewayProfileStartupGuardOptions{ExitCodeError: newExitCodeError}); err != nil {
		return err
	}
	if gormescli.CommandIsGateway(cmd) {
		return nil
	}
	baseHome := config.GormesBaseHome()
	name := strings.TrimSpace(commandStringFlag(cmd, "profile"))
	if name == "" {
		if commandSkipsStickyActiveProfile(cmd) {
			return nil
		}
		active, err := cli.ReadActiveProfile(filepath.Join(baseHome, "active_profile"))
		if err != nil {
			if errors.Is(err, cli.ErrActiveProfileUnset) {
				return nil
			}
			return newExitCodeError(2, fmt.Errorf("profile: read active profile: %w", err))
		}
		name = active
	}
	if err := cli.ValidateProfileName(name); err != nil {
		return newExitCodeError(2, fmt.Errorf("profile_name_invalid: %w", err))
	}
	root, err := startupProfileRoot(baseHome, name)
	if err != nil {
		return newExitCodeError(2, err)
	}
	if err := os.Setenv("GORMES_HOME", root); err != nil {
		return newExitCodeError(2, fmt.Errorf("profile: set GORMES_HOME: %w", err))
	}
	return nil
}

func startupProfileRoot(baseHome, name string) (string, error) {
	root, err := cli.ResolveProfileRuntimeRoot(baseHome, name)
	if err != nil {
		return "", fmt.Errorf("profile: resolve root: %w", err)
	}
	return root, nil
}

func commandSkipsStickyActiveProfile(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "profile", "config", "gateway":
			return true
		}
	}
	return false
}

func newChatCommand(runtime rootRuntime) *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "chat [prompt...]",
		Short: "Open chat or send a single query",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			restoreKanbanDB := pinCurrentKanbanBoardDBForChat()
			defer restoreKanbanDB()
			prompt := strings.TrimSpace(query)
			if prompt == "" && len(args) > 0 {
				prompt = strings.TrimSpace(strings.Join(args, " "))
			}
			if prompt == "" {
				invocation, err := resolveTUIInvocation(cmd)
				if err != nil {
					return err
				}
				return runtime.runResolvedTUI(cmd, invocation)
			}
			invocation, err := resolveOneshotInvocationForPrompt(cmd, prompt)
			if err != nil {
				return err
			}
			return runtime.runOneshot(cmd, invocation)
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "send one chat query and exit")
	return cmd
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

func runResolvedRPC(cmd *cobra.Command, invocation rpcInvocation) error {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runtime := gateway.NewKernelRPCRuntime(gateway.KernelRPCRuntimeOptions{
		Inference:         invocation.Inference,
		Config:            invocation.Config,
		ForcedSkills:      invocation.ForcedSkills,
		NoSession:         invocation.NoSession,
		ProviderClient:    providerPool.GetOrCreate,
		MaxToolIterations: gormescli.ConfiguredMaxToolIterations(invocation.Config),
		PrefillMessages:   gormescli.ConfiguredPrefillMessages(invocation.Config),
		RedactSecretText:  redactRuntimeSecretText,
	})
	defer runtime.Shutdown()
	if err := gateway.RunRPCMode(rootCtx, gateway.RPCModeOptions{
		In:      cmd.InOrStdin(),
		Out:     cmd.OutOrStdout(),
		Runtime: runtime,
	}); err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes rpc: %w", err))
	}
	return nil
}

func runRootCommand(cmd *cobra.Command, args []string, runtime rootRuntime) error {
	restoreKanbanDB := pinCurrentKanbanBoardDBForChat()
	defer restoreKanbanDB()
	if strings.TrimSpace(commandStringFlag(cmd, "mode")) != "" {
		invocation, err := resolveRPCInvocation(cmd)
		if err != nil {
			return err
		}
		return runtime.runRPC(cmd, invocation)
	}
	invocation, err := resolveTUIInvocation(cmd)
	if err != nil {
		return err
	}
	if handled, err := tuiapp.MaybeHandleFirstRun(cmd, invocation, tuiAppRuntime(runtime)); handled || err != nil {
		return err
	}
	return runtime.runResolvedTUI(cmd, invocation)
}

func resolveRPCInvocation(cmd *cobra.Command) (rpcInvocation, error) {
	mode := strings.TrimSpace(commandStringFlag(cmd, "mode"))
	if mode != "rpc" {
		return rpcInvocation{}, newExitCodeError(2, fmt.Errorf("unsupported mode %q; supported modes: rpc", mode))
	}
	modelFlag := commandStringFlag(cmd, "model")
	providerFlag := commandStringFlag(cmd, "provider")
	endpointFlag := commandStringFlag(cmd, "endpoint")
	apiKeyFlag := commandStringFlag(cmd, "api-key")
	cfg, err := config.Load(nil)
	if err != nil {
		return rpcInvocation{}, err
	}
	forcedSkills := forcedSkillNames(cmd)
	if err := validateForcedSkills(cfg, forcedSkills); err != nil {
		return rpcInvocation{Config: cfg, ForcedSkills: forcedSkills}, newExitCodeError(2, err)
	}
	applyProviderStartupFlags(&cfg, endpointFlag, apiKeyFlag)
	resolution, err := config.ResolveTUIInference(config.TUIInferenceRequest{
		Config:       cfg,
		ModelFlag:    modelFlag,
		ProviderFlag: providerFlag,
	})
	resolution = resolveStaticStartupInference(resolution)
	invocation := rpcInvocation{
		Inference:    resolution,
		Config:       cfg,
		ForcedSkills: forcedSkills,
		NoSession:    commandBoolFlag(cmd, "no-session"),
	}
	if err != nil {
		return invocation, newExitCodeError(2, err)
	}
	return invocation, nil
}

func commandBoolFlag(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	if flags := cmd.Flags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetBool(name)
		return value
	}
	if flags := cmd.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetBool(name)
		return value
	}
	if flags := cmd.InheritedFlags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetBool(name)
		return value
	}
	if root := cmd.Root(); root != nil && root != cmd {
		if flags := root.Flags(); flags != nil && flags.Lookup(name) != nil {
			value, _ := flags.GetBool(name)
			return value
		}
		if flags := root.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
			value, _ := flags.GetBool(name)
			return value
		}
	}
	return false
}

func resolveOneshotInvocationForPrompt(cmd *cobra.Command, prompt string) (oneshotInvocation, error) {
	modelFlag := commandStringFlag(cmd, "model")
	providerFlag := commandStringFlag(cmd, "provider")
	endpointFlag := commandStringFlag(cmd, "endpoint")
	apiKeyFlag := commandStringFlag(cmd, "api-key")

	cfg, err := config.Load(nil)
	if err != nil {
		return oneshotInvocation{Prompt: prompt}, err
	}
	forcedSkills := forcedSkillNames(cmd)
	if err := validateForcedSkills(cfg, forcedSkills); err != nil {
		return oneshotInvocation{Prompt: prompt, Config: cfg, ForcedSkills: forcedSkills}, newExitCodeError(2, err)
	}
	applyProviderStartupFlags(&cfg, endpointFlag, apiKeyFlag)
	resolution, err := config.ResolveOneshotInference(config.OneshotInferenceRequest{
		Config:       cfg,
		ModelFlag:    modelFlag,
		ProviderFlag: providerFlag,
	})
	resolution = resolveStaticStartupInference(resolution)
	invocation := oneshotInvocation{
		Prompt:       prompt,
		Inference:    resolution,
		Config:       cfg,
		ForcedSkills: forcedSkills,
	}
	if err != nil {
		return invocation, newExitCodeError(2, err)
	}
	return invocation, nil
}

func resolveTUIInvocation(cmd *cobra.Command) (tuiInvocation, error) {
	return tuiapp.ResolveInvocation(cmd)
}

func applyProviderStartupFlags(cfg *config.Config, endpointFlag, apiKeyFlag string) {
	if endpoint := strings.TrimSpace(endpointFlag); endpoint != "" {
		cfg.Hermes.Endpoint = endpoint
	}
	if apiKey := strings.TrimSpace(apiKeyFlag); apiKey != "" {
		cfg.Hermes.APIKey = apiKey
	}
}

func commandStringFlag(cmd *cobra.Command, name string) string {
	if cmd == nil {
		return ""
	}
	if flags := cmd.Flags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetString(name)
		return value
	}
	if flags := cmd.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetString(name)
		return value
	}
	if flags := cmd.InheritedFlags(); flags != nil && flags.Lookup(name) != nil {
		value, _ := flags.GetString(name)
		return value
	}
	if root := cmd.Root(); root != nil && root != cmd {
		if flags := root.Flags(); flags != nil && flags.Lookup(name) != nil {
			value, _ := flags.GetString(name)
			return value
		}
		if flags := root.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
			value, _ := flags.GetString(name)
			return value
		}
	}
	return ""
}

func forcedSkillNames(cmd *cobra.Command) []string {
	raw := commandStringArrayFlag(cmd, "skills")
	if len(raw) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		for _, part := range strings.Split(value, ",") {
			name := strings.TrimSpace(part)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

func commandStringArrayFlag(cmd *cobra.Command, name string) []string {
	if cmd == nil {
		return nil
	}
	if flags := cmd.Flags(); flags != nil && flags.Lookup(name) != nil {
		values, _ := flags.GetStringArray(name)
		return values
	}
	if flags := cmd.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
		values, _ := flags.GetStringArray(name)
		return values
	}
	if flags := cmd.InheritedFlags(); flags != nil && flags.Lookup(name) != nil {
		values, _ := flags.GetStringArray(name)
		return values
	}
	if root := cmd.Root(); root != nil && root != cmd {
		if flags := root.Flags(); flags != nil && flags.Lookup(name) != nil {
			values, _ := flags.GetStringArray(name)
			return values
		}
		if flags := root.PersistentFlags(); flags != nil && flags.Lookup(name) != nil {
			values, _ := flags.GetStringArray(name)
			return values
		}
	}
	return nil
}

func validateForcedSkills(cfg config.Config, names []string) error {
	if len(names) == 0 {
		return nil
	}
	snapshot, err := skills.NewStore(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes).SnapshotActive()
	if err != nil {
		return fmt.Errorf("skills_unavailable: %w", err)
	}
	available := map[string]struct{}{}
	for _, skill := range snapshot.Skills {
		for _, key := range forcedSkillPolicyKeys(skill) {
			available[key] = struct{}{}
		}
	}
	var missing []string
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			continue
		}
		if _, ok := available[key]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("skill_unavailable: %s", strings.Join(missing, ", "))
	}
	return nil
}

func forcedSkillPolicyKeys(skill skills.Skill) []string {
	var keys []string
	add := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			keys = append(keys, value)
			keys = append(keys, strings.ReplaceAll(value, " ", "-"))
		}
	}
	add(skill.Name)
	if skill.Path != "" {
		add(filepath.Base(filepath.Dir(skill.Path)))
	}
	return keys
}

func resolveStaticStartupInference(resolution config.InferenceResolution) config.InferenceResolution {
	if resolution.Model == "" {
		return resolution
	}
	metadata := llm.LookupModelMetadata(llm.ModelRegistryQuery{
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

type oneshotClientFactory func(context.Context, config.Config, oneshotInvocation) (llm.Client, error)
type oneshotKernelConfigurer func(*kernel.Config)

func newOneshotHTTPClient(_ context.Context, cfg config.Config, invocation oneshotInvocation) (llm.Client, error) {
	return providerPool.GetOrCreate(cfg, invocation.Inference.Provider)
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
		return newExitCodeError(1, fmt.Errorf("gormes chat -q: provider setup failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)))
	}
	if client == nil {
		return newExitCodeError(1, fmt.Errorf("gormes chat -q: provider setup failed: %w", errors.New("nil hermes client")))
	}

	toolSafety, err := kernel.NewOneshotToolSafetyPolicy(kernel.OneshotToolSafetyOptions{
		TrustClass: kernel.TrustClassOperator,
	})
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes chat -q: safety policy setup failed: %w", err))
	}
	kernelCfg := kernel.Config{
		Model:             model,
		Provider:          cfg.Hermes.Provider,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		MaxToolIterations: gormescli.ConfiguredMaxToolIterations(cfg),
		ToolAudit:         audit.NewJSONLWriter(config.ToolAuditLogPath()),
		ToolSafety:        toolSafety,
		PrefillMessages:   gormescli.ConfiguredPrefillMessages(cfg),
	}
	if skillRuntime := newForcedSkillRuntime(cfg, invocation.ForcedSkills); skillRuntime != nil {
		kernelCfg.Skills = skillRuntime
		kernelCfg.SkillUsage = skillRuntime
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
		return newExitCodeError(1, fmt.Errorf("gormes chat -q: kernel startup failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)))
	}
	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: invocation.Prompt}); err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes chat -q: submit failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)))
	}
	final, err := waitForOneshotFinalFrame(rootCtx, k.Render(), initial.Seq)
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes chat -q: kernel turn failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey)))
	}
	if final.LastError != "" {
		return newExitCodeError(1, fmt.Errorf("gormes chat -q: %s", redactRuntimeSecretText(final.LastError, cfg.Hermes.APIKey)))
	}
	content, ok := finalAssistantContent(final.History)
	if !ok {
		return newExitCodeError(1, errors.New("gormes chat -q: no final assistant content"))
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), content); err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes chat -q: write stdout: %w", err))
	}
	return nil
}

type forcedSkillRuntime struct {
	runtime *skills.Runtime
	names   []string
}

func newForcedSkillRuntime(cfg config.Config, names []string) *forcedSkillRuntime {
	if len(names) == 0 {
		return nil
	}
	return &forcedSkillRuntime{
		runtime: skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath()),
		names:   append([]string(nil), names...),
	}
}

func (r *forcedSkillRuntime) BuildSkillBlock(ctx context.Context, userMessage string) (string, []string, error) {
	if r == nil || r.runtime == nil || len(r.names) == 0 {
		return "", nil, nil
	}
	allowed := make(map[string]bool, len(r.names))
	for _, name := range r.names {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[strings.ToLower(name)] = true
		}
	}
	query := strings.TrimSpace(strings.Join(append([]string{userMessage}, r.names...), " "))
	block, names, _, err := r.runtime.BuildSkillBlockWithOptions(ctx, query, skills.RuntimeOptions{AllowedSkillNames: allowed})
	return block, names, err
}

func (r *forcedSkillRuntime) RecordSkillUsage(ctx context.Context, skillNames []string) error {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.RecordSkillUsage(ctx, skillNames)
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

func finalAssistantContent(history []llm.Message) (string, bool) {
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

type tuiProgram = tuiapp.Program

type tuiProgramFactory = tuiapp.ProgramFactory

func defaultTUIProgramFactory(model tea.Model, options ...tea.ProgramOption) tuiProgram {
	return tuiapp.DefaultProgramFactory(model, options...)
}

// welcomeStartupSeed returns the operator-facing release version and the
// agent tool count used to seed the session-aware welcome panel. internal/tui
// cannot import main.Version and the tool count is absent from
// kernel.RenderFrame, so cmd/gormes computes both here at startup.
func welcomeStartupSeed(reg *tools.Registry) (string, int, []string) {
	descs := gormescli.RegistryDescriptors(reg)
	return Version, len(descs), welcomeToolsets(descs)
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

func runResolvedTUIWithRuntime(cmd *cobra.Command, invocation tuiInvocation, runtime rootRuntime) error {
	return tuiapp.RunResolved(cmd, invocation, tuiAppRuntime(runtime))
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
	fmt.Fprintln(os.Stderr, gormescli.CrashStderrMessage(r, path))
}
