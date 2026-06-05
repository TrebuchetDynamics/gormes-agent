package adaptertest

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestContainsString(t *testing.T) {
	if !ContainsString([]string{"alpha", "beta"}, "beta") {
		t.Fatal("ContainsString returned false for existing value")
	}
	if ContainsString([]string{"alpha", "beta"}, "gamma") {
		t.Fatal("ContainsString returned true for missing value")
	}
}

func TestApprovalRecorderRecordsSnapshotAndError(t *testing.T) {
	wantErr := errors.New("resolver unavailable")
	recorder := &ApprovalRecorder{Err: wantErr}
	resolution := gateway.ApprovalResolution{SessionKey: "session-1", Choice: gateway.ApprovalChoiceOnce}

	if err := recorder.ResolveGatewayApproval(context.Background(), resolution); !errors.Is(err, wantErr) {
		t.Fatalf("ResolveGatewayApproval error = %v, want %v", err, wantErr)
	}

	snapshot := recorder.Snapshot()
	if len(snapshot) != 1 || snapshot[0].SessionKey != resolution.SessionKey || snapshot[0].Choice != resolution.Choice {
		t.Fatalf("Snapshot = %+v, want [%+v]", snapshot, resolution)
	}
	snapshot[0].SessionKey = "mutated"
	if got := recorder.Snapshot()[0].SessionKey; got != "session-1" {
		t.Fatalf("Snapshot exposed recorder storage, got session key %q", got)
	}
}
