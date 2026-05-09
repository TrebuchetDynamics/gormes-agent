package tools

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type singularityRunnerCall struct {
	argv    []string
	timeout time.Duration
}

type fakeSingularityRunner struct {
	results []SingularityCommandResult
	calls   []singularityRunnerCall
}

func (f *fakeSingularityRunner) RunSingularityCommand(ctx context.Context, argv []string, timeout time.Duration) SingularityCommandResult {
	copied := append([]string(nil), argv...)
	f.calls = append(f.calls, singularityRunnerCall{argv: copied, timeout: timeout})
	if len(f.results) == 0 {
		return SingularityCommandResult{ExitCode: 0}
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result
}

func TestSingularityExecutableResolutionPrefersApptainer(t *testing.T) {
	got, evidence := ResolveSingularityExecutable(fakeSingularityLookup(map[string]string{
		"apptainer":   "/usr/bin/apptainer",
		"singularity": "/usr/bin/singularity",
	}))

	if got != "/usr/bin/apptainer" {
		t.Fatalf("ResolveSingularityExecutable() = %q, want apptainer", got)
	}
	if evidence.Code != EnvironmentSingularityExecutableResolved ||
		evidence.Status != EnvironmentStatusRecorded ||
		evidence.Backend != EnvironmentBackendSingularity ||
		evidence.Resource != "/usr/bin/apptainer" {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
}

func TestSingularityExecutableResolutionFallsBackToSingularity(t *testing.T) {
	got, evidence := ResolveSingularityExecutable(fakeSingularityLookup(map[string]string{
		"singularity": "/usr/local/bin/singularity",
	}))

	if got != "/usr/local/bin/singularity" {
		t.Fatalf("ResolveSingularityExecutable() = %q, want singularity", got)
	}
	if !strings.Contains(evidence.Message, "singularity") {
		t.Fatalf("evidence should name fallback executable: %+v", evidence)
	}
}

func TestSingularityExecutableResolutionUnavailableEvidence(t *testing.T) {
	got, evidence := ResolveSingularityExecutable(fakeSingularityLookup(nil))

	if got != "" {
		t.Fatalf("ResolveSingularityExecutable() = %q, want empty", got)
	}
	if evidence.Code != EnvironmentSingularityUnavailable ||
		evidence.Status != EnvironmentStatusUnavailable ||
		evidence.Backend != EnvironmentBackendSingularity {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
	if !strings.Contains(evidence.Message, "Install Apptainer or Singularity") {
		t.Fatalf("unavailable message should include install guidance: %q", evidence.Message)
	}
}

func TestSingularityPreflightRunsVersionWithTimeout(t *testing.T) {
	runner := &fakeSingularityRunner{
		results: []SingularityCommandResult{{ExitCode: 0, Stdout: "apptainer version 1.3.6"}},
	}
	preflight, err := CheckSingularityAvailable(context.Background(), fakeSingularityLookup(map[string]string{
		"apptainer": "/usr/bin/apptainer",
	}), runner)
	if err != nil {
		t.Fatalf("CheckSingularityAvailable returned error: %v", err)
	}
	if preflight.Executable != "/usr/bin/apptainer" {
		t.Fatalf("preflight executable = %q, want apptainer", preflight.Executable)
	}
	wantArgv(t, runner.calls[0].argv, []string{"/usr/bin/apptainer", "version"})
	if runner.calls[0].timeout != SingularityVersionCheckTimeout {
		t.Fatalf("version timeout = %s, want %s", runner.calls[0].timeout, SingularityVersionCheckTimeout)
	}
	if got := lastSingularityEvidence(preflight.Evidence); got.Code != EnvironmentSingularityVersionChecked {
		t.Fatalf("last evidence = %+v, want version checked", got)
	}
}

func TestSingularityPreflightReportsVersionFailures(t *testing.T) {
	tests := []struct {
		name     string
		result   SingularityCommandResult
		wantCode string
		wantText string
	}{
		{
			name:     "nonzero exit",
			result:   SingularityCommandResult{ExitCode: 2, Stderr: "permission denied"},
			wantCode: EnvironmentSingularityVersionFailed,
			wantText: "exit 2",
		},
		{
			name:     "runner error",
			result:   SingularityCommandResult{Err: errors.New("spawn failed")},
			wantCode: EnvironmentSingularityVersionFailed,
			wantText: "spawn failed",
		},
		{
			name:     "timeout",
			result:   SingularityCommandResult{TimedOut: true},
			wantCode: EnvironmentSingularityVersionTimeout,
			wantText: "timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CheckSingularityAvailable(context.Background(), fakeSingularityLookup(map[string]string{
				"apptainer": "/usr/bin/apptainer",
			}), &fakeSingularityRunner{results: []SingularityCommandResult{tt.result}})
			if err == nil {
				t.Fatal("CheckSingularityAvailable returned nil error")
			}
			evidence, ok := EnvironmentEvidenceFromError(err)
			if !ok {
				t.Fatalf("error did not carry EnvironmentEvidence: %T %v", err, err)
			}
			if evidence.Code != tt.wantCode || evidence.Status != EnvironmentStatusUnavailable {
				t.Fatalf("unexpected evidence: %+v", evidence)
			}
			if !strings.Contains(evidence.Message, tt.wantText) {
				t.Fatalf("message %q does not contain %q", evidence.Message, tt.wantText)
			}
		})
	}
}

