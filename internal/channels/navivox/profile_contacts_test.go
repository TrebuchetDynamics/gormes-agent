package navivox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestNavivoxProfileContactSnapshotIsAuthBoundedAndRedacted(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	defaultWorkspace := filepath.Join(home, "workspace-main")
	if err := os.MkdirAll(defaultWorkspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[hermes]
provider = "openrouter"
model = "openai/gpt-4o"

[agents.defaults]
workspaces = ["`+defaultWorkspace+`", "`+filepath.Join(home, "missing-secret-root")+`"]
`), 0o600); err != nil {
		t.Fatal(err)
	}
	workRoot := filepath.Join(home, "profiles", "work")
	if err := os.MkdirAll(workRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workRoot, "config.toml"), []byte(`
[agents.defaults]
workspace = "`+filepath.Join(workRoot, "workspace")+`"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannel(t)
	ch.now = func() time.Time { return time.Date(2026, 5, 18, 6, 30, 0, 0, time.UTC) }
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	unauth, err := http.Get(server.URL + "/v1/navivox/profile-contacts")
	if err != nil {
		t.Fatal(err)
	}
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized contacts status = %d, want 401", unauth.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/profile-contacts", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("contacts status = %d, want 200", resp.StatusCode)
	}
	var snapshot profileContactSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if got := profileContactIDs(snapshot.Contacts); !slices.Equal(got, []string{"default", "work"}) {
		t.Fatalf("contact IDs = %v, want default/work", got)
	}
	defaultContact := snapshot.Contacts[0]
	if defaultContact.WorkspaceRootCount != 2 || defaultContact.WorkspaceRootsWarning != 1 || defaultContact.WorkspaceRootsOK {
		t.Fatalf("default workspace summary = %+v, want count=2 warning=1 ok=false", defaultContact)
	}
	if defaultContact.Health != ProfileContactHealthWarning || !slices.Contains(defaultContact.AttentionBadges, "workspace") {
		t.Fatalf("default health/badges = %s/%v, want workspace warning", defaultContact.Health, defaultContact.AttentionBadges)
	}
	workContact := snapshot.Contacts[1]
	if workContact.Health != ProfileContactHealthNeedsAuth || !slices.Contains(workContact.AttentionBadges, "auth") {
		t.Fatalf("work health/badges = %s/%v, want auth attention", workContact.Health, workContact.AttentionBadges)
	}

	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{home, defaultWorkspace, workRoot, "missing-secret-root", "config.toml"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("profile contact snapshot leaked %q:\n%s", forbidden, raw)
		}
	}
}

func TestNavivoxProfileContactWebSocketUpdateSanitizesLatestPreview(t *testing.T) {
	ch := newTestChannel(t)
	ch.now = func() time.Time { return time.Date(2026, 5, 18, 6, 45, 0, 0, time.UTC) }
	ch.loadContacts = func(context.Context) ([]ProfileContact, error) {
		return []ProfileContact{{
			ServerID:           navivoxDefaultServerID,
			ProfileID:          "default",
			DisplayName:        "Default profile",
			ServerLabel:        navivoxDefaultServerLabel,
			AvatarSeed:         navivoxDefaultServerID + ":default",
			LatestPreview:      "Ready",
			LatestPreviewKind:  "status",
			Health:             ProfileContactHealthOnline,
			WorkspaceRootCount: 1,
			WorkspaceRootsOK:   true,
			MicAvailable:       true,
			ActiveTurnState:    ProfileContactTurnIdle,
		}}, nil
	}
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	secretText := `read /home/xel/.gormes/config.toml token=sk-secret {"api_key":"abc"}`
	if err := conn.WriteJSON(ClientMessage{
		Type:      "start_turn",
		RequestID: "req-contact",
		Text:      secretText,
		Metadata: map[string]any{
			"server_id":  navivoxDefaultServerID,
			"profile_id": "default",
		},
	}); err != nil {
		t.Fatal(err)
	}

	var update ServerEvent
	for i := 0; i < 4; i++ {
		if err := conn.ReadJSON(&update); err != nil {
			t.Fatal(err)
		}
		if update.Type == "profile_contact_update" {
			break
		}
	}
	if update.Type != "profile_contact_update" || update.Contact == nil {
		t.Fatalf("event = %+v, want profile_contact_update with contact", update)
	}
	if update.Contact.ActiveTurnState != ProfileContactTurnActive || update.Contact.LatestPreviewKind != "user" {
		t.Fatalf("contact update = %+v, want active user preview", update.Contact)
	}
	preview := update.Contact.LatestPreview
	for _, forbidden := range []string{"/home/xel", ".gormes", "config.toml", "sk-secret", "api_key", "abc"} {
		if strings.Contains(preview, forbidden) {
			t.Fatalf("preview leaked %q: %q", forbidden, preview)
		}
	}
}

func profileContactIDs(contacts []ProfileContact) []string {
	out := make([]string, 0, len(contacts))
	for _, contact := range contacts {
		out = append(out, contact.ProfileID)
	}
	return out
}
