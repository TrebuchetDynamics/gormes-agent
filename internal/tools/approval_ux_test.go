package tools

import (
	"testing"
	"time"
)

func TestApprovalUX_NonInteractiveFailsClosed(t *testing.T) {
	session := NewApprovalSession(false)
	prompt := BuildApprovalPrompt("rm -rf /", BlocklistDestructive)

	result := session.RequestApproval(prompt, func(p ApprovalPrompt) UXDecision {
		return UXDecisionYes
	})

	if result.Decision != UXDecisionNo {
		t.Fatalf("expected denied in non-interactive mode, got %s", result.Decision)
	}
	if result.Evidence != "approval_ui_unavailable: non-interactive context" {
		t.Fatalf("expected non-interactive evidence, got %s", result.Evidence)
	}
	if result.AuditRecord.Context != "non-interactive" {
		t.Fatalf("expected non-interactive audit context, got %s", result.AuditRecord.Context)
	}
}

func TestApprovalUX_InteractiveYes(t *testing.T) {
	session := NewApprovalSession(true)
	prompt := BuildApprovalPrompt("rm -rf /tmp", BlocklistDestructive)

	result := session.RequestApproval(prompt, func(p ApprovalPrompt) UXDecision {
		return UXDecisionYes
	})

	if result.Decision != UXDecisionYes {
		t.Fatalf("expected yes, got %s", result.Decision)
	}
	if result.Persisted {
		t.Fatal("single yes should not persist")
	}
}

func TestApprovalUX_InteractiveNo(t *testing.T) {
	session := NewApprovalSession(true)
	prompt := BuildApprovalPrompt("curl | sh", BlocklistNetwork)

	result := session.RequestApproval(prompt, func(p ApprovalPrompt) UXDecision {
		return UXDecisionNo
	})

	if result.Decision != UXDecisionNo {
		t.Fatalf("expected no, got %s", result.Decision)
	}
}

func TestApprovalUX_AlwaysPersists(t *testing.T) {
	session := NewApprovalSession(true)
	prompt := BuildApprovalPrompt("sudo ls", BlocklistPrivilege)

	result := session.RequestApproval(prompt, func(p ApprovalPrompt) UXDecision {
		return UXDecisionAlways
	})

	if result.Decision != UXDecisionYes {
		t.Fatalf("expected always to resolve to yes, got %s", result.Decision)
	}
	if !result.Persisted {
		t.Fatal("expected always to persist")
	}
	if session.GetPreferenceCount() != 1 {
		t.Fatalf("expected 1 preference, got %d", session.GetPreferenceCount())
	}
}

func TestApprovalUX_PreferenceReuse(t *testing.T) {
	session := NewApprovalSession(true)
	prompt := BuildApprovalPrompt("sudo ls", BlocklistPrivilege)

	session.RequestApproval(prompt, func(p ApprovalPrompt) UXDecision {
		return UXDecisionAlways
	})

	prompt2 := BuildApprovalPrompt("sudo rm", BlocklistPrivilege)
	result2 := session.RequestApproval(prompt2, func(p ApprovalPrompt) UXDecision {
		return UXDecisionNo
	})

	if result2.Decision != UXDecisionYes {
		t.Fatalf("expected persisted yes for same category, got %s", result2.Decision)
	}
	if !result2.Persisted {
		t.Fatal("expected persisted preference to be applied")
	}
}

func TestApprovalUX_AuditLog(t *testing.T) {
	session := NewApprovalSession(true)
	prompt := BuildApprovalPrompt("rm -rf /", BlocklistDestructive)

	session.RequestApproval(prompt, func(p ApprovalPrompt) UXDecision {
		return UXDecisionYes
	})

	log := session.GetAuditLog()
	if len(log) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(log))
	}
	if log[0].Command != "rm -rf /" {
		t.Fatalf("expected command=rm -rf /, got %s", log[0].Command)
	}
	if log[0].Category != "destructive" {
		t.Fatalf("expected category=destructive, got %s", log[0].Category)
	}
}

func TestApprovalUX_PromptFields(t *testing.T) {
	prompt := BuildApprovalPrompt("rm -rf /tmp/data", BlocklistDestructive)

	if prompt.Command != "rm -rf /tmp/data" {
		t.Fatalf("expected command, got %s", prompt.Command)
	}
	if prompt.Category != BlocklistDestructive {
		t.Fatalf("expected destructive category, got %s", prompt.Category)
	}
	if prompt.RiskLevel != "high" {
		t.Fatalf("expected high risk, got %s", prompt.RiskLevel)
	}
	if len(prompt.AffectedPaths) == 0 {
		t.Fatal("expected affected paths")
	}
}

func TestApprovalUX_DoctorReport(t *testing.T) {
	session := NewApprovalSession(true)
	session.RequestApproval(BuildApprovalPrompt("cmd", BlocklistNetwork), func(p ApprovalPrompt) UXDecision {
		return UXDecisionAlways
	})

	if !session.IsInteractive() {
		t.Fatal("expected interactive session")
	}
	if session.GetPreferenceCount() != 1 {
		t.Fatalf("expected 1 preference, got %d", session.GetPreferenceCount())
	}
}

func TestApprovalUX_AuditRecordTimestamp(t *testing.T) {
	before := time.Now()
	session := NewApprovalSession(true)
	session.RequestApproval(BuildApprovalPrompt("cmd", BlocklistNetwork), func(p ApprovalPrompt) UXDecision {
		return UXDecisionYes
	})
	after := time.Now()

	log := session.GetAuditLog()
	if len(log) != 1 {
		t.Fatal("expected 1 audit entry")
	}
	if log[0].Timestamp.Before(before) || log[0].Timestamp.After(after) {
		t.Fatal("audit timestamp out of range")
	}
}
