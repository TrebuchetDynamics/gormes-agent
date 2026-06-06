package commandpolicy

import "testing"

func TestFacadePreservesPublicCommandPolicyAPI(t *testing.T) {
	classifier := NewCommandClassifier()
	if got := classifier.Classify("git status --short"); got != CommandSafe {
		t.Fatalf("Classify(git status) = %s, want safe", got)
	}

	if matched, desc := DetectHardline("rm -rf /"); !matched || desc == "" {
		t.Fatalf("DetectHardline(root rm) = (%v, %q), want match with description", matched, desc)
	}

	guarded := GuardCommand("rm -rf /tmp/gormes-danger", "manual")
	if guarded.Approved || !guarded.ApprovalRequired || guarded.Description == "" {
		t.Fatalf("GuardCommand(recoverable rm) = %#v, want approval-required block", guarded)
	}
}
