package plannerloop

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/builderloop"
)

func TestSummarizeAutoloopAuditAggregatesRecentLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.jsonl")
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	events := []builderloop.LedgerEvent{
		{TS: now.Add(-2 * time.Hour), Event: "run_started", Status: "started"},
		{TS: now.Add(-90 * time.Minute), Event: "worker_claimed", Task: "2/2.B.4/WhatsApp identity", Status: "claimed"},
		{TS: now.Add(-80 * time.Minute), Event: "worker_failed", Task: "2/2.B.4/WhatsApp identity", Status: "worktree_dirty"},
		{TS: now.Add(-70 * time.Minute), Event: "worker_claimed", Task: "3/3.E.7/Memory scope", Status: "claimed"},
		{TS: now.Add(-60 * time.Minute), Event: "worker_promoted", Task: "3/3.E.7/Memory scope", Status: "promoted"},
		{TS: now.Add(-50 * time.Minute), Event: "worker_success", Task: "3/3.E.7/Memory scope", Status: "success"},
		{TS: now.Add(-8 * 24 * time.Hour), Event: "worker_failed", Task: "2/2.B.3/Old", Status: "backend_failed"},
	}
	for _, event := range events {
		if err := builderloop.AppendLedgerEvent(path, event); err != nil {
			t.Fatalf("AppendLedgerEvent() error = %v", err)
		}
	}

	audit, err := SummarizeAutoloopAudit(path, 7*24*time.Hour, now)
	if err != nil {
		t.Fatalf("SummarizeAutoloopAudit() error = %v", err)
	}

	if audit.Runs != 1 || audit.Claimed != 2 || audit.Failed != 1 || audit.Promoted != 1 || audit.Succeeded != 1 {
		t.Fatalf("audit counts = runs:%d claimed:%d failed:%d promoted:%d succeeded:%d", audit.Runs, audit.Claimed, audit.Failed, audit.Promoted, audit.Succeeded)
	}
	if got := audit.FailStatusCounts["worktree_dirty"]; got != 1 {
		t.Fatalf("worktree_dirty count = %d, want 1", got)
	}
	if got := audit.ProductivityPercent(); got != 50 {
		t.Fatalf("ProductivityPercent() = %d, want 50", got)
	}
	if len(audit.ToxicSubphases) != 1 || audit.ToxicSubphases[0].SubphaseID != "2/2.B.4" {
		t.Fatalf("ToxicSubphases = %#v, want 2/2.B.4", audit.ToxicSubphases)
	}
	if len(audit.RecentFailedTasks) != 1 || audit.RecentFailedTasks[0].Status != "worktree_dirty" {
		t.Fatalf("RecentFailedTasks = %#v, want dirty failed task", audit.RecentFailedTasks)
	}
}

func TestSummarizeAutoloopAuditMissingLedgerIsEmpty(t *testing.T) {
	audit, err := SummarizeAutoloopAudit(filepath.Join(t.TempDir(), "missing.jsonl"), time.Hour, time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("SummarizeAutoloopAudit() error = %v", err)
	}
	if audit.Runs != 0 || audit.Claimed != 0 || audit.ProductivityPercent() != 0 {
		t.Fatalf("audit = %#v, want empty summary", audit)
	}
}

func TestControlPlaneSubphaseIDExported(t *testing.T) {
	if ControlPlaneSubphaseID != "control-plane/backend" {
		t.Fatalf("ControlPlaneSubphaseID = %q, want %q", ControlPlaneSubphaseID, "control-plane/backend")
	}
}

func TestSubphaseFromTaskEmptyReturnsControlPlane(t *testing.T) {
	cases := []struct {
		name string
		task string
		want string
	}{
		{name: "empty string", task: "", want: ControlPlaneSubphaseID},
		{name: "whitespace only", task: "   ", want: ControlPlaneSubphaseID},
		{name: "tabs and spaces", task: "\t  \n", want: ControlPlaneSubphaseID},
		{name: "non-conforming label", task: "backend_failed", want: ControlPlaneSubphaseID},
		{name: "named row preserved", task: "5/5.J/Foo", want: "5/5.J"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := subphaseFromTask(tc.task); got != tc.want {
				t.Fatalf("subphaseFromTask(%q) = %q, want %q", tc.task, got, tc.want)
			}
		})
	}
}

