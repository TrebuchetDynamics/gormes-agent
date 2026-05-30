package channels

import (
	"reflect"
	"strings"
	"testing"
)

func TestTeamsValueObjectDefaultsAllowedUsersAndRedactedStatus(t *testing.T) {
	cfg := TeamsCfg{
		Enabled:      true,
		ClientID:     "cfg-client",
		ClientSecret: "cfg-secret",
		TenantID:     "cfg-tenant",
		AllowedUsers: []string{" aad-1 ", "", "aad-2"},
	}
	if cfg.EffectivePort() != TeamsDefaultPort {
		t.Fatalf("EffectivePort = %d, want default %d", cfg.EffectivePort(), TeamsDefaultPort)
	}
	if !reflect.DeepEqual(cfg.AllowedUserIDs(), []string{"aad-1", "aad-2"}) {
		t.Fatalf("AllowedUserIDs = %v", cfg.AllowedUserIDs())
	}
	if status := cfg.RedactedStatus(); strings.Contains(status, "cfg-secret") || !strings.Contains(status, "configured") {
		t.Fatalf("RedactedStatus = %q", status)
	}
}

func TestTeamsValueObjectReportsMissingCredentials(t *testing.T) {
	cfg := TeamsCfg{Port: 5000, AllowAllUsers: true}
	missing := cfg.MissingCredentials()
	if !reflect.DeepEqual(missing, []string{"client_id", "client_secret", "tenant_id"}) {
		t.Fatalf("MissingCredentials = %v", missing)
	}
	status := cfg.RedactedStatus()
	for _, want := range []string{"missing_credentials=client_id,client_secret,tenant_id", "port=5000", "allow_all_users=true"} {
		if !strings.Contains(status, want) {
			t.Fatalf("RedactedStatus = %q, missing %q", status, want)
		}
	}
}
