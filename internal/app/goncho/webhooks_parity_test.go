package goncho

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	gonchoservice "github.com/TrebuchetDynamics/goncho/service"
)

// TestGonchoWebhookCRUDParity proves Gormes consumes the Goncho module's
// Honcho-compatible webhook surface after Goncho moved out of this repository.
// The historical fixture lived at internal/goncho/webhooks_test.go; this test
// keeps the evidence in-repo against the pinned go.mod version.
func TestGonchoWebhookCRUDParity(t *testing.T) {
	db, err := sqlOpenGoncho(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatalf("sqlOpenGoncho: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	svc := gonchoservice.NewService(db, gonchoservice.Config{
		WorkspaceID:    "gormes",
		ObserverPeerID: "operator",
	}, nil)
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)

	created, err := svc.GetOrCreateWebhookEndpoint(ctx, gonchoservice.WebhookEndpointCreateParams{
		WorkspaceID: "gormes",
		URL:         "https://example.com/honcho-hook",
		Now:         now,
	})
	if err != nil {
		t.Fatalf("create webhook endpoint: %v", err)
	}
	if !created.Created {
		t.Fatal("created.Created = false, want first create to insert")
	}
	if created.Endpoint.WorkspaceID != "gormes" || created.Endpoint.URL != "https://example.com/honcho-hook" {
		t.Fatalf("created endpoint = %+v", created.Endpoint)
	}

	// Get-or-create is idempotent for the same workspace and URL.
	repeated, err := svc.GetOrCreateWebhookEndpoint(ctx, gonchoservice.WebhookEndpointCreateParams{
		WorkspaceID: "gormes",
		URL:         "https://example.com/honcho-hook",
		Now:         now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("repeat webhook endpoint: %v", err)
	}
	if repeated.Created || repeated.Endpoint.ID != created.Endpoint.ID {
		t.Fatalf("repeated = %+v, want existing endpoint %s", repeated, created.Endpoint.ID)
	}

	// Endpoints are workspace-scoped.
	other, err := svc.GetOrCreateWebhookEndpoint(ctx, gonchoservice.WebhookEndpointCreateParams{
		WorkspaceID: "other",
		URL:         "https://example.com/honcho-hook",
		Now:         now,
	})
	if err != nil {
		t.Fatalf("other workspace webhook endpoint: %v", err)
	}
	if other.Endpoint.WorkspaceID != "other" || other.Endpoint.ID == created.Endpoint.ID {
		t.Fatalf("other workspace endpoint = %+v, want distinct resource", other.Endpoint)
	}

	list, err := svc.ListWebhookEndpoints(ctx, "gormes")
	if err != nil {
		t.Fatalf("list webhook endpoints: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.Endpoint.ID {
		t.Fatalf("list = %+v, want only the gormes endpoint", list)
	}

	if err := svc.DeleteWebhookEndpoint(ctx, "gormes", created.Endpoint.ID); err != nil {
		t.Fatalf("delete webhook endpoint: %v", err)
	}
	list, err = svc.ListWebhookEndpoints(ctx, "gormes")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list after delete = %+v, want empty", list)
	}
}

func TestGonchoWebhookEventAndSignatureParity(t *testing.T) {
	event, err := gonchoservice.NewTestWebhookEvent("gormes")
	if err != nil {
		t.Fatalf("NewTestWebhookEvent: %v", err)
	}
	if event.WorkspaceID != "gormes" || event.Type != gonchoservice.WebhookEventTest {
		t.Fatalf("event = %+v, want gormes test event", event)
	}

	payload := `{"workspace_id":"gormes","type":"test"}`
	secret := "honcho-secret"
	got, err := gonchoservice.SignWebhookPayload(payload, secret)
	if err != nil {
		t.Fatalf("SignWebhookPayload: %v", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	want := hex.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("signature = %q, want Honcho-compatible HMAC-SHA256 %q", got, want)
	}
	if _, err := gonchoservice.SignWebhookPayload(payload, ""); err == nil {
		t.Fatal("SignWebhookPayload with empty secret succeeded, want error")
	}
}
