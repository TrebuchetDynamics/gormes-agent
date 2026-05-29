package apiserver

import (
	"strings"
	"testing"
)

func TestAPIServerWeakKeyGuard_RefusesNetworkExposedPlaceholders(t *testing.T) {
	for _, host := range []string{"0.0.0.0", "::", "[::]", "192.168.1.25", "api.example.test"} {
		t.Run(host, func(t *testing.T) {
			_, err := NewServerChecked(Config{
				APIKey:             " placeholder ",
				DashboardBoundHost: host,
			})
			if err == nil {
				t.Fatalf("NewServerChecked() error = nil, want weak key rejection")
			}
			if !strings.Contains(err.Error(), "api_server_weak_key") {
				t.Fatalf("error = %q, want api_server_weak_key", err)
			}
			if strings.Contains(err.Error(), "placeholder") {
				t.Fatalf("error leaked placeholder key: %q", err)
			}
		})
	}
}

func TestAPIServerWeakKeyGuard_AllowsLoopbackDevelopment(t *testing.T) {
	for _, host := range []string{"", "127.0.0.1", "localhost", "::1", "[::1]"} {
		t.Run(host, func(t *testing.T) {
			if _, err := NewServerChecked(Config{
				APIKey:             " placeholder ",
				DashboardBoundHost: host,
			}); err != nil {
				t.Fatalf("NewServerChecked() error = %v, want loopback development allowed", err)
			}
		})
	}
}
