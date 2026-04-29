package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// editorRunner abstracts the two operations `gormes config edit` performs
// against the host: locating an editor binary on PATH and spawning it
// against the resolved config path. Production wiring uses os/exec; tests
// pass a stub so no real shell command is invoked.
type editorRunner interface {
	LookPath(name string) (string, bool)
	Run(editor, path string) error
}

type osEditorRunner struct{}

func (osEditorRunner) LookPath(name string) (string, bool) {
	resolved, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return resolved, true
}

func (osEditorRunner) Run(editor, path string) error {
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// configEditorRunner is the package-level injection seam for the editor
// lookup + dispatch used by `gormes config edit`. Production wiring is the
// os/exec-backed default; tests overwrite this var (with t.Cleanup
// restoration) so no real binary is ever spawned. Tests must restore the
// previous value or use the helper withConfigEditorRunner below.
var configEditorRunner editorRunner = osEditorRunner{}

// withConfigEditorRunner swaps the package-level editor runner for the
// duration of a test and restores the previous value when the test ends.
func withConfigEditorRunner(t interface{ Cleanup(func()) }, runner editorRunner) {
	prev := configEditorRunner
	configEditorRunner = runner
	t.Cleanup(func() { configEditorRunner = prev })
}

// newConfigCommand builds the `gormes config` subtree. It exposes the
// Hermes-aliased command surface — show, path, env-path, set, edit, check,
// migrate — over the native GormesHome TOML/dotenv files. Writes route
// through internal/config helpers so secrets never land in config.toml.
// `config migrate` applies only Gormes-native schema migrations; importing
// upstream Hermes/OpenClaw state is owned by the separate `gormes migrate`
// command tree.
func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "config",
		Short:        "Inspect or update the Gormes configuration files",
		SilenceUsage: true,
	}
	cmd.AddCommand(newConfigPathCommand())
	cmd.AddCommand(newConfigEnvPathCommand())
	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigSetCommand())
	cmd.AddCommand(newConfigEditCommand())
	cmd.AddCommand(newConfigCheckCommand())
	cmd.AddCommand(newConfigMigrateCommand())
	return cmd
}

func newConfigPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the Gormes TOML config path",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.ConfigPath())
			return nil
		},
	}
}

func newConfigEnvPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "env-path",
		Short: "Print the Gormes dotenv (.env) secrets path",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.EnvPath())
			return nil
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value (TOML for non-secret keys, .env for *_API_KEY/*_TOKEN/api_key)",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("usage: gormes config set <key> <value>")
			}
			if len(args) > 2 {
				return errors.New("gormes config set takes exactly two arguments: <key> <value>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.TrimSpace(args[0])
			value := args[1]
			if key == "" {
				return errors.New("gormes config set: empty key")
			}
			if config.IsSecretKey(key) {
				envName := config.SecretEnvName(key)
				if err := config.WriteEnvValue(config.EnvPath(), envName, value); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "set %s in %s\n", envName, config.EnvPath())
				return nil
			}
			if err := config.WriteTOMLValue(config.ConfigPath(), key, value); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set %s in %s\n", key, config.ConfigPath())
			return nil
		},
	}
	return cmd
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show resolved Gormes configuration with secrets redacted",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Paths")
			fmt.Fprintf(out, "  config: %s\n", config.ConfigPath())
			fmt.Fprintf(out, "  env:    %s\n", config.EnvPath())
			fmt.Fprintln(out, "Hermes")
			fmt.Fprintf(out, "  endpoint: %s\n", cfg.Hermes.Endpoint)
			fmt.Fprintf(out, "  model:    %s\n", cfg.Hermes.Model)
			fmt.Fprintf(out, "  provider: %s\n", cfg.Hermes.Provider)
			fmt.Fprintln(out, "Secrets")
			fmt.Fprintf(out, "  api_key: %s\n", redactedSecretStatus(cfg.Hermes.APIKey))
			fmt.Fprintf(out, "  GORMES_API_KEY (env): %s\n", redactedSecretStatus(os.Getenv("GORMES_API_KEY")))
			return nil
		},
	}
}

func redactedSecretStatus(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(not set)"
	}
	return "set [REDACTED]"
}

// commonEditorCandidates lists the fallback editors `gormes config edit`
// probes when EDITOR/VISUAL are both unset, in priority order. Mirrors
// hermes_cli/config.py edit_config().
var commonEditorCandidates = []string{"nano", "vim", "vi", "code", "notepad"}

func newConfigEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the Gormes config.toml in $EDITOR / $VISUAL or a discovered fallback",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := config.ConfigPath()
			if err := config.EnsureConfigFile(path); err != nil {
				return err
			}
			runner := configEditorRunner
			if runner == nil {
				runner = osEditorRunner{}
			}
			chosen := pickEditor(runner)
			out := cmd.OutOrStdout()
			if chosen == "" {
				fmt.Fprintln(out, "No editor found. Set $EDITOR or $VISUAL, or open the file directly:")
				fmt.Fprintf(out, "  %s\n", path)
				return nil
			}
			fmt.Fprintf(out, "Opening %s in %s...\n", path, chosen)
			return runner.Run(chosen, path)
		},
	}
}

// pickEditor mirrors edit_config(): EDITOR > VISUAL > common-editor scan.
// Empty / unset env vars fall through to the lookup. The injected runner
// is the only path through which a binary name is checked or spawned.
func pickEditor(runner editorRunner) string {
	for _, env := range []string{"EDITOR", "VISUAL"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v
		}
	}
	for _, candidate := range commonEditorCandidates {
		if _, ok := runner.LookPath(candidate); ok {
			return candidate
		}
	}
	return ""
}

func newConfigCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate config.toml schema without writing — reports _config_version, missing/empty fields, and dotenv presence",
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := config.Check()
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Paths")
			fmt.Fprintf(out, "  config: %s\n", report.ConfigPath)
			fmt.Fprintf(out, "  env:    %s\n", report.EnvPath)
			fmt.Fprintf(out, "_config_version: %d (latest known: %d)\n", report.ConfigVersion, report.LatestVersion)
			fmt.Fprintf(out, "dotenv present: %t\n", report.DotenvPresent)
			if err != nil {
				fmt.Fprintf(out, "error: %s\n", err.Error())
				return errors.New("config check failed")
			}
			if len(report.Issues) == 0 {
				fmt.Fprintln(out, "OK")
				return nil
			}
			var sawError bool
			for _, issue := range report.Issues {
				fmt.Fprintf(out, "  [%s] %s: %s\n", issue.Severity, issue.Field, issue.Message)
				if issue.Severity == "error" {
					sawError = true
				}
			}
			if sawError {
				return errors.New("config check found errors")
			}
			return nil
		},
	}
}

func newConfigMigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply native Gormes config.toml schema/default migrations",
		Long: `Apply native Gormes config.toml schema/default migrations.

This command updates only the native Gormes config schema (TOML _config_version
and any default fill-ins). It is a no-op when the file is already at the
current version. Atomic writes guarantee the file is never left half-written.

Importing upstream Hermes or OpenClaw state is a separate concern; use
` + "`gormes migrate hermes`" + ` or ` + "`gormes migrate openclaw`" + ` for those cross-product paths.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := config.MigrateConfigFile(config.ConfigPath())
			out := cmd.OutOrStdout()
			if err != nil {
				return err
			}
			if result.NoOp {
				fmt.Fprintf(out, "no-op: %s already at _config_version=%d\n", result.Path, result.ToVersion)
				return nil
			}
			fmt.Fprintf(out, "migrated %s: _config_version %d -> %d (wrote=%t)\n",
				result.Path, result.FromVersion, result.ToVersion, result.Wrote)
			return nil
		},
	}
}
