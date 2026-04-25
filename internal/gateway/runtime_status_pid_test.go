package gateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeStatusPIDValidation_ClassifiesPIDIdentityEvidence(t *testing.T) {
	baseStatus := RuntimeStatus{
		Kind:         runtimeStatusKind,
		PID:          4242,
		StartTime:    100,
		Generation:   7,
		Command:      "gormes gateway",
		Argv:         []string{"gormes", "gateway"},
		GatewayState: GatewayStateRunning,
		Platforms: map[string]PlatformRuntimeStatus{
			"telegram": {State: PlatformStateRunning},
		},
		UpdatedAt: "2026-04-25T16:30:00Z",
	}

	tests := []struct {
		name       string
		writeState bool
		writePID   bool
		processes  fakeRuntimeProcessTable
		want       RuntimeProcessValidationStatus
		wantLive   bool
	}{
		{
			name:       "missing state",
			writeState: false,
			writePID:   false,
			processes:  fakeRuntimeProcessTable{},
			want:       RuntimeProcessValidationMissingState,
		},
		{
			name:       "missing PID file",
			writeState: true,
			writePID:   false,
			processes:  fakeRuntimeProcessTable{},
			want:       RuntimeProcessValidationMissingPIDFile,
		},
		{
			name:       "stale PID",
			writeState: true,
			writePID:   true,
			processes:  fakeRuntimeProcessTable{},
			want:       RuntimeProcessValidationStalePID,
		},
		{
			name:       "PID reused with mismatched start time",
			writeState: true,
			writePID:   true,
			processes: fakeRuntimeProcessTable{
				4242: {startTime: 200, command: "gormes gateway"},
			},
			want: RuntimeProcessValidationPIDReused,
		},
		{
			name:       "stopped process",
			writeState: true,
			writePID:   true,
			processes: fakeRuntimeProcessTable{
				4242: {startTime: 100, command: "gormes gateway", stopped: true},
			},
			want: RuntimeProcessValidationStopped,
		},
		{
			name:       "permission denied",
			writeState: true,
			writePID:   true,
			processes: fakeRuntimeProcessTable{
				4242: {err: errRuntimeProcessPermissionDenied},
			},
			want: RuntimeProcessValidationPermissionDenied,
		},
		{
			name:       "live matching process",
			writeState: true,
			writePID:   true,
			processes: fakeRuntimeProcessTable{
				4242: {startTime: 100, command: "gormes gateway"},
			},
			want:     RuntimeProcessValidationLive,
			wantLive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			statusPath := filepath.Join(root, "gateway_state.json")
			pidPath := filepath.Join(root, "gateway.pid")
			if tt.writeState {
				writeRuntimeStatusFixture(t, statusPath, baseStatus)
			}
			if tt.writePID {
				writeRuntimeStatusFixture(t, pidPath, RuntimeStatus{
					Kind:       baseStatus.Kind,
					PID:        baseStatus.PID,
					StartTime:  baseStatus.StartTime,
					Generation: baseStatus.Generation,
					Command:    baseStatus.Command,
					Argv:       baseStatus.Argv,
					UpdatedAt:  baseStatus.UpdatedAt,
				})
			}

			store := NewRuntimeStatusStore(statusPath)
			store.pidPath = pidPath
			store.processes = tt.processes

			snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(context.Background())
			if err != nil {
				t.Fatalf("read validated runtime status: %v", err)
			}
			if snapshot.Validation.Status != tt.want {
				t.Fatalf("validation status = %q, want %q", snapshot.Validation.Status, tt.want)
			}
			if snapshot.Validation.Live != tt.wantLive {
				t.Fatalf("validation live = %v, want %v", snapshot.Validation.Live, tt.wantLive)
			}
			if snapshot.Status.ProcessValidation.Status != tt.want {
				t.Fatalf("status validation = %+v, want %q", snapshot.Status.ProcessValidation, tt.want)
			}
			if tt.wantLive && snapshot.Status.GatewayState != GatewayStateRunning {
				t.Fatalf("live GatewayState = %q, want running", snapshot.Status.GatewayState)
			}
			if !tt.wantLive && tt.writeState && snapshot.Status.GatewayState == GatewayStateRunning {
				t.Fatalf("stale GatewayState = %q, want not running", snapshot.Status.GatewayState)
			}
		})
	}
}

