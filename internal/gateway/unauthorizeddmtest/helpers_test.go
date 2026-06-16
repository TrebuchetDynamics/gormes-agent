package unauthorizeddmtest

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/pairing"
)

func TestNewStoreAndPairingAssertions(t *testing.T) {
	store := NewStore(t)
	AssertPairingFileNotCreated(t, store)

	result, err := store.GeneratePairingCode(context.Background(), pairing.PairingCodeRequest{Platform: "telegram", UserID: "u1"})
	if err != nil {
		t.Fatalf("GeneratePairingCode: %v", err)
	}
	AssertPairingCode(t, result.Code)
}

func TestAssertDegradedEvidence(t *testing.T) {
	store := NewStore(t)
	_, err := store.GeneratePairingCode(context.Background(), pairing.PairingCodeRequest{
		Platform:        "telegram",
		UserID:          "u1",
		AllowlistDenied: true,
	})
	if err != nil {
		t.Fatalf("GeneratePairingCode: %v", err)
	}
	AssertDegradedEvidence(t, store, pairing.PairingDegradedAllowlistDenied, "telegram", "u1")
}

func TestAssertNoAuthorizedSessionLeakAllowsNeutralText(t *testing.T) {
	AssertNoAuthorizedSessionLeak(t, "Pairing required. Ask the operator for access.")
}
