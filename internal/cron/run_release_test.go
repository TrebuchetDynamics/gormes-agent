package cron

import (
	"errors"
	"sync"
	"testing"
)

type recordingCloser struct {
	name      string
	mu        *sync.Mutex
	calls     *int
	order     *[]string
	returnErr error
}

func (c *recordingCloser) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	*c.calls++
	*c.order = append(*c.order, c.name)
	return c.returnErr
}

type recordingKiller struct {
	called []int
	err    error
}

func (k *recordingKiller) Kill(pid int) error {
	k.called = append(k.called, pid)
	return k.err
}

func TestRunReleaseLedger_ClosesAllRegisteredClosers(t *testing.T) {
	mu := &sync.Mutex{}
	var calls int
	var order []string
	c1 := &recordingCloser{name: "first", mu: mu, calls: &calls, order: &order}
	c2 := &recordingCloser{name: "second", mu: mu, calls: &calls, order: &order}
	c3 := &recordingCloser{name: "third", mu: mu, calls: &calls, order: &order}

	ledger := NewRunReleaseLedger()
	ledger.RegisterCloser("session-db-a", c1)
	ledger.RegisterCloser("session-db-b", c2)
	ledger.RegisterCloser("session-db-c", c3)

	evidence, err := ledger.Release(nil)
	if err != nil {
		t.Fatalf("Release error = %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	wantOrder := []string{"first", "second", "third"}
	for i, name := range wantOrder {
		if order[i] != name {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], name)
		}
	}
	closedCount := 0
	for _, ev := range evidence {
		if ev.Code == ReleaseEvidenceSessionDBClosed {
			closedCount++
		}
	}
	if closedCount != 3 {
		t.Fatalf("evidence cron_release_session_db_closed count = %d, want 3 (one per closable)", closedCount)
	}
}

func TestRunReleaseLedger_KillsRegisteredSubprocesses(t *testing.T) {
	killer := &recordingKiller{}
	ledger := NewRunReleaseLedger()
	ledger.RegisterSubprocess(1234)
	ledger.RegisterSubprocess(5678)

	evidence, err := ledger.Release(killer)
	if err != nil {
		t.Fatalf("Release error = %v", err)
	}
	if len(killer.called) != 2 || killer.called[0] != 1234 || killer.called[1] != 5678 {
		t.Fatalf("killer.called = %v, want [1234 5678]", killer.called)
	}
	killedPIDs := map[int]bool{}
	for _, ev := range evidence {
		if ev.Code != ReleaseEvidenceSubprocessKilled {
			continue
		}
		pid, ok := ev.Fields["pid"].(int)
		if !ok {
			t.Fatalf("subprocess evidence pid field = %v, want int", ev.Fields["pid"])
		}
		killedPIDs[pid] = true
	}
	if !killedPIDs[1234] || !killedPIDs[5678] {
		t.Fatalf("expected subprocess_killed evidence for 1234 and 5678; got %v", killedPIDs)
	}
}

func TestRunReleaseLedger_PartialFailureContinues(t *testing.T) {
	mu := &sync.Mutex{}
	var calls int
	var order []string
	c1 := &recordingCloser{name: "first", mu: mu, calls: &calls, order: &order}
	c2 := &recordingCloser{name: "second", mu: mu, calls: &calls, order: &order, returnErr: errors.New("boom")}
	c3 := &recordingCloser{name: "third", mu: mu, calls: &calls, order: &order}

	ledger := NewRunReleaseLedger()
	ledger.RegisterIdleClosable("rt-a", c1)
	ledger.RegisterIdleClosable("rt-b", c2)
	ledger.RegisterIdleClosable("rt-c", c3)

	evidence, err := ledger.Release(nil)
	if err == nil {
		t.Fatalf("Release error = nil, want aggregate failure")
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (continued past failure)", calls)
	}
	if order[2] != "third" {
		t.Fatalf("third closer not called; order = %v", order)
	}
	failed := false
	for _, ev := range evidence {
		if ev.Code == ReleaseEvidenceHTTPIdleClosedFailed {
			failed = true
			if got := ev.Fields["error"]; got != "boom" {
				t.Fatalf("failed evidence error = %v, want %q", got, "boom")
			}
		}
	}
	if !failed {
		t.Fatalf("evidence = %+v, want one cron_release_http_idle_closed_failed entry", evidence)
	}
}

func TestRunReleaseLedger_NoResourceRecordsSkippedEvidence(t *testing.T) {
	ledger := NewRunReleaseLedger()
	evidence, err := ledger.Release(nil)
	if err != nil {
		t.Fatalf("Release error = %v", err)
	}
	skipped := 0
	for _, ev := range evidence {
		if ev.Code == ReleaseEvidenceSkippedNoResource {
			skipped++
		}
	}
	if skipped != 1 {
		t.Fatalf("evidence = %+v, want exactly one skipped_no_resource entry", evidence)
	}
}

func TestRunReleaseLedger_IsIdempotent(t *testing.T) {
	mu := &sync.Mutex{}
	var calls int
	var order []string
	c := &recordingCloser{name: "only", mu: mu, calls: &calls, order: &order}
	killer := &recordingKiller{}

	ledger := NewRunReleaseLedger()
	ledger.RegisterCloser("session-db", c)
	ledger.RegisterSubprocess(42)

	if _, err := ledger.Release(killer); err != nil {
		t.Fatalf("first Release error = %v", err)
	}
	if calls != 1 || len(killer.called) != 1 {
		t.Fatalf("first release: calls=%d killed=%v, want 1 close + 1 kill", calls, killer.called)
	}

	evidence, err := ledger.Release(killer)
	if err != nil {
		t.Fatalf("second Release error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("close called twice: calls = %d", calls)
	}
	if len(killer.called) != 1 {
		t.Fatalf("kill called twice: killer.called = %v", killer.called)
	}
	if len(evidence) != 1 || evidence[0].Code != ReleaseEvidenceSkippedNoResource {
		t.Fatalf("second release evidence = %+v, want one skipped_no_resource entry", evidence)
	}
}
