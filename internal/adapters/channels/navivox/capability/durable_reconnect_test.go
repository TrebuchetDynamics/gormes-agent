package capability

import "testing"

func TestDurableReconnectSupportedOnSafeTransport(t *testing.T) {
	for _, security := range []string{"loopback", "tls", "https", "private_network"} {
		schema := DurableReconnect(security)
		if !schema.Supported {
			t.Fatalf("DurableReconnect(%q).Supported = false, want true", security)
		}
		if schema.IssueEndpoint != DurableReconnectIssueEndpoint ||
			schema.RevokeEndpoint != DurableReconnectRevokeEndpoint {
			t.Fatalf("DurableReconnect(%q) endpoints = issue=%q revoke=%q", security, schema.IssueEndpoint, schema.RevokeEndpoint)
		}
		if len(schema.AuthMethods) == 0 || len(schema.Scopes) == 0 || !schema.Interim {
			t.Fatalf("DurableReconnect(%q) = %+v, want non-empty interim auth/scopes", security, schema)
		}
		if schema.BlockedReason != "" {
			t.Fatalf("DurableReconnect(%q).BlockedReason = %q, want empty", security, schema.BlockedReason)
		}
	}
}

func TestDurableReconnectFailsClosedOnInsecureTransport(t *testing.T) {
	for _, security := range []string{"insecure", "", "lan"} {
		schema := DurableReconnect(security)
		if schema.Supported {
			t.Fatalf("DurableReconnect(%q).Supported = true, want fail-closed", security)
		}
		if schema.IssueEndpoint != "" || len(schema.AuthMethods) != 0 {
			t.Fatalf("DurableReconnect(%q) advertised an issue contract while unsupported: %+v", security, schema)
		}
		if schema.BlockedReason == "" {
			t.Fatalf("DurableReconnect(%q).BlockedReason empty, want a blocker", security)
		}
	}
}

func TestDurableReconnectSecurityAllowed(t *testing.T) {
	allowed := map[string]bool{
		"loopback":        true,
		"tls":             true,
		"https":           true,
		"private_network": true,
		"private-network": true,
		"insecure":        false,
		"":                false,
		"lan":             false,
	}
	for security, want := range allowed {
		if got := DurableReconnectSecurityAllowed(security); got != want {
			t.Fatalf("DurableReconnectSecurityAllowed(%q) = %v, want %v", security, got, want)
		}
	}
}