func TestSummarizeAutoloopAuditBlankBucket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.jsonl")
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	events := []builderloop.LedgerEvent{
		{TS: now.Add(-3 * time.Hour), Event: "worker_failed", Task: "", Status: "backend_waiting_for_stdin"},
		{TS: now.Add(-2 * time.Hour), Event: "worker_failed", Task: "", Status: "backend_killed"},
		{TS: now.Add(-1 * time.Hour), Event: "worker_failed", Task: "", Status: "backend_no_progress"},
	}
	for _, event := range events {
		if err := builderloop.AppendLedgerEvent(path, event); err != nil {
			t.Fatalf("AppendLedgerEvent() error = %v", err)
		}
	}

	audit, err := SummarizeAutoloopAudit(path, 7*24*time.Hour, now)
	if err != nil {
		t.Fatalf("SummarizeAutoloopAudit() error = %v", err)
	}

	for _, row := range audit.ToxicSubphases {
		if row.SubphaseID == "" {
			t.Fatalf("ToxicSubphases contains empty subphase_id: %#v", audit.ToxicSubphases)
		}
	}
	for _, row := range audit.HotSubphases {
		if row.SubphaseID == "" {
			t.Fatalf("HotSubphases contains empty subphase_id: %#v", audit.HotSubphases)
		}
	}

	var toxic *SubphaseAuditRow
	for i := range audit.ToxicSubphases {
		if audit.ToxicSubphases[i].SubphaseID == ControlPlaneSubphaseID {
			toxic = &audit.ToxicSubphases[i]
			break
		}
	}
	if toxic == nil {
		t.Fatalf("ToxicSubphases missing ControlPlaneSubphaseID row: %#v", audit.ToxicSubphases)
	}
	if toxic.Failed != 3 {
		t.Fatalf("ControlPlaneSubphaseID toxic.Failed = %d, want 3", toxic.Failed)
	}

	var hot *SubphaseAuditRow
	for i := range audit.HotSubphases {
		if audit.HotSubphases[i].SubphaseID == ControlPlaneSubphaseID {
			hot = &audit.HotSubphases[i]
			break
		}
	}
	if hot == nil {
		t.Fatalf("HotSubphases missing ControlPlaneSubphaseID row: %#v", audit.HotSubphases)
	}

	for _, row := range audit.RecentFailedTasks {
		if row.SubphaseID == "" {
			t.Fatalf("RecentFailedTasks contains empty subphase_id: %#v", audit.RecentFailedTasks)
		}
	}
}

