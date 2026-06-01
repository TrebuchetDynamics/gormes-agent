package allowlist

import "testing"

func TestGatewayAdmissionAllowlistMissingEvidence(t *testing.T) {
	got := CheckStartupAllowlist(StartupAdmissionInput{})
	if len(got) != 1 {
		t.Fatalf("evidence count = %d, want 1: %#v", len(got), got)
	}
	if got[0].Code != "gateway_allowlist_missing" {
		t.Fatalf("Code = %q, want gateway_allowlist_missing", got[0].Code)
	}
}

func TestGatewayAdmissionAllowlistConfiguredOrAllowAllSuppressesWarning(t *testing.T) {
	cases := []StartupAdmissionInput{
		{AllowlistConfigured: true},
		{AllowAll: true},
	}
	for _, tc := range cases {
		if got := CheckStartupAllowlist(tc); len(got) != 0 {
			t.Fatalf("CheckStartupAllowlist(%#v) = %#v, want no evidence", tc, got)
		}
	}
}
