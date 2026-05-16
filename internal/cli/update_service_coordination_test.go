package cli

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestUpdateLockBlocksConcurrentMutation(t *testing.T) {
	lock := &fakeUpdateLock{err: errors.New("locked by pid=123")}
	mutated := false

	report := RunUpdateServiceCoordination(context.Background(), UpdateServiceCoordinationOptions{
		Lock: lock,
		Mutation: func(context.Context) UpdateReleaseBinaryReport {
			mutated = true
			return UpdateReleaseBinaryReport{}
		},
	})

	if !report.Failed {
		t.Fatalf("lock conflict should fail report: %+v", report)
	}
	if mutated {
		t.Fatal("mutation ran despite lock conflict")
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceUpdateLockBlocked)
}

func TestUpdateServiceCoordinationStopsRestartsManagedServiceAroundMutation(t *testing.T) {
	lock := &fakeUpdateLock{}
	service := &fakeUpdateManagedService{name: "gormes-gateway.service", running: true}
	events := []string{}

	report := RunUpdateServiceCoordination(context.Background(), UpdateServiceCoordinationOptions{
		Lock:          lock,
		Services:      []UpdateManagedService{service},
		DrainTimeout:  time.Second,
		HealthTimeout: 2 * time.Second,
		Mutation: func(context.Context) UpdateReleaseBinaryReport {
			events = append(events, "mutation")
			return UpdateReleaseBinaryReport{Evidence: []UpdateEvidence{{
				Kind:   UpdateEvidenceReleaseSwapCompleted,
				Detail: "binary swapped",
			}}}
		},
	})

	if report.Failed {
		t.Fatalf("coordination failed: %+v", report)
	}
	got := append([]string{}, lock.events...)
	got = append(got, service.events...)
	got = append(got, events...)
	want := []string{"acquire", "release", "status", "drain:1s", "stop", "start", "health:2s", "mutation"}
	if !sameElementsIgnoringOrder(got, want) {
		t.Fatalf("events = %#v\nwant elements %#v", got, want)
	}
	service.assertOrder(t, "status", "drain:1s", "stop", "start", "health:2s")
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceUpdateLockAcquired)
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseServiceDrainCompleted)
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseServiceStopCompleted)
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseServiceRestartCompleted)
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseServiceHealthPassed)
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceUpdateLockReleased)
}

func TestUpdateServiceCoordinationBlocksUnmanagedSessionsUnlessForce(t *testing.T) {
	mutated := false
	report := RunUpdateServiceCoordination(context.Background(), UpdateServiceCoordinationOptions{
		Lock: &fakeUpdateLock{},
		UnmanagedSessions: []UpdateUnmanagedSession{{
			PID:    42,
			Detail: "manual gateway active_agents=1",
		}},
		Mutation: func(context.Context) UpdateReleaseBinaryReport {
			mutated = true
			return UpdateReleaseBinaryReport{}
		},
	})
	if !report.Failed {
		t.Fatalf("unmanaged session should block without force: %+v", report)
	}
	if mutated {
		t.Fatal("mutation ran despite unmanaged session blocker")
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseServiceUnmanagedBlocked)

	forced := RunUpdateServiceCoordination(context.Background(), UpdateServiceCoordinationOptions{
		Lock:  &fakeUpdateLock{},
		Force: true,
		UnmanagedSessions: []UpdateUnmanagedSession{{
			PID:    42,
			Detail: "manual gateway active_agents=1",
		}},
		Mutation: func(context.Context) UpdateReleaseBinaryReport {
			return UpdateReleaseBinaryReport{Evidence: []UpdateEvidence{{Kind: UpdateEvidenceReleaseSwapCompleted}}}
		},
	})
	if forced.Failed {
		t.Fatalf("forced unmanaged session should continue: %+v", forced)
	}
	assertUpdateEvidence(t, UpdateReport{Evidence: forced.Evidence}, UpdateEvidenceReleaseServiceUnmanagedForced)
}

