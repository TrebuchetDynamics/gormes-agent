package gateway

import (
	"context"
	"path/filepath"
	"testing"
)

func TestTokenLockEvidencePreservesRuntimeStatusExitReason(t *testing.T) {
	evidence := TokenLockEvidence{
		Status:  TokenLockStatusReleaseFailed,
		Message: "unlink denied",
	}
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	if err := statusStore.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		GatewayState: GatewayStateStartupFailed,
		ExitReason:   "original gateway exit",
	}); err != nil {
		t.Fatalf("write original exit: %v", err)
	}
	if err := statusStore.UpdateRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		TokenLockEvidence: &evidence,
	}); err != nil {
		t.Fatalf("write lock evidence: %v", err)
	}
	status, err := statusStore.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.ExitReason != "original gateway exit" {
		t.Fatalf("ExitReason = %q, want original gateway exit preserved", status.ExitReason)
	}
	if len(status.TokenLocks) != 1 || status.TokenLocks[0].Status != TokenLockStatusReleaseFailed {
		t.Fatalf("TokenLocks = %+v, want release-failed evidence", status.TokenLocks)
	}
}
