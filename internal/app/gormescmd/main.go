// Command gormes is the Go-native Hermes-compatible agent runtime.
package gormescmd

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	gonchoservice "github.com/TrebuchetDynamics/goncho/service"
	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/spf13/cobra"

	appgateway "github.com/TrebuchetDynamics/gormes-agent/internal/app/gateway"
	appgoncho "github.com/TrebuchetDynamics/gormes-agent/internal/app/goncho"
	navivoxapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
	apptelegram "github.com/TrebuchetDynamics/gormes-agent/internal/app/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
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
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
	gatewaymodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/profileapp"
	tuiapp "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/tuiapp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/security"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/runtime"
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

func Main() {
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
			if banner := gormescli.AdvisoryStartupBanner(config.GormesHome(), security.NoInstalledPackages, time.Now()); banner != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), banner)
			}
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
	installProfileShortcutCommands(root, runtime)
	gormescli.InstallRootRPCModeFlags(root)
	return root
}

func installProfileShortcutCommands(root *cobra.Command, runtime rootRuntime) {
	profiles, err := defaultListKnownProfiles()
	if err != nil {
		return
	}
	existing := map[string]struct{}{}
	for _, cmd := range root.Commands() {
		existing[cmd.Name()] = struct{}{}
		for _, alias := range cmd.Aliases {
			existing[alias] = struct{}{}
		}
	}
	baseHome := config.GormesBaseHome()
	for _, profile := range profiles {
		profile := strings.TrimSpace(profile)
		if profile == config.DefaultProfileID {
			continue
		}
		if profile == "" {
			continue
		}
		if _, conflict := existing[profile]; conflict {
			continue
		}
		if err := cli.ValidateProfileName(profile); err != nil {
			continue
		}
		cmd := &cobra.Command{
			Use:          profile,
			Short:        "Open the " + profile + " profile",
			Hidden:       true,
			Args:         cobra.NoArgs,
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, _ []string) error {
				if err := applyProfileRuntimeHome(baseHome, profile); err != nil {
					return err
				}
				invocation, err := resolveTUIInvocation(cmd)
				if err != nil {
					return err
				}
				return runtime.runResolvedTUI(cmd, invocation)
			},
		}
		root.AddCommand(cmd)
	}
}

