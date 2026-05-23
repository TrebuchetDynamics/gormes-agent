package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
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

func TestUpdateReleaseServiceCoordinationUsesFleetSupervisorForProfileConfig(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INSTALL_HOME", t.TempDir())
	writeOneshotFlagConfig(t, []byte(`
config_version = 2
[profiles.main]
enabled = true
[profiles.ops]
enabled = true
`))
	targetVersion := nextPatchVersionForTest(t)
	events := []string{}
	fake := &fakeUpdateFleetSupervisor{
		events: &events,
		status: gateway.FleetStatus{Profiles: []gateway.FleetProfileStatus{
			{ProfileID: "main", Enabled: true, Runtime: gateway.FleetProfileRuntime{State: gateway.FleetRuntimeStateRunning, Live: true}},
			{ProfileID: "ops", Enabled: true, Runtime: gateway.FleetProfileRuntime{State: gateway.FleetRuntimeStateRunning, Live: true}},
		}},
		startReport: gateway.FleetOperationReport{
			Action:  gateway.FleetOperationStartAll,
			Results: []gateway.FleetOperationResult{{ProfileID: "main", Status: gateway.FleetOperationStatusStarted}, {ProfileID: "ops", Status: gateway.FleetOperationStatusStarted}},
			Summary: gateway.FleetOperationSummary{TargetedProfiles: 2, Succeeded: 2},
		},
		stopReport: gateway.FleetOperationReport{
			Action:  gateway.FleetOperationStopAll,
			Results: []gateway.FleetOperationResult{{ProfileID: "main", Status: gateway.FleetOperationStatusStopped}, {ProfileID: "ops", Status: gateway.FleetOperationStatusStopped}},
			Summary: gateway.FleetOperationSummary{TargetedProfiles: 2, Succeeded: 2},
		},
	}
	restore := updateFleetSupervisorForTest(t, fake)
	defer restore()
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
		LoadReleaseAssetManifest: func(context.Context, cli.UpdateReleasePlan) (cli.UpdateReleaseManifest, string, error) {
			return cli.UpdateReleaseManifest{SchemaVersion: 1}, "/tmp/payload", nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--json")
	if err != nil {
		t.Fatalf("update --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	wantEvents := []string{"status", "stop-all", "mutation", "start-all", "status"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %#v, want release coordination through FleetSupervisor %#v", events, wantEvents)
	}
	if fake.stopCalls != 1 || fake.startCalls != 1 || fake.restartCalls != 0 {
		t.Fatalf("fleet calls start=%d stop=%d restart=%d, want start/stop only", fake.startCalls, fake.stopCalls, fake.restartCalls)
	}
	var got struct {
		Evidence []struct {
			Kind   string `json:"kind"`
			Detail string `json:"detail"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\n%s", err, stdout)
	}
	assertUpdateReleaseEvidence(t, got.Evidence, string(cli.UpdateEvidenceReleaseServiceStopCompleted), "gormes-profile-fleet")
	assertUpdateReleaseEvidence(t, got.Evidence, string(cli.UpdateEvidenceReleaseServiceRestartCompleted), "gormes-profile-fleet")
	assertUpdateReleaseEvidence(t, got.Evidence, string(cli.UpdateEvidenceReleaseServiceHealthPassed), "gormes-profile-fleet")
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

func assertUpdateReleaseEvidence(t *testing.T, evidence []struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}, kind, detail string) {
	t.Helper()
	for _, ev := range evidence {
		if ev.Kind == kind && ev.Detail == detail {
			return
		}
	}
	t.Fatalf("evidence = %+v, want kind=%s detail=%s", evidence, kind, detail)
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
