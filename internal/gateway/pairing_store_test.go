package gateway

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPairingCodeRequestFromInboundCompatibility(t *testing.T) {
	store := NewPairingStore(filepath.Join(t.TempDir(), "pairing.json"))

	dmRequest := PairingCodeRequestFromInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "424242",
		ChatType: "private",
		UserName: "Private Chat",
	}, false)
	if dmRequest.UserID != "424242" {
		t.Fatalf("DM fallback UserID = %q, want chat ID", dmRequest.UserID)
	}

	dmCode, err := store.GeneratePairingCode(context.Background(), dmRequest)
	if err != nil {
		t.Fatalf("GeneratePairingCode(DM fallback): %v", err)
	}
	if dmCode.Status != PairingCodeIssued {
		t.Fatalf("DM fallback status = %q, want %q", dmCode.Status, PairingCodeIssued)
	}

	groupRequest := PairingCodeRequestFromInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "-100",
		ChatType: "group",
	}, false)
	if groupRequest.UserID != "" {
		t.Fatalf("group fallback UserID = %q, want empty", groupRequest.UserID)
	}
	unresolved, err := store.GeneratePairingCode(context.Background(), groupRequest)
	if err != nil {
		t.Fatalf("GeneratePairingCode(unresolved): %v", err)
	}
	if unresolved.Status != PairingCodeUnresolvedUser || unresolved.Code != "" {
		t.Fatalf("unresolved result = %#v, want unresolved-user with no code", unresolved)
	}

	denied, err := store.GeneratePairingCode(context.Background(), PairingCodeRequest{
		Platform:        "telegram",
		UserID:          "allowlist-denied",
		AllowlistDenied: true,
	})
	if err != nil {
		t.Fatalf("GeneratePairingCode(allowlist denied): %v", err)
	}
	if denied.Status != PairingCodeAllowlistDenied || denied.Code != "" {
		t.Fatalf("allowlist denied result = %#v, want allowlist-denied with no code", denied)
	}
}