func TestSummarizeAutoloopAuditIncludesRecentFailureDetail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.jsonl")
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	longStderr := strings.Repeat("x", 4096)
	events := []builderloop.LedgerEvent{
		{
			TS:         now.Add(-3 * time.Hour),
			Event:      "worker_failed",
			Task:       "",
			Status:     "backend_failed",
			Detail:     "backend_waiting_for_stdin: backend requested interactive input",
			StderrTail: "Reading additional input from stdin",
		},
		{
			TS:         now.Add(-2 * time.Hour),
			Event:      "worker_failed",
			Task:       "5/5.N/Long stderr",
			Status:     "backend_failed",
			StderrTail: longStderr,
		},
		{
			TS:     now.Add(-1 * time.Hour),
			Event:  "worker_failed",
			Task:   "5/5.N/No detail",
			Status: "worktree_dirty",
		},
	}
	for _, event := range events {
		if err := builderloop.AppendLedgerEvent(path, event); err != nil {
			t.Fatalf("AppendLedgerEvent() error = %v", err)
		}
	}

	audit, err := SummarizeAutoloopAudit(path, 7*24*time.Hour, now)
	if err != nil {
		t.Fatalf("SummarizeAutoloopAudit() error = %v", err)
	}

	if audit.Failed != 3 {
		t.Fatalf("Failed = %d, want 3", audit.Failed)
	}
	if got := audit.FailStatusCounts["backend_failed"]; got != 2 {
		t.Fatalf("backend_failed count = %d, want 2", got)
	}
	if got := audit.FailStatusCounts["worktree_dirty"]; got != 1 {
		t.Fatalf("worktree_dirty count = %d, want 1", got)
	}
	if len(audit.RecentFailedTasks) != 3 {
		t.Fatalf("RecentFailedTasks len = %d, want 3: %#v", len(audit.RecentFailedTasks), audit.RecentFailedTasks)
	}

	controlPlane := audit.RecentFailedTasks[0]
	if controlPlane.SubphaseID != ControlPlaneSubphaseID {
		t.Fatalf("control-plane SubphaseID = %q, want %q", controlPlane.SubphaseID, ControlPlaneSubphaseID)
	}
	if !strings.Contains(controlPlane.Detail, "backend_waiting_for_stdin") {
		t.Fatalf("control-plane Detail = %q, want classified detail", controlPlane.Detail)
	}
	if !strings.Contains(controlPlane.Detail, "Reading additional input from stdin") {
		t.Fatalf("control-plane Detail = %q, want stderr clue", controlPlane.Detail)
	}

	longDetail := audit.RecentFailedTasks[1].Detail
	if len(longDetail) != 240 {
		t.Fatalf("long Detail length = %d, want 240", len(longDetail))
	}
	if strings.Contains(longDetail, "\n") {
		t.Fatalf("long Detail should be a single-line excerpt: %q", longDetail)
	}

	if audit.RecentFailedTasks[2].Detail != "" {
		t.Fatalf("empty-detail row Detail = %q, want empty", audit.RecentFailedTasks[2].Detail)
	}
}

func TestSummarizeAutoloopAuditPreservesNamedRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.jsonl")
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	events := []builderloop.LedgerEvent{
		{TS: now.Add(-4 * time.Hour), Event: "worker_claimed", Task: "5/5.J/Foo", Status: "claimed"},
		{TS: now.Add(-3 * time.Hour), Event: "worker_failed", Task: "5/5.J/Foo", Status: "worktree_dirty"},
		{TS: now.Add(-2 * time.Hour), Event: "worker_failed", Task: "", Status: "backend_killed"},
	}
	for _, event := range events {
		if err := builderloop.AppendLedgerEvent(path, event); err != nil {
			t.Fatalf("AppendLedgerEvent() error = %v", err)
		}
	}

	audit, err := SummarizeAutoloopAudit(path, 7*24*time.Hour, now)
	if err != nil {
		t.Fatalf("SummarizeAutoloopAudit() error = %v", err)
	}

	var named, controlPlane *SubphaseAuditRow
	for i := range audit.ToxicSubphases {
		switch audit.ToxicSubphases[i].SubphaseID {
		case "5/5.J":
			named = &audit.ToxicSubphases[i]
		case ControlPlaneSubphaseID:
			controlPlane = &audit.ToxicSubphases[i]
		}
	}
	if named == nil {
		t.Fatalf("ToxicSubphases missing 5/5.J row: %#v", audit.ToxicSubphases)
	}
	if controlPlane == nil {
		t.Fatalf("ToxicSubphases missing ControlPlaneSubphaseID row: %#v", audit.ToxicSubphases)
	}
	if named.Failed != 1 {
		t.Fatalf("5/5.J Failed = %d, want 1", named.Failed)
	}
	if controlPlane.Failed != 1 {
		t.Fatalf("ControlPlaneSubphaseID Failed = %d, want 1", controlPlane.Failed)
	}
}
