package unauthorizeddmtest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/pairing"
)

const hermesPairingAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var storePaths = map[*pairing.PairingStore]string{}

func NewStore(t *testing.T) *pairing.PairingStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pairing.json")
	store := pairing.NewPairingStore(path)
	storePaths[store] = path
	return store
}

func AssertPairingCode(t *testing.T, code string) {
	t.Helper()
	if len(code) != 8 {
		t.Fatalf("len(%q) = %d, want %d", code, len(code), 8)
	}
	for _, c := range code {
		if !strings.ContainsRune(hermesPairingAlphabet, c) {
			t.Fatalf("code %q contains %q outside Hermes alphabet %q", code, c, hermesPairingAlphabet)
		}
	}
}

func AssertNoAuthorizedSessionLeak(t *testing.T, text string) {
	t.Helper()
	for _, leak := range []string{"allowed", "authorized", "session", "allowed-chat-42"} {
		if strings.Contains(strings.ToLower(text), leak) {
			t.Fatalf("response %q leaks authorized-session state marker %q", text, leak)
		}
	}
}

func AssertDegradedEvidence(t *testing.T, store *pairing.PairingStore, reason pairing.PairingDegradedReason, platform, userID string) {
	t.Helper()
	status, err := store.ReadPairingStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadPairingStatus: %v", err)
	}
	if len(status.Pending) != 0 || len(status.Approved) != 0 {
		t.Fatalf("status = %+v, want denied user evidence without pending or approved records", status)
	}
	for _, evidence := range status.Degraded {
		if evidence.Reason == reason && evidence.Platform == platform && evidence.UserID == userID {
			return
		}
	}
	t.Fatalf("degraded evidence = %+v, want %s for %s/%s", status.Degraded, reason, platform, userID)
}

func AssertPairingFileNotCreated(t *testing.T, store *pairing.PairingStore) {
	t.Helper()
	path := storePaths[store]
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("pairing store file err = %v, want not created", err)
	}
}
