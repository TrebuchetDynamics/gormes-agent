package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	EnvironmentBackendSingularity = "singularity"

	EnvironmentSingularityExecutableResolved = "singularity_executable_resolved"
	EnvironmentSingularityUnavailable        = "singularity_unavailable"
	EnvironmentSingularityVersionChecked     = "singularity_version_checked"
	EnvironmentSingularityVersionFailed      = "singularity_version_failed"
	EnvironmentSingularityVersionTimeout     = "singularity_version_timeout"
	EnvironmentSingularityStartPlanned       = "singularity_instance_start_planned"
	EnvironmentSingularityExecPlanned        = "singularity_exec_planned"
	EnvironmentSingularityCleanupPlanned     = "singularity_cleanup_planned"
)

const (
	SingularityVersionCheckTimeout  = 10 * time.Second
	SingularityInstanceStartTimeout = 120 * time.Second
	SingularityInstanceStopTimeout  = 30 * time.Second
)

type SingularityExecutableLookup func(string) string

type SingularityCommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
	Err      error
}

type SingularityCommandRunner interface {
	RunSingularityCommand(ctx context.Context, argv []string, timeout time.Duration) SingularityCommandResult
}

type SingularityPreflight struct {
	Executable string
	Evidence   []EnvironmentEvidence
}

type SingularityBind struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

type SingularityInstanceConfig struct {
	Executable           string
	Image                string
	InstanceID           string
	PersistentFilesystem bool
	OverlayPath          string
	Binds                []SingularityBind
	MemoryMB             int
	CPUs                 float64
}

type SingularityExecConfig struct {
	Executable string
	InstanceID string
	Command    string
	Login      bool
	Timeout    time.Duration
}

type SingularityCommandSpec struct {
	Argv     []string
	Timeout  time.Duration
	Evidence EnvironmentEvidence
}

type localSingularityCommandRunner struct{}

func ResolveSingularityExecutable(lookup SingularityExecutableLookup) (string, EnvironmentEvidence) {
	if lookup == nil {
		lookup = func(name string) string {
			path, err := exec.LookPath(name)
			if err != nil {
				return ""
			}
			return path
		}
	}

	for _, candidate := range []string{"apptainer", "singularity"} {
		if path := strings.TrimSpace(lookup(candidate)); path != "" {
			return path, EnvironmentEvidence{
				Code:      EnvironmentSingularityExecutableResolved,
				Status:    EnvironmentStatusRecorded,
				Backend:   EnvironmentBackendSingularity,
				Operation: "preflight",
				Resource:  path,
				Message:   fmt.Sprintf("resolved %s executable", candidate),
			}
		}
	}

	return "", EnvironmentEvidence{
		Code:      EnvironmentSingularityUnavailable,
		Status:    EnvironmentStatusUnavailable,
		Backend:   EnvironmentBackendSingularity,
		Operation: "preflight",
		Message:   "Install Apptainer or Singularity to use the Singularity sandbox backend.",
	}
}

func CheckSingularityAvailable(ctx context.Context, lookup SingularityExecutableLookup, runner SingularityCommandRunner) (SingularityPreflight, error) {
	executable, resolved := ResolveSingularityExecutable(lookup)
	preflight := SingularityPreflight{
		Executable: executable,
		Evidence:   []EnvironmentEvidence{resolved},
	}
	if executable == "" {
		return preflight, &EnvironmentEvidenceError{Evidence: resolved}
	}

	if runner == nil {
		runner = localSingularityCommandRunner{}
	}
	result := runner.RunSingularityCommand(ctx, []string{executable, "version"}, SingularityVersionCheckTimeout)
	if result.TimedOut {
		evidence := singularityUnavailableEvidence(
			EnvironmentSingularityVersionTimeout,
			"preflight",
			executable+" version",
			fmt.Sprintf("%s version timed out after %s", executable, SingularityVersionCheckTimeout),
		)
		preflight.Evidence = append(preflight.Evidence, evidence)
		return preflight, &EnvironmentEvidenceError{Evidence: evidence}
	}
	if result.ExitCode > 0 {
		detail := firstNonEmptySingularityText(result.Stderr, result.Stdout)
		evidence := singularityUnavailableEvidence(
			EnvironmentSingularityVersionFailed,
			"preflight",
			executable+" version",
			fmt.Sprintf("%s version exit %d: %s", executable, result.ExitCode, boundedSingularityText(detail)),
		)
		preflight.Evidence = append(preflight.Evidence, evidence)
		return preflight, &EnvironmentEvidenceError{Evidence: evidence}
	}
	if result.Err != nil {
		evidence := singularityUnavailableEvidence(
			EnvironmentSingularityVersionFailed,
			"preflight",
			executable+" version",
			fmt.Sprintf("%s version failed: %s", executable, boundedSingularityText(result.Err.Error())),
		)
		preflight.Evidence = append(preflight.Evidence, evidence)
		return preflight, &EnvironmentEvidenceError{Evidence: evidence}
	}
	if result.ExitCode != 0 {
		evidence := singularityUnavailableEvidence(
			EnvironmentSingularityVersionFailed,
			"preflight",
			executable+" version",
			fmt.Sprintf("%s version exit %d", executable, result.ExitCode),
		)
		preflight.Evidence = append(preflight.Evidence, evidence)
		return preflight, &EnvironmentEvidenceError{Evidence: evidence}
	}

	preflight.Evidence = append(preflight.Evidence, EnvironmentEvidence{
		Code:      EnvironmentSingularityVersionChecked,
		Status:    EnvironmentStatusRecorded,
		Backend:   EnvironmentBackendSingularity,
		Operation: "preflight",
		Resource:  executable + " version",
		Message:   boundedSingularityText(firstNonEmptySingularityText(result.Stdout, "version check passed")),
	})
	return preflight, nil
}