func tuiAppRuntime(runtime rootRuntime) tuiapp.Runtime {
	return tuiapp.Runtime{
		ProgramFactory:       runtime.tuiProgramFactory,
		Version:              Version,
		VersionDateAlias:     VersionDateAlias,
		GitCommit:            resolveGitCommit(),
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
		"doctor": func() *cobra.Command {
			return gormescli.NewDoctorCommand(gormescli.DoctorCommandOptions{
				BuildProvenance: func() gormescli.BuildProvenance {
					build := newBuildProvenance()
					return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
				},
				NewExitCodeError:        newExitCodeError,
				BuildFirstRunPlan:       tuiapp.BuildFirstRunPlanFromConfig,
				FirstRunGuidanceCommand: tuiapp.FirstRunGuidanceCommand,
			})
		},
		"version": func() *cobra.Command { return gormescli.NewVersionCommand(versionInfo()) },
		"telegram": func() *cobra.Command {
			return channelsmodule.NewTelegramCommandWithSeams(channelsmodule.TelegramCommandSeams{
				Run: func(cmd *cobra.Command, args []string) error {
					return apptelegram.RunTelegram(cmd, args, apptelegram.RunOptions{
						GatewayTelegramDynamicCommands: appgateway.TelegramDynamicCommands,
						GatewayManagerConfig: func(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, allowedWhitelists map[string]gateway.WhitelistConfig, smap session.Map) gateway.ManagerConfig {
							return gatewayManagerConfig(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap, nil, nil, nil, gateway.RestartConfig{})
						},
						EnsureAgentTemplates: func(cfg config.Config, log *slog.Logger) error {
							_, err := gatewaymodule.EnsureAgentTemplates(cfg, log)
							return err
						},
					})
				},
			})
		},
		"gateway": func() *cobra.Command {
			return gatewaymodule.NewGatewayCommand(
				func(cmd *cobra.Command, args []string) error {
					return appgateway.RunGateway(cmd, args, appgateway.RunOptions{
						RegisterChannels:     registerConfiguredGatewayChannelsWithDefaults,
						NoWakeLock:           func(c *cobra.Command) bool { ok, _ := c.Flags().GetBool("no-wakelock"); return ok },
						NewExitCodeError:     newExitCodeError,
						GatewayManagerConfig: gatewayManagerConfig,
					})
				},
				gatewaymodule.NewGatewayCommandOptions(
					func() string { return newBuildProvenance().Version },
					func() string { return newBuildProvenance().GitCommit },
					newExitCodeError,
				),
			)
		},
		"channels": newChannelsCommand,
		"whatsapp": newWhatsAppCommand,
		"slack":    gormescli.NewSlackCommand,
		"session":  newSessionCommand,
		"memory":   newMemoryCommand,
		"goncho": func() *cobra.Command {
			return gormescli.NewGonchoCommand(gormescli.GonchoCommandOptions{
				BuildProvenance: func() appgoncho.BuildProvenance {
					build := newBuildProvenance()
					return appgoncho.BuildProvenance{
						Version:   build.Version,
						GitCommit: build.GitCommit,
					}
				},
			})
		},
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
		"login":     gormescli.NewDeprecatedLoginCommand,
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
		"webhook":    func() *cobra.Command { return gatewaymodule.NewWebhookCommand(gatewayOptions()) },
		"hooks":      func() *cobra.Command { return gatewaymodule.NewHooksCommand(gatewayOptions()) },
		"dump":       func() *cobra.Command { return gormescli.NewDumpCommand(hermesUnavailableOptions()) },
		"debug":      func() *cobra.Command { return gormescli.NewDebugCommand(hermesUnavailableOptions()) },
		"backup":     func() *cobra.Command { return gormescli.NewBackupCommand(hermesUnavailableOptions()) },
		"import":     func() *cobra.Command { return gormescli.NewImportCommand(hermesUnavailableOptions()) },
		"pairing":    func() *cobra.Command { return gatewaymodule.NewPairingCommand(gatewayOptions()) },
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
	_ = os.Unsetenv("GORMES_ACTIVE_PROFILE")
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
	if err := applyProfileRuntimeHome(baseHome, name); err != nil {
		return err
	}
	return nil
}

func applyProfileRuntimeHome(baseHome, name string) error {
	root, err := startupProfileRoot(baseHome, name)
	if err != nil {
		return newExitCodeError(2, err)
	}
	if err := os.Setenv("GORMES_HOME", root); err != nil {
		return newExitCodeError(2, fmt.Errorf("profile: set GORMES_HOME: %w", err))
	}
	if err := os.Setenv("GORMES_ACTIVE_PROFILE", name); err != nil {
		return newExitCodeError(2, fmt.Errorf("profile: set active profile: %w", err))
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
			if prompt == "" && len(args) == 1 && isKnownProfileShortcut(args[0]) {
				if err := applyProfileRuntimeHome(config.GormesBaseHome(), strings.TrimSpace(args[0])); err != nil {
					return err
				}
				invocation, err := resolveTUIInvocation(cmd)
				if err != nil {
					return err
				}
				return runtime.runResolvedTUI(cmd, invocation)
			}
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

func isKnownProfileShortcut(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if err := cli.ValidateProfileName(name); err != nil {
		return false
	}
	profiles, err := defaultListKnownProfiles()
	if err != nil {
		return false
	}
	for _, profile := range profiles {
		if profile == name {
			return true
		}
	}
	return false
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
	invocation, err := tuiapp.ResolveInvocation(cmd)
	if err != nil {
		return invocation, err
	}
	profiles, listErr := defaultListKnownProfiles()
	if listErr == nil {
		invocation.ProfileNames = profiles
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
		SystemPrompt:      llm.DefaultAgentIdentity,
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
	case strings.Contains(name, "browser_cdp"), strings.Contains(name, "browser-cdp"), strings.Contains(name, "browser_dialog"):
		return []string{"browser-cdp"}
	case strings.Contains(name, "browser"), strings.HasPrefix(name, "web_"):
		return []string{"browser"}
	case strings.Contains(name, "clarify"):
		return []string{"clarify"}
	case strings.Contains(name, "execute_code"):
		return []string{"code_execution"}
	case strings.Contains(name, "computer"):
		return []string{"computer_use"}
	case strings.Contains(name, "cron"):
		return []string{"cronjob"}
	case strings.Contains(name, "delegate"):
		return []string{"delegation"}
	case strings.Contains(name, "discord"):
		return []string{"discord"}
	case strings.Contains(name, "email"), strings.Contains(name, "himalaya"):
		return []string{"email"}
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

// =============================================================================
// Transitional functions moved from cmd/gormes/gateway.go (to be extracted to
// internal/app/gateway or internal/platform/cli/gormescli/modules/gateway in
// future passes).
// =============================================================================

// gatewayOptions returns a minimal gatewaymodule.Options for subcommands like
// webhook, hooks, and pairing that need one but don't need the full runtime.
func gatewayOptions() gatewaymodule.Options {
	return gatewaymodule.Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
		ExitError: newExitCodeError,
	}
}

func activateGatewaySecretRuntime(ctx context.Context, cfg config.Config, resolver runtime.SecretStringResolver) (config.Config, runtime.SecretRuntimeSnapshot, error) {
	activation, err := runtime.ActivateGatewaySecretRefs(ctx, cfg, runtime.GatewaySecretRuntimeOptions{Resolver: resolver})
	return activation.Config, activation.Snapshot, err
}

func newGatewayHermesClient(cfg config.Config) (llm.Client, error) {
	return gormescli.NewProviderHTTPClient(cfg, cfg.Hermes.Provider)
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
		KanbanSlashRunner:          gatewaymodule.NewKanbanSlashRunner(kanbanCommandOptions()),
		SkillsCommandOptions:       skillsCommandOptionsForConfig(cfg),
		RememberedSourceStore:      gateway.NewChannelDirectorySourceStore(config.GormesHome()),
		ContextFilesCWD:            gatewaymodule.ContextFilesCWD(cfg),
		LiveTurnNow:                func() time.Time { return time.Now() },
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
		MaxToolIterations: gormescli.ConfiguredMaxToolIterations(cfg),
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

// registerConfiguredGatewayChannelsWithDefaults wraps appgateway.RegisterConfiguredGatewayChannels
// with default channel factories for use as an injected dependency in RunOptions.
func registerConfiguredGatewayChannelsWithDefaults(mgr *gateway.Manager, cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, log *slog.Logger) (int, error) {
	return appgateway.RegisterConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, appgateway.DefaultChannelFactories(), nil, log)
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
	if err := gonchoservice.RunMigrations(db); err != nil {
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
	// Pin the pool to one connection so the per-connection busy_timeout above is
	// honored; otherwise database/sql may open extra connections with
	// busy_timeout=0 that fail concurrent writes immediately with SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	return db, nil
}
