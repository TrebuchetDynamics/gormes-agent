package ssrfguard

import "testing"

func TestBrowserSSRFGuard_CoercesQuotedFalseValues(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want bool
	}{
		{name: "double_quoted_false", raw: `"false"`, want: false},
		{name: "single_quoted_false", raw: `'false'`, want: false},
		{name: "numeric_zero", raw: 0, want: false},
		{name: "no", raw: "no", want: false},
		{name: "off", raw: "off", want: false},
		{name: "true", raw: "true", want: true},
		{name: "yes", raw: "yes", want: true},
		{name: "on", raw: "on", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoerceBool(tt.raw, true)
			if got.Value != tt.want {
				t.Fatalf("CoerceBool(%#v).Value = %v, want %v", tt.raw, got.Value, tt.want)
			}
			if got.Evidence != "" {
				t.Fatalf("CoerceBool(%#v).Evidence = %q, want empty", tt.raw, got.Evidence)
			}
		})
	}
}

func TestBrowserSSRFGuard_PrivateURLBlockedWhenCloudWouldReceiveIt(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
	}{
		{name: "localhost", rawURL: "http://localhost:3000/dashboard"},
		{name: "schemeless_localhost_with_port", rawURL: "localhost:3000/dashboard"},
		{name: "rfc1918", rawURL: "http://192.168.1.10/admin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check("task-1", tt.rawURL, Options{
				CloudConfigured:         true,
				AllowPrivateURLs:        false,
				AutoLocalForPrivateURLs: false,
			})
			if got.Allowed {
				t.Fatalf("Check(%q).Allowed = true, want false", tt.rawURL)
			}
			if got.Evidence != "private_url_blocked" {
				t.Fatalf("Check(%q).Evidence = %q, want private_url_blocked", tt.rawURL, got.Evidence)
			}
		})
	}
}

func TestBrowserSSRFGuard_PublicURLAllowed(t *testing.T) {
	got := Check("task-2", "https://example.com/docs", Options{
		CloudConfigured:         true,
		AllowPrivateURLs:        false,
		AutoLocalForPrivateURLs: false,
	})
	if !got.Allowed {
		t.Fatalf("Check(public).Allowed = false, want true")
	}
	if got.Evidence == "private_url_blocked" {
		t.Fatalf("Check(public).Evidence = private_url_blocked, want no private block evidence")
	}
}