func (localSingularityCommandRunner) RunSingularityCommand(ctx context.Context, argv []string, timeout time.Duration) SingularityCommandResult {
	if len(argv) == 0 {
		return SingularityCommandResult{ExitCode: -1, Err: errors.New("singularity command argv is empty")}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return SingularityCommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: -1,
			TimedOut: true,
			Err:      ctx.Err(),
		}
	}
	if err != nil {
		exitCode := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return SingularityCommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: exitCode,
			Err:      err,
		}
	}
	return SingularityCommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}
}

func BuildSingularityStartCommand(config SingularityInstanceConfig) (SingularityCommandSpec, error) {
	executable := defaultSingularityExecutable(config.Executable)
	image := strings.TrimSpace(config.Image)
	if image == "" {
		return SingularityCommandSpec{}, errors.New("singularity start: image is required")
	}
	instanceID := strings.TrimSpace(config.InstanceID)
	if instanceID == "" {
		return SingularityCommandSpec{}, errors.New("singularity start: instance id is required")
	}

	argv := []string{
		executable,
		"instance",
		"start",
		"--containall",
		"--no-home",
	}
	if config.PersistentFilesystem {
		overlay := strings.TrimSpace(config.OverlayPath)
		if overlay == "" {
			return SingularityCommandSpec{}, errors.New("singularity start: overlay path is required for persistent filesystem")
		}
		argv = append(argv, "--overlay", overlay)
	} else {
		argv = append(argv, "--writable-tmpfs")
	}

	for _, bind := range config.Binds {
		arg, err := bind.singularityArg()
		if err != nil {
			return SingularityCommandSpec{}, err
		}
		argv = append(argv, "--bind", arg)
	}
	if config.MemoryMB > 0 {
		argv = append(argv, "--memory", fmt.Sprintf("%dM", config.MemoryMB))
	}
	if config.CPUs > 0 {
		argv = append(argv, "--cpus", strconv.FormatFloat(config.CPUs, 'f', -1, 64))
	}
	argv = append(argv, image, instanceID)

	return SingularityCommandSpec{
		Argv:    argv,
		Timeout: SingularityInstanceStartTimeout,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentSingularityStartPlanned,
			Status:    EnvironmentStatusRecorded,
			Backend:   EnvironmentBackendSingularity,
			Operation: "start",
			Resource:  instanceID,
			Message:   "planned Singularity instance start command",
		},
	}, nil
}

func BuildSingularityExecCommand(config SingularityExecConfig) (SingularityCommandSpec, error) {
	executable := defaultSingularityExecutable(config.Executable)
	instanceID := strings.TrimSpace(config.InstanceID)
	if instanceID == "" {
		return SingularityCommandSpec{}, errors.New("singularity exec: instance id is required")
	}
	command := strings.TrimSpace(config.Command)
	if command == "" {
		return SingularityCommandSpec{}, errors.New("singularity exec: command is required")
	}

	argv := []string{
		executable,
		"exec",
		"instance://" + instanceID,
		"bash",
	}
	if config.Login {
		argv = append(argv, "-l")
	}
	argv = append(argv, "-c", command)

	return SingularityCommandSpec{
		Argv:    argv,
		Timeout: config.Timeout,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentSingularityExecPlanned,
			Status:    EnvironmentStatusRecorded,
			Backend:   EnvironmentBackendSingularity,
			Operation: "execute",
			Resource:  instanceID,
			Message:   "planned Singularity exec command",
		},
	}, nil
}

func BuildSingularityStopCommand(executable, instanceID string) (SingularityCommandSpec, error) {
	executable = defaultSingularityExecutable(executable)
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return SingularityCommandSpec{}, errors.New("singularity cleanup: instance id is required")
	}
	return SingularityCommandSpec{
		Argv:    []string{executable, "instance", "stop", instanceID},
		Timeout: SingularityInstanceStopTimeout,
		Evidence: EnvironmentEvidence{
			Code:      EnvironmentSingularityCleanupPlanned,
			Status:    EnvironmentStatusRecorded,
			Backend:   EnvironmentBackendSingularity,
			Operation: "cleanup",
			Resource:  instanceID,
			Message:   "planned Singularity instance stop command",
		},
	}, nil
}

func (b SingularityBind) singularityArg() (string, error) {
	host := strings.TrimSpace(b.HostPath)
	if host == "" {
		return "", errors.New("singularity bind: host path is required")
	}
	container := strings.TrimSpace(b.ContainerPath)
	if container == "" {
		return "", errors.New("singularity bind: container path is required")
	}
	arg := host + ":" + container
	if b.ReadOnly {
		arg += ":ro"
	}
	return arg, nil
}

func defaultSingularityExecutable(executable string) string {
	if executable = strings.TrimSpace(executable); executable != "" {
		return executable
	}
	return "apptainer"
}

func singularityUnavailableEvidence(code, operation, resource, message string) EnvironmentEvidence {
	return EnvironmentEvidence{
		Code:      code,
		Status:    EnvironmentStatusUnavailable,
		Backend:   EnvironmentBackendSingularity,
		Operation: operation,
		Resource:  resource,
		Message:   message,
	}
}

func firstNonEmptySingularityText(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func boundedSingularityText(value string) string {
	value = strings.TrimSpace(value)
	const max = 240
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
