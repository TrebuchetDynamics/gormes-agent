package gateway

import (
	"log/slog"
	"testing"
)

func TestManagerAllowedUsersWildcardAllowsOpaquePlatformUsers(t *testing.T) {
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedUsers: map[string]map[string]bool{
			"simplex": {"*": true},
		},
	}, &fakeKernel{}, slog.Default())
	if !m.allowed(InboundEvent{Platform: "simplex", ChatID: "contact-opaque", ChatType: "dm", UserID: "contact-opaque"}) {
		t.Fatal("simplex wildcard allowed-users map did not admit opaque contact id")
	}
}
