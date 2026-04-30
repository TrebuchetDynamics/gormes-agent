package tools

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestStatefulToolMigrationQueueRegistersDomains(t *testing.T) {
	root := t.TempDir()
	q := NewStatefulToolMigrationQueue(StatefulToolQueueOptions{MutationRoots: []string{root}})

	plans := []StatefulToolPlan{
		{Name: "write_file", Domain: ToolStateDomainFile, RootPolicy: ToolRootPolicyInjectedXDG, RollbackPolicy: ToolRollbackPolicyCheckpoint, ConcurrencyPolicy: ToolConcurrencySerializedWrites, OwnerRow: "File write/patch tool port"},
		{Name: "session_update", Domain: ToolStateDomainSession, RootPolicy: ToolRootPolicyGormesData, RollbackPolicy: ToolRollbackPolicyAuditLog, ConcurrencyPolicy: ToolConcurrencySerializedWrites, OwnerRow: "Session state tool port"},
		{Name: "checkpoint_restore", Domain: ToolStateDomainCheckpoint, RootPolicy: ToolRootPolicyGormesData, RollbackPolicy: ToolRollbackPolicyCheckpoint, ConcurrencyPolicy: ToolConcurrencySerializedWrites, OwnerRow: "Checkpoint restore tool port"},
		{Name: "terminal", Domain: ToolStateDomainProcess, RootPolicy: ToolRootPolicyInjectedXDG, RollbackPolicy: ToolRollbackPolicyAuditLog, ConcurrencyPolicy: ToolConcurrencySerializedWrites, OwnerRow: "Terminal process execution port"},
	}
	for _, plan := range plans {
		if ev := q.Register(plan); ev.Code != ToolStateContractRegistered {
			t.Fatalf("Register(%s) evidence = %#v", plan.Name, ev)
		}
	}

	got := q.Plans()
	if len(got) != len(plans) {
		t.Fatalf("Plans len = %d, want %d", len(got), len(plans))
	}
	if got[0].Name != "checkpoint_restore" || got[1].Name != "session_update" || got[2].Name != "terminal" || got[3].Name != "write_file" {
		t.Fatalf("Plans not stably sorted by name: %#v", got)
	}
	for _, plan := range got {
		if plan.Domain == "" || plan.RootPolicy == "" || plan.RollbackPolicy == "" || plan.ConcurrencyPolicy == "" || plan.OwnerRow == "" {
			t.Fatalf("registered plan missing required state contract evidence: %#v", plan)
		}
	}
}

func TestStatefulToolMigrationQueueRejectsMissingRollback(t *testing.T) {
	q := NewStatefulToolMigrationQueue(StatefulToolQueueOptions{MutationRoots: []string{t.TempDir()}})

	ev := q.Register(StatefulToolPlan{
		Name:              "write_file",
		Domain:            ToolStateDomainFile,
		RootPolicy:        ToolRootPolicyInjectedXDG,
		ConcurrencyPolicy: ToolConcurrencySerializedWrites,
		OwnerRow:          "File write/patch tool port",
	})

	if ev.Code != ToolStateContractMissing {
		t.Fatalf("evidence code = %q, want %q (%#v)", ev.Code, ToolStateContractMissing, ev)
	}
	if ev.Tool != "write_file" || ev.Message == "" {
		t.Fatalf("missing bounded evidence: %#v", ev)
	}
	if _, ok := q.Plan("write_file"); ok {
		t.Fatalf("invalid write-capable plan was registered")
	}
}

func TestStatefulToolMigrationQueuePathIsolation(t *testing.T) {
	root := t.TempDir()
	q := NewStatefulToolMigrationQueue(StatefulToolQueueOptions{MutationRoots: []string{root}})
	if ev := q.Register(StatefulToolPlan{Name: "write_file", Domain: ToolStateDomainFile, RootPolicy: ToolRootPolicyInjectedXDG, RollbackPolicy: ToolRollbackPolicyCheckpoint, ConcurrencyPolicy: ToolConcurrencySerializedWrites, OwnerRow: "File write/patch tool port"}); ev.Code != ToolStateContractRegistered {
		t.Fatalf("register evidence: %#v", ev)
	}

	if ev := q.AuthorizePath("write_file", root+"/nested/file.txt"); ev.Code != ToolStatePathAllowed {
		t.Fatalf("inside root evidence = %#v", ev)
	}
	if ev := q.AuthorizePath("write_file", root+"/../escape.txt"); ev.Code != ToolPathDenied {
		t.Fatalf("traversal evidence = %#v, want %q", ev, ToolPathDenied)
	}
	if ev := q.AuthorizePath("write_file", "/etc/passwd"); ev.Code != ToolPathDenied {
		t.Fatalf("foreign absolute evidence = %#v, want %q", ev, ToolPathDenied)
	}
}

