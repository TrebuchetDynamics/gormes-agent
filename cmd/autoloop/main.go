package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/autoloop"
)

var commandStdout io.Writer = os.Stdout
var serviceRunner autoloop.Runner = autoloop.ExecRunner{}

const usage = "usage: autoloop run [--dry-run] | audit | digest | service install | service install-audit | service disable legacy-timers"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	switch {
	case len(args) == 1 && args[0] == "run":
		cfg, err := autoloop.ConfigFromEnv(root, autoloopEnv())
		if err != nil {
			return err
		}
		return runAutoloop(cfg, false)
	case len(args) == 2 && args[0] == "run" && args[1] == "--dry-run":
		cfg, err := autoloop.ConfigFromEnv(root, autoloopEnv())
		if err != nil {
			return err
		}
		return runAutoloop(cfg, true)
	case len(args) == 1 && args[0] == "digest":
		digest, err := autoloop.DigestLedger(filepath.Join(digestRunRoot(root), "state", "runs.jsonl"))
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(commandStdout, digest)
		return err
	case len(args) == 1 && args[0] == "audit":
		digest, err := autoloop.DigestLedger(filepath.Join(digestRunRoot(root), "state", "runs.jsonl"))
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(commandStdout, digest)
		return err
	case len(args) >= 2 && args[0] == "service" && args[1] == "install":
		force, err := serviceForce(args[2:])
		if err != nil {
			return err
		}
		return installService(root, "gormes-orchestrator.service", force)
	case len(args) >= 2 && args[0] == "service" && args[1] == "install-audit":
		force, err := serviceForce(args[2:])
		if err != nil {
			return err
		}
		return installService(root, "gormes-orchestrator-audit.service", force)
	case len(args) == 3 && args[0] == "service" && args[1] == "disable" && args[2] == "legacy-timers":
		return autoloop.DisableLegacyTimers(context.Background(), serviceRunner)
	default:
		return fmt.Errorf(usage)
	}
}

func serviceForce(args []string) (bool, error) {
	force := os.Getenv("FORCE") == "1"
	for _, arg := range args {
		if arg != "--force" {
			return false, fmt.Errorf(usage)
		}
		force = true
	}

	return force, nil
}

func installService(root string, unitName string, force bool) error {
	unitDir, err := systemdUserUnitDir()
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	return autoloop.InstallService(context.Background(), autoloop.ServiceInstallOptions{
		Runner:       serviceRunner,
		UnitDir:      unitDir,
		UnitName:     unitName,
		AutoloopPath: executable,
		WorkDir:      root,
		AutoStart:    true,
		Force:        force,
	})
}

func systemdUserUnitDir() (string, error) {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "systemd", "user"), nil
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".config", "systemd", "user"), nil
	}

	return "", fmt.Errorf("cannot determine systemd user unit directory: set XDG_CONFIG_HOME or HOME")
}

func digestRunRoot(root string) string {
	if runRoot := os.Getenv("RUN_ROOT"); runRoot != "" {
		return runRoot
	}

	return filepath.Join(root, ".codex", "orchestrator")
}

func runAutoloop(cfg autoloop.Config, dryRun bool) error {
	summary, err := autoloop.RunOnce(context.Background(), autoloop.RunOptions{
		Config: cfg,
		DryRun: dryRun,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(commandStdout, "candidates: %d\nselected: %d\n", summary.Candidates, len(summary.Selected))
	for _, candidate := range summary.Selected {
		fmt.Fprintf(commandStdout, "- %s/%s %s [%s]\n", candidate.PhaseID, candidate.SubphaseID, candidate.ItemName, candidate.Status)
	}

	return nil
}

func autoloopEnv() map[string]string {
	env := map[string]string{}
	for _, key := range []string{"PROGRESS_JSON", "RUN_ROOT", "BACKEND", "MODE", "MAX_AGENTS"} {
		env[key] = os.Getenv(key)
	}

	return env
}
