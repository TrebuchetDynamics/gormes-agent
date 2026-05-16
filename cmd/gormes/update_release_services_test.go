package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestUpdateReleaseServiceCoordinationWrapsBinaryMutation(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INSTALL_HOME", t.TempDir())
	targetVersion := nextPatchVersionForTest(t)
	events := []string{}
	service := fakeUpdateCommandManagedService{name: "gormes-gateway.service"}
	command := newUpdateCommandWithSeams(updateCommandSeams{
		DetectInstallKind: func() cli.UpdateInstallKind {
			return cli.UpdateInstallKindRelease
		},
		RuntimePlatform: func() (string, string) {
			return "linux", "amd64"
		},
		LoadReleaseMetadata: func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error) {
			return nextPatchReleaseMetadataForTest(t, "abc1234"), nil
		},
		ReleaseUpdateLock: func() cli.UpdateLock {
			return fakeUpdateCommandLock{}
		},
		ReleaseManagedServices: func() []cli.UpdateManagedService {
			return []cli.UpdateManagedService{service}
		},
		RunReleaseServiceCoordination: func(ctx context.Context, opts cli.UpdateServiceCoordinationOptions) cli.UpdateReleaseBinaryReport {
			events = append(events, "coordination")
			if opts.Lock == nil || len(opts.Services) != 1 {
				t.Fatalf("service coordination opts = %+v, want lock and managed service", opts)
			}
			return opts.Mutation(ctx)
		},
		RunReleaseBinaryUpdate: func(context.Context, cli.UpdateReleaseBinaryOptions) cli.UpdateReleaseBinaryReport {
			events = append(events, "mutation")
			return cli.UpdateReleaseBinaryReport{
				NewVersion: targetVersion,
				Evidence: []cli.UpdateEvidence{{
					Kind:   cli.UpdateEvidenceReleaseSwapCompleted,
					Detail: "binary swapped",
				}},
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--json")
	if err != nil {
		t.Fatalf("update --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if !reflect.DeepEqual(events, []string{"coordination", "mutation"}) {
		t.Fatalf("events = %#v, want coordination wrapping mutation", events)
	}
	var got struct {
		Evidence []struct {
			Kind string `json:"kind"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\n%s", err, stdout)
	}
	found := false
	for _, ev := range got.Evidence {
		if ev.Kind == string(cli.UpdateEvidenceReleaseSwapCompleted) {
			found = true
		}
	}
	if !found {
		t.Fatalf("json evidence = %+v, want swap evidence", got.Evidence)
	}
}

func TestUpdateReleaseServiceCoordinationReceivesForcePolicy(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INSTALL_HOME", t.TempDir())
	command := newUpdateCommandWithSeams(updateCommandSeams{
		DetectInstallKind: func() cli.UpdateInstallKind {
			return cli.UpdateInstallKindRelease
		},
		RuntimePlatform: func() (string, string) {
			return "linux", "amd64"
		},
		LoadReleaseMetadata: func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error) {
			return nextPatchReleaseMetadataForTest(t, "abc1234"), nil
		},
		ReleaseUnmanagedSessions: func(context.Context) []cli.UpdateUnmanagedSession {
			return []cli.UpdateUnmanagedSession{{PID: 42, Detail: "manual gateway active_agents=1"}}
		},
		RunReleaseServiceCoordination: func(ctx context.Context, opts cli.UpdateServiceCoordinationOptions) cli.UpdateReleaseBinaryReport {
			if !opts.Force {
				t.Fatal("--force was not forwarded to service coordination")
			}
			if len(opts.UnmanagedSessions) != 1 || opts.UnmanagedSessions[0].PID != 42 {
				t.Fatalf("unmanaged sessions = %+v, want detected session", opts.UnmanagedSessions)
			}
			return opts.Mutation(ctx)
		},
		RunReleaseBinaryUpdate: func(context.Context, cli.UpdateReleaseBinaryOptions) cli.UpdateReleaseBinaryReport {
			return cli.UpdateReleaseBinaryReport{Evidence: []cli.UpdateEvidence{{
				Kind: cli.UpdateEvidenceReleaseSwapCompleted,
			}}}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--force", "--json")
	if err != nil {
		t.Fatalf("update --force --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
}

func TestUpdateReleaseServiceCoordinationWrapsRollbackMutation(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INSTALL_HOME", t.TempDir())
	events := []string{}
	command := newUpdateCommandWithSeams(updateCommandSeams{
		RunReleaseServiceCoordination: func(ctx context.Context, opts cli.UpdateServiceCoordinationOptions) cli.UpdateReleaseBinaryReport {
			events = append(events, "coordination")
			return opts.Mutation(ctx)
		},
		RunReleaseRollback: func(context.Context, cli.UpdateReleaseRollbackOptions) cli.UpdateReleaseBinaryReport {
			events = append(events, "rollback")
			return cli.UpdateReleaseBinaryReport{
				SnapshotID: "snap-1",
				Evidence: []cli.UpdateEvidence{{
					Kind: cli.UpdateEvidenceReleaseRollbackCompleted,
				}},
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--rollback", "snap-1", "--json")
	if err != nil {
		t.Fatalf("update --rollback --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if !reflect.DeepEqual(events, []string{"coordination", "rollback"}) {
		t.Fatalf("events = %#v, want coordination wrapping rollback", events)
	}
}

type fakeUpdateCommandLock struct{}

func (fakeUpdateCommandLock) AcquireUpdateLock(context.Context) (cli.UpdateLockHandle, error) {
	return fakeUpdateCommandLockHandle{}, nil
}

type fakeUpdateCommandLockHandle struct{}

func (fakeUpdateCommandLockHandle) Release() error { return nil }

type fakeUpdateCommandManagedService struct {
	name string
}

func (s fakeUpdateCommandManagedService) UpdateServiceName() string {
	return s.name
}

func (fakeUpdateCommandManagedService) UpdateServiceRunning(context.Context) (bool, error) {
	return false, nil
}

func (fakeUpdateCommandManagedService) DrainUpdateService(context.Context, time.Duration) error {
	return nil
}

func (fakeUpdateCommandManagedService) StopUpdateService(context.Context) error {
	return nil
}

func (fakeUpdateCommandManagedService) StartUpdateService(context.Context) error {
	return nil
}

func (fakeUpdateCommandManagedService) HealthCheckUpdateService(context.Context, time.Duration) error {
	return nil
}