func TestSingularityStartCommandBuildsHardenedInstance(t *testing.T) {
	spec, err := BuildSingularityStartCommand(SingularityInstanceConfig{
		Executable: "/usr/bin/apptainer",
		Image:      "docker://ubuntu:22.04",
		InstanceID: "gormes-test",
		Binds: []SingularityBind{
			{HostPath: "/home/user/.config/gormes", ContainerPath: "/root/.config/gormes", ReadOnly: true},
			{HostPath: "/home/user/.gormes/skills", ContainerPath: "/opt/gormes/skills", ReadOnly: true},
		},
		MemoryMB: 2048,
		CPUs:     2.5,
	})
	if err != nil {
		t.Fatalf("BuildSingularityStartCommand returned error: %v", err)
	}

	wantArgv(t, spec.Argv, []string{
		"/usr/bin/apptainer", "instance", "start",
		"--containall", "--no-home", "--writable-tmpfs",
		"--bind", "/home/user/.config/gormes:/root/.config/gormes:ro",
		"--bind", "/home/user/.gormes/skills:/opt/gormes/skills:ro",
		"--memory", "2048M", "--cpus", "2.5",
		"docker://ubuntu:22.04", "gormes-test",
	})
	if spec.Timeout != SingularityInstanceStartTimeout {
		t.Fatalf("start timeout = %s, want %s", spec.Timeout, SingularityInstanceStartTimeout)
	}
	if spec.Evidence.Code != EnvironmentSingularityStartPlanned {
		t.Fatalf("start evidence = %+v, want planned", spec.Evidence)
	}
}

func TestSingularityStartCommandUsesOverlayWhenPersistent(t *testing.T) {
	spec, err := BuildSingularityStartCommand(SingularityInstanceConfig{
		Executable:           "apptainer",
		Image:                "/var/cache/gormes/sandbox.sif",
		InstanceID:           "gormes-test",
		PersistentFilesystem: true,
		OverlayPath:          "/var/lib/gormes/overlay.img",
	})
	if err != nil {
		t.Fatalf("BuildSingularityStartCommand returned error: %v", err)
	}
	if !containsArgPair(spec.Argv, "--overlay", "/var/lib/gormes/overlay.img") {
		t.Fatalf("start argv should include overlay: %v", spec.Argv)
	}
	if containsArg(spec.Argv, "--writable-tmpfs") {
		t.Fatalf("persistent start argv should not include writable tmpfs: %v", spec.Argv)
	}
}

func TestSingularityExecAndCleanupCommands(t *testing.T) {
	execSpec, err := BuildSingularityExecCommand(SingularityExecConfig{
		Executable: "apptainer",
		InstanceID: "gormes-test",
		Command:    "pwd",
	})
	if err != nil {
		t.Fatalf("BuildSingularityExecCommand returned error: %v", err)
	}
	wantArgv(t, execSpec.Argv, []string{"apptainer", "exec", "instance://gormes-test", "bash", "-c", "pwd"})
	if execSpec.Evidence.Code != EnvironmentSingularityExecPlanned {
		t.Fatalf("exec evidence = %+v, want planned", execSpec.Evidence)
	}

	loginSpec, err := BuildSingularityExecCommand(SingularityExecConfig{
		Executable: "apptainer",
		InstanceID: "gormes-test",
		Command:    "echo $SHELL",
		Login:      true,
	})
	if err != nil {
		t.Fatalf("BuildSingularityExecCommand login returned error: %v", err)
	}
	wantArgv(t, loginSpec.Argv, []string{"apptainer", "exec", "instance://gormes-test", "bash", "-l", "-c", "echo $SHELL"})

	stopSpec, err := BuildSingularityStopCommand("apptainer", "gormes-test")
	if err != nil {
		t.Fatalf("BuildSingularityStopCommand returned error: %v", err)
	}
	wantArgv(t, stopSpec.Argv, []string{"apptainer", "instance", "stop", "gormes-test"})
	if stopSpec.Timeout != SingularityInstanceStopTimeout {
		t.Fatalf("stop timeout = %s, want %s", stopSpec.Timeout, SingularityInstanceStopTimeout)
	}
	if stopSpec.Evidence.Code != EnvironmentSingularityCleanupPlanned {
		t.Fatalf("stop evidence = %+v, want planned", stopSpec.Evidence)
	}
}

func fakeSingularityLookup(paths map[string]string) func(string) string {
	return func(name string) string {
		return paths[name]
	}
}

func wantArgv(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %#v, want %#v", got, want)
	}
}

func lastSingularityEvidence(evidence []EnvironmentEvidence) EnvironmentEvidence {
	if len(evidence) == 0 {
		return EnvironmentEvidence{}
	}
	return evidence[len(evidence)-1]
}

func containsArg(argv []string, arg string) bool {
	for _, got := range argv {
		if got == arg {
			return true
		}
	}
	return false
}

func containsArgPair(argv []string, flag, value string) bool {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == flag && argv[i+1] == value {
			return true
		}
	}
	return false
}
