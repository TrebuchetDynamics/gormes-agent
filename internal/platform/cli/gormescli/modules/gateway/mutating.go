package gateway

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// gatewayMutatingUnavailableExitCode is the stable non-zero exit code surfaced
// by non-Windows lifecycle subcommands that still do not own a native service
// manager path.
const gatewayMutatingUnavailableExitCode = 2

const gatewayDetachedEnvName = "GORMES_GATEWAY_DETACHED"

var gatewayMutatingUnavailableSubcommands = MutatingUnavailableSubcommands

func NewMutatingUnavailableCommand(name string, opts Options) *cobra.Command {
	return &cobra.Command{
		Use:          name,
		Short:        fmt.Sprintf("Manage gateway %s through the platform service helper", name),
		Long:         fmt.Sprintf("On Windows, the %s subcommand uses the native Scheduled Task gateway service. On Termux, run the foreground gateway in tmux with `gormes gateway`. On other platforms it remains unavailable; use the systemd/launchd helper exposed by internal/cli/service_restart.go to drive the live service manager.", name),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if gatewayRuntimeGOOS == "windows" {
				return runGatewayWindowsScheduledTaskCommand(cmd, name, opts)
			}
			if gatewayTermuxDetected(opts) {
				if opts.TermuxLifecycleGuidanceError != nil {
					return gatewayExitError(opts, gatewayMutatingUnavailableExitCode, opts.TermuxLifecycleGuidanceError(name))
				}
				return gatewayExitError(opts, gatewayMutatingUnavailableExitCode, fmt.Errorf("gateway: %s uses the Termux foreground/tmux lifecycle; run `gormes gateway` inside tmux; termux-wake-lock and Android battery settings are best-effort only, and Android may still stop background processes; use `gormes gateway status` and `gormes gateway stop` for runtime control", name))
			}
			return gatewayExitError(opts, gatewayMutatingUnavailableExitCode,
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

func runGatewayWindowsScheduledTaskCommand(cmd *cobra.Command, action string, opts Options) error {
	cfg := defaultGatewayWindowsScheduledTaskConfig()
	out := cmd.OutOrStdout()
	ctx := cmd.Context()

	switch action {
	case "install":
		if err := gatewayWindowsTaskRunner.Install(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("install", err, opts)
		}
		fmt.Fprintf(out, "gateway install: Scheduled Task service installed name=%q\n", cfg.TaskName)
		if err := gatewayWindowsTaskRunner.Start(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("install start", err, opts)
		}
		fmt.Fprintf(out, "gateway install: Scheduled Task service started name=%q\n", cfg.TaskName)
	case "start":
		if err := gatewayWindowsTaskRunner.Start(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("start", err, opts)
		}
		fmt.Fprintf(out, "gateway start: Scheduled Task service started name=%q\n", cfg.TaskName)
	case "restart":
		if err := gatewayWindowsTaskRunner.Restart(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("restart", err, opts)
		}
		fmt.Fprintf(out, "gateway restart: Scheduled Task service restarted name=%q\n", cfg.TaskName)
	case "uninstall":
		if err := gatewayWindowsTaskRunner.Uninstall(ctx, cfg); err != nil {
			return gatewayWindowsScheduledTaskError("uninstall", err, opts)
		}
		fmt.Fprintf(out, "gateway uninstall: Scheduled Task service removed name=%q\n", cfg.TaskName)
	default:
		return gatewayExitError(opts, gatewayMutatingUnavailableExitCode, fmt.Errorf("gateway: %s is not available; use the service_restart helper", action))
	}
	return nil
}

func gatewayWindowsScheduledTaskError(action string, err error, opts Options) error {
	return gatewayExitError(opts, gatewayMutatingUnavailableExitCode,
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
	escaped := strings.ReplaceAll(value, `"`, `\"`)
	return `"` + escaped + `"`
}
