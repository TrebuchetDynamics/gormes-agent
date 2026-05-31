package urlpolicy

import "testing"

func TestExtractHostFromURLKeepsBareIPv6Literal(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "bare_ipv6", raw: "2001:db8::1", want: "2001:db8::1"},
		{name: "bare_ipv6_metadata", raw: "fd00:ec2::254", want: "fd00:ec2::254"},
		{name: "bracketed_ipv6_port", raw: "[2001:db8::1]:8443", want: "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractHostFromURL(tt.raw); got != tt.want {
				t.Fatalf("ExtractHostFromURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