func TestStatefulToolMigrationQueueSerializedWrites(t *testing.T) {
	q := NewStatefulToolMigrationQueue(StatefulToolQueueOptions{MutationRoots: []string{t.TempDir()}})
	if ev := q.Register(StatefulToolPlan{Name: "write_file", Domain: ToolStateDomainFile, RootPolicy: ToolRootPolicyInjectedXDG, RollbackPolicy: ToolRollbackPolicyCheckpoint, ConcurrencyPolicy: ToolConcurrencySerializedWrites, OwnerRow: "File write/patch tool port"}); ev.Code != ToolStateContractRegistered {
		t.Fatalf("register write_file evidence: %#v", ev)
	}
	if ev := q.Register(StatefulToolPlan{Name: "read_file", Domain: ToolStateDomainReadOnly, RootPolicy: ToolRootPolicyInjectedXDG, RollbackPolicy: ToolRollbackPolicyNone, ConcurrencyPolicy: ToolConcurrencyConcurrentReads, OwnerRow: "File read tool port"}); ev.Code != ToolStateContractRegistered {
		t.Fatalf("register read_file evidence: %#v", ev)
	}

	var mu sync.Mutex
	order := []string{}
	block := make(chan struct{})
	firstStarted := make(chan struct{})
	secondDone := make(chan struct{})
	go func() {
		_ = q.Run(context.Background(), "write_file", func(context.Context) error {
			mu.Lock(); order = append(order, "first-start"); mu.Unlock()
			close(firstStarted)
			<-block
			mu.Lock(); order = append(order, "first-done"); mu.Unlock()
			return nil
		})
	}()
	<-firstStarted
	go func() {
		_ = q.Run(context.Background(), "write_file", func(context.Context) error {
			mu.Lock(); order = append(order, "second"); mu.Unlock()
			return nil
		})
		close(secondDone)
	}()

	select {
	case <-secondDone:
		t.Fatalf("second write completed before first write released")
	case <-time.After(25 * time.Millisecond):
	}
	close(block)
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatalf("second write did not finish after first write released")
	}
	mu.Lock(); got := append([]string(nil), order...); mu.Unlock()
	want := []string{"first-start", "first-done", "second"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("write execution order = %#v, want %#v", got, want)
	}

	readStarted := make(chan struct{}, 2)
	readRelease := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			_ = q.Run(context.Background(), "read_file", func(context.Context) error {
				readStarted <- struct{}{}
				<-readRelease
				return nil
			})
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-readStarted:
		case <-time.After(time.Second):
			t.Fatalf("read-only tool did not start concurrently")
		}
	}
	close(readRelease)
}

func TestStatefulToolMigrationQueueNoRuntimePort(t *testing.T) {
	q := NewStatefulToolMigrationQueue(StatefulToolQueueOptions{MutationRoots: []string{t.TempDir()}})
	if ev := q.Register(StatefulToolPlan{Name: "terminal", Domain: ToolStateDomainProcess, RootPolicy: ToolRootPolicyInjectedXDG, RollbackPolicy: ToolRollbackPolicyAuditLog, ConcurrencyPolicy: ToolConcurrencySerializedWrites, OwnerRow: "Terminal process execution port"}); ev.Code != ToolStateContractRegistered {
		t.Fatalf("register evidence: %#v", ev)
	}

	ev := q.RuntimePortEvidence("terminal")
	if ev.Code != ToolStateRuntimeNotPorted || ev.Tool != "terminal" || ev.Message == "" {
		t.Fatalf("runtime evidence = %#v, want deterministic not-ported evidence", ev)
	}
}
