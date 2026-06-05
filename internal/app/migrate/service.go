package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/migrationruntime"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type HermesDryRunReportJSON struct {
	Build BuildProvenance `json:"build"`
	*migrationruntime.MigrateHermesManifest
}

type HermesApplyReportJSON struct {
	Build BuildProvenance `json:"build"`
	migrationruntime.MigrateHermesWriteOutcome
}

type OpenClawDryRunReportJSON struct {
	Build BuildProvenance `json:"build"`
	*migrationruntime.MigrateOpenClawManifest
}

type OpenClawApplyReportJSON struct {
	Build BuildProvenance `json:"build"`
	migrationruntime.MigrateOpenClawApplyOutcome
}

type OpenClawCleanupReportJSON struct {
	Build BuildProvenance `json:"build"`
	migrationruntime.MigrateOpenClawCleanupOutcome
}

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e ExitError) Unwrap() error { return e.Err }

func ExitCode(err error) int {
	var exit ExitError
	if errors.As(err, &exit) {
		return exit.Code
	}
	return 0
}

func RunHermesDryRun(out io.Writer, source string, build BuildProvenance) error {
	m, err := migrationruntime.BuildMigrateHermesManifest(migrationruntime.MigrateHermesOptions{
		Source:            strings.TrimSpace(source),
		ExistingGormesEnv: CollectGormesEnvSnapshot(),
	})
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("gormes migrate hermes: %w", err)}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(HermesDryRunReportJSON{Build: build, MigrateHermesManifest: m}); err != nil {
		return fmt.Errorf("gormes migrate hermes: encode manifest: %w", err)
	}
	return nil
}

func RunHermesApply(out io.Writer, source, dest string, overwrite bool, build BuildProvenance) error {
	existingEnv := CollectGormesEnvSnapshot()
	manifest, err := migrationruntime.BuildMigrateHermesManifest(migrationruntime.MigrateHermesOptions{
		Source:            strings.TrimSpace(source),
		ExistingGormesEnv: existingEnv,
	})
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("gormes migrate hermes: %w", err)}
	}
	if manifest.Source.SelectedPath == "" {
		return ExitError{Code: 2, Err: fmt.Errorf("gormes migrate hermes: no Hermes source found; pass --source /path/to/hermes-home or set HERMES_HOME")}
	}
	sourcePath := manifest.Source.SelectedPath
	cfgBody, _ := os.ReadFile(filepath.Join(sourcePath, "config.yaml"))
	envBody, _ := os.ReadFile(filepath.Join(sourcePath, ".env"))

	destDir := strings.TrimSpace(dest)
	if destDir == "" {
		destDir = filepath.Dir(GormesConfigPath())
	}
	destEnv := filepath.Join(destDir, ".env")

	result, err := migrationruntime.ApplyMigrateHermesManifest(migrationruntime.MigrateHermesWriteRequest{
		Manifest:          *manifest,
		DestConfigDir:     destDir,
		DestEnvFile:       destEnv,
		ExistingGormesEnv: existingEnv,
		Overwrite:         overwrite,
		Yes:               true,
		SourceConfigBytes: map[string][]byte{"config.yaml": cfgBody, ".env": envBody},
	})
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("gormes migrate hermes: %w", err)}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(HermesApplyReportJSON{Build: build, MigrateHermesWriteOutcome: result}); err != nil {
		return fmt.Errorf("gormes migrate hermes: encode outcome: %w", err)
	}
	return nil
}

func RunOpenClawDryRun(out io.Writer, source string, build BuildProvenance) error {
	m, err := migrationruntime.BuildMigrateOpenClawManifest(migrationruntime.MigrateOpenClawOptions{
		Source:            strings.TrimSpace(source),
		ExistingGormesEnv: CollectMigrationEnvSnapshot(),
	})
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("gormes migrate openclaw: %w", err)}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(OpenClawDryRunReportJSON{Build: build, MigrateOpenClawManifest: m}); err != nil {
		return fmt.Errorf("gormes migrate openclaw: encode manifest: %w", err)
	}
	return nil
}

func RunOpenClawApply(out io.Writer, source, dest string, overwrite, secrets bool, build BuildProvenance) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return ExitError{Code: 2, Err: errors.New("gormes migrate openclaw --yes: --source is required")}
	}
	existingEnv := CollectMigrationEnvSnapshot()
	manifest, err := migrationruntime.BuildMigrateOpenClawManifest(migrationruntime.MigrateOpenClawOptions{
		Source:            source,
		ExistingGormesEnv: existingEnv,
	})
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("gormes migrate openclaw: %w", err)}
	}
	cfgBody, _ := os.ReadFile(filepath.Join(source, "config.yaml"))
	envBody, _ := os.ReadFile(filepath.Join(source, ".env"))

	destDir := strings.TrimSpace(dest)
	if destDir == "" {
		destDir = filepath.Dir(GormesConfigPath())
	}
	destEnv := filepath.Join(destDir, ".env")

	result, err := migrationruntime.ApplyMigrateOpenClawManifest(migrationruntime.MigrateOpenClawApplyRequest{
		Manifest:          *manifest,
		DestConfigDir:     destDir,
		DestEnvFile:       destEnv,
		DestSkillsDir:     GormesSkillsDir(),
		DestMemoryDir:     GormesMemoryDir(),
		ReportRootDir:     GormesMigrationReportRoot(),
		ExistingGormesEnv: existingEnv,
		SourceConfigBytes: map[string][]byte{"config.yaml": cfgBody, ".env": envBody},
		SourceRoot:        source,
		Overwrite:         overwrite,
		Yes:               true,
		SecretsEnabled:    secrets,
	})
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("gormes migrate openclaw: %w", err)}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(OpenClawApplyReportJSON{Build: build, MigrateOpenClawApplyOutcome: result}); err != nil {
		return fmt.Errorf("gormes migrate openclaw: encode outcome: %w", err)
	}
	return nil
}

func RunOpenClawCleanup(out io.Writer, commandLabel string, dryRun bool, build BuildProvenance) error {
	home, _ := os.UserHomeDir()
	result, err := migrationruntime.PerformMigrateOpenClawCleanup(migrationruntime.MigrateOpenClawCleanupRequest{HomeDir: home, DryRun: dryRun})
	if err != nil {
		return ExitError{Code: 2, Err: fmt.Errorf("%s: %w", commandLabel, err)}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(OpenClawCleanupReportJSON{Build: build, MigrateOpenClawCleanupOutcome: result}); err != nil {
		return fmt.Errorf("%s: encode outcome: %w", commandLabel, err)
	}
	return nil
}

func GormesConfigPath() string { return config.ConfigPath() }

func GormesSkillsDir() string { return filepath.Join(config.GormesHome(), "skills") }

func GormesMemoryDir() string { return filepath.Join(config.GormesHome(), "memory") }

func GormesMigrationReportRoot() string {
	return filepath.Join(config.GormesHome(), "migrations", "openclaw")
}

func CollectGormesEnvSnapshot() map[string]string {
	out := make(map[string]string)
	for _, kv := range os.Environ() {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		k := kv[:eq]
		if !strings.HasPrefix(k, "GORMES_") {
			continue
		}
		out[k] = kv[eq+1:]
	}
	return out
}

func CollectMigrationEnvSnapshot() map[string]string {
	out := CollectGormesEnvSnapshot()
	for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "OPENROUTER_API_KEY"} {
		if value, ok := os.LookupEnv(key); ok {
			out[key] = value
		}
	}
	return out
}