func TestUpdateServiceCoordinationRestoresManagedServiceAfterRollback(t *testing.T) {
	service := &fakeUpdateManagedService{name: "gormes-gateway.service", running: true}

	report := RunUpdateServiceCoordination(context.Background(), UpdateServiceCoordinationOptions{
		Lock:     &fakeUpdateLock{},
		Services: []UpdateManagedService{service},
		Mutation: func(context.Context) UpdateReleaseBinaryReport {
			return UpdateReleaseBinaryReport{
				Failed: true,
				Evidence: []UpdateEvidence{{
					Kind:   UpdateEvidenceReleaseRollbackCompleted,
					Detail: "snapshot restored",
				}},
			}
		},
	})

	if !report.Failed {
		t.Fatalf("failed mutation should stay failed after service restore: %+v", report)
	}
	service.assertOrder(t, "status", "drain:0s", "stop", "start", "health:0s")
	assertUpdateEvidence(t, UpdateReport{Evidence: report.Evidence}, UpdateEvidenceReleaseServiceRestoreCompleted)
}

func TestUpdateFileLockBlocksSecondOwnerUntilRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update.lock")
	first := NewFileUpdateLock(path, "first-owner")
	second := NewFileUpdateLock(path, "second-owner")

	handle, err := first.AcquireUpdateLock(context.Background())
	if err != nil {
		t.Fatalf("first lock acquire failed: %v", err)
	}
	if _, err := second.AcquireUpdateLock(context.Background()); err == nil || !strings.Contains(err.Error(), "first-owner") {
		t.Fatalf("second acquire err = %v, want first owner detail", err)
	}
	if err := handle.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	handle, err = second.AcquireUpdateLock(context.Background())
	if err != nil {
		t.Fatalf("second acquire after release failed: %v", err)
	}
	if err := handle.Release(); err != nil {
		t.Fatalf("second release: %v", err)
	}
}

type fakeUpdateLock struct {
	err    error
	events []string
}

func (l *fakeUpdateLock) AcquireUpdateLock(context.Context) (UpdateLockHandle, error) {
	l.events = append(l.events, "acquire")
	if l.err != nil {
		return nil, l.err
	}
	return fakeUpdateLockHandle{release: func() { l.events = append(l.events, "release") }}, nil
}

type fakeUpdateLockHandle struct {
	release func()
}

func (h fakeUpdateLockHandle) Release() error {
	if h.release != nil {
		h.release()
	}
	return nil
}

type fakeUpdateManagedService struct {
	name    string
	running bool
	events  []string
}

func (s *fakeUpdateManagedService) UpdateServiceName() string {
	return s.name
}

func (s *fakeUpdateManagedService) UpdateServiceRunning(context.Context) (bool, error) {
	s.events = append(s.events, "status")
	return s.running, nil
}

func (s *fakeUpdateManagedService) DrainUpdateService(_ context.Context, timeout time.Duration) error {
	s.events = append(s.events, "drain:"+timeout.String())
	return nil
}

func (s *fakeUpdateManagedService) StopUpdateService(context.Context) error {
	s.events = append(s.events, "stop")
	s.running = false
	return nil
}

func (s *fakeUpdateManagedService) StartUpdateService(context.Context) error {
	s.events = append(s.events, "start")
	s.running = true
	return nil
}

func (s *fakeUpdateManagedService) HealthCheckUpdateService(_ context.Context, timeout time.Duration) error {
	s.events = append(s.events, "health:"+timeout.String())
	return nil
}

func (s *fakeUpdateManagedService) assertOrder(t *testing.T, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(s.events, want) {
		t.Fatalf("service events = %#v\nwant %#v", s.events, want)
	}
}

func sameElementsIgnoringOrder(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := map[string]int{}
	for _, item := range left {
		counts[item]++
	}
	for _, item := range right {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}
	return true
}
