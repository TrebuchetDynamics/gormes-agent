package doctorbrowser

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

func TestRuntimeStatusUsesHermesCDPEnvAlias(t *testing.T) {
	got := RuntimeStatusWithDeps(RuntimeDoctorDeps{
		LookPath: func(name string) (string, error) { return "/bin/" + name, nil },
		Getenv: func(key string) string {
			if key == "BROWSER_CDP_URL" {
				return "http://127.0.0.1:9333"
			}
			return ""
		},
		ProbeCDP: func(_ context.Context, endpoint string) error {
			if endpoint != "http://127.0.0.1:9333" {
				t.Fatalf("endpoint = %q", endpoint)
			}
			return nil
		},
	})
	if got.Status != doctor.StatusPass || !strings.Contains(got.Summary, "local_cdp_ready") {
		t.Fatalf("status = %+v", got)
	}
}

func TestRuntimeStatusRecommendsChromeWhenUnavailable(t *testing.T) {
	got := RuntimeStatusWithDeps(RuntimeDoctorDeps{
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
		Getenv:   func(string) string { return "" },
		ProbeCDP: func(context.Context, string) error { return errors.New("unexpected") },
	})
	if got.Status != doctor.StatusWarn || got.Summary != "chrome_unavailable" {
		t.Fatalf("status = %+v", got)
	}
	if !strings.Contains(got.Format(), "BROWSER_CDP_URL") || !strings.Contains(got.Format(), "remote-debugging-port=9222") {
		t.Fatalf("missing setup guidance:\n%s", got.Format())
	}
}