func TestRuntimeStatusPIDValidation_CleansStaleStateWithoutDroppingLastError(t *testing.T) {
	root := t.TempDir()
	statusPath := filepath.Join(root, "gateway_state.json")
	pidPath := filepath.Join(root, "gateway.pid")
	status := RuntimeStatus{
		Kind:          runtimeStatusKind,
		PID:           4242,
		StartTime:     100,
		Generation:    7,
		Command:       "gormes gateway",
		Argv:          []string{"gormes", "gateway"},
		GatewayState:  GatewayStateRunning,
		ExitReason:    "last_error: discord startup denied",
		ActiveAgents:  3,
		UpdatedAt:     "2026-04-25T16:30:00Z",
		DrainTimeouts: []RuntimeDrainTimeoutEvidence{{SessionID: "sess-running", Reason: "shutdown_timeout"}},
		Platforms: map[string]PlatformRuntimeStatus{
			"telegram": {State: PlatformStateRunning, UpdatedAt: "2026-04-25T16:30:00Z"},
			"discord": {
				State:        PlatformStateFailed,
				ErrorMessage: "discord startup denied",
				UpdatedAt:    "2026-04-25T16:30:00Z",
			},
		},
	}
	writeRuntimeStatusFixture(t, statusPath, status)
	writeRuntimeStatusFixture(t, pidPath, RuntimeStatus{
		Kind:       status.Kind,
		PID:        status.PID,
		StartTime:  status.StartTime,
		Generation: status.Generation,
		Command:    status.Command,
		Argv:       status.Argv,
		UpdatedAt:  status.UpdatedAt,
	})

	store := NewRuntimeStatusStore(statusPath)
	store.pidPath = pidPath
	store.processes = fakeRuntimeProcessTable{}

	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(context.Background())
	if err != nil {
		t.Fatalf("read validated runtime status: %v", err)
	}
	if snapshot.Validation.Status != RuntimeProcessValidationStalePID {
		t.Fatalf("validation status = %q, want stale_pid", snapshot.Validation.Status)
	}
	if snapshot.Status.GatewayState != GatewayStateStopped {
		t.Fatalf("GatewayState = %q, want stopped", snapshot.Status.GatewayState)
	}
	if snapshot.Status.ActiveAgents != 0 {
		t.Fatalf("ActiveAgents = %d, want stale cleanup to zero active agents", snapshot.Status.ActiveAgents)
	}
	if snapshot.Status.ExitReason != status.ExitReason {
		t.Fatalf("ExitReason = %q, want preserved last error %q", snapshot.Status.ExitReason, status.ExitReason)
	}
	if got := snapshot.Status.Platforms["telegram"].State; got != PlatformStateStopped {
		t.Fatalf("telegram state = %q, want stopped after stale cleanup", got)
	}
	discord := snapshot.Status.Platforms["discord"]
	if discord.State != PlatformStateFailed || discord.ErrorMessage != "discord startup denied" {
		t.Fatalf("discord status = %+v, want failed state and error evidence preserved", discord)
	}
	if len(snapshot.Status.DrainTimeouts) != 1 {
		t.Fatalf("DrainTimeouts = %+v, want evidence preserved", snapshot.Status.DrainTimeouts)
	}
}

func TestRuntimeStatusPIDValidation_RejectsMismatchedPIDRecordGeneration(t *testing.T) {
	root := t.TempDir()
	statusPath := filepath.Join(root, "gateway_state.json")
	pidPath := filepath.Join(root, "gateway.pid")
	status := RuntimeStatus{
		Kind:         runtimeStatusKind,
		PID:          4242,
		StartTime:    100,
		Generation:   7,
		Command:      "gormes gateway",
		Argv:         []string{"gormes", "gateway"},
		GatewayState: GatewayStateRunning,
		Platforms: map[string]PlatformRuntimeStatus{
			"telegram": {State: PlatformStateRunning},
		},
	}
	writeRuntimeStatusFixture(t, statusPath, status)
	pidRecord := status
	pidRecord.Generation = 6
	writeRuntimeStatusFixture(t, pidPath, pidRecord)

	store := NewRuntimeStatusStore(statusPath)
	store.pidPath = pidPath
	store.processes = fakeRuntimeProcessTable{
		4242: {startTime: 100, command: "gormes gateway"},
	}

	snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(context.Background())
	if err != nil {
		t.Fatalf("read validated runtime status: %v", err)
	}
	if snapshot.Validation.Status != RuntimeProcessValidationStalePID {
		t.Fatalf("validation status = %q, want stale_pid for mismatched generation", snapshot.Validation.Status)
	}
	if snapshot.Validation.Live {
		t.Fatal("validation live = true, want false for mismatched generation")
	}
	if snapshot.Status.GatewayState == GatewayStateRunning {
		t.Fatalf("GatewayState = %q, want stale cleanup to refuse running", snapshot.Status.GatewayState)
	}
}

type fakeRuntimeProcessTable map[int]fakeRuntimeProcess

type fakeRuntimeProcess struct {
	startTime int64
	command   string
	stopped   bool
	err       error
}

func (f fakeRuntimeProcessTable) LookupRuntimeProcess(pid int) (runtimeProcessInfo, error) {
	record, ok := f[pid]
	if !ok {
		return runtimeProcessInfo{}, errRuntimeProcessNotFound
	}
	if record.err != nil {
		return runtimeProcessInfo{}, record.err
	}
	return runtimeProcessInfo{
		PID:       pid,
		StartTime: record.startTime,
		Command:   record.command,
		Stopped:   record.stopped,
	}, nil
}

func writeRuntimeStatusFixture(t *testing.T, path string, status RuntimeStatus) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create status dir: %v", err)
	}
	raw, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}
