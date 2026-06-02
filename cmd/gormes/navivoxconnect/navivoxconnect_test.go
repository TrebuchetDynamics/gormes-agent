package navivoxconnect

import "testing"

func TestServerBindHostPortResolvesDefaultsAndOverrides(t *testing.T) {
	tests := []struct {
		name     string
		bind     string
		wantHost string
		wantPort int
	}{
		{name: "empty", wantHost: "127.0.0.1", wantPort: 8765},
		{name: "host port", bind: "0.0.0.0:8787", wantHost: "0.0.0.0", wantPort: 8787},
		{name: "host only", bind: "100.64.1.2", wantHost: "100.64.1.2", wantPort: 8765},
		{name: "ipv6 brackets", bind: "[::1]", wantHost: "::1", wantPort: 8765},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotPort := ServerBindHostPort(tt.bind, "127.0.0.1", 8765)
			if gotHost != tt.wantHost || gotPort != tt.wantPort {
				t.Fatalf("ServerBindHostPort(%q) = %q, %d; want %q, %d", tt.bind, gotHost, gotPort, tt.wantHost, tt.wantPort)
			}
		})
	}
}

func TestURLsBuildsBaseAndStreamURLs(t *testing.T) {
	base, stream := URLs("127.0.0.1", 8765)
	if base != "http://127.0.0.1:8765" {
		t.Fatalf("base = %q", base)
	}
	if stream != "ws://127.0.0.1:8765/v1/navivox/stream" {
		t.Fatalf("stream = %q", stream)
	}
}
