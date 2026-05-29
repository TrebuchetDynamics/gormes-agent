package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

func TestDoctorBrowserRuntimeRecommendsChromeInstallWhenUnavailable(t *testing.T) {
	got := doctorBrowserRuntimeStatusWithDeps(browserRuntimeDoctorDeps{
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		getenv: func(string) string { return "" },
		probeCDP: func(context.Context, string) error {
			return errors.New("unexpected probe")
		},
	})

	if got.Name != "Browser runtime" || got.Status != doctor.StatusWarn {
		t.Fatalf("browser runtime status = %+v, want Browser runtime WARN", got)
	}
	chrome, ok := findItem(got.Items, "chrome")
	if !ok {
		t.Fatalf("missing chrome item in %+v", got.Items)
	}
	for _, want := range []string{"Install Google Chrome or Chromium", "remote-debugging-port=9222"} {
		if !strings.Contains(chrome.Note, want) {
			t.Fatalf("chrome note = %q, want %q", chrome.Note, want)
		}
	}
	cdp, ok := findItem(got.Items, "cdp")
	if !ok {
		t.Fatalf("missing cdp item in %+v", got.Items)
	}
	if !strings.Contains(cdp.Note, "BROWSER_CDP_URL") || !strings.Contains(cdp.Note, "CHROME_REMOTE_DEBUGGING_URL") {
		t.Fatalf("cdp note = %q, want both CDP env aliases in guidance", cdp.Note)
	}
}

func TestDoctorBrowserRuntimeChromeLaunchCommandHonorsGormesHome(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)

	got := chromeLaunchCommand("/usr/bin/chromium")
	wantUserDataDir := "--user-data-dir=" + gormesHome + "/chrome-debug"
	if !strings.Contains(got, wantUserDataDir) {
		t.Fatalf("chrome launch command = %q, want %q", got, wantUserDataDir)
	}
	if strings.Contains(got, "$HOME/.gormes/chrome-debug") {
		t.Fatalf("chrome launch command still hard-codes HOME fallback: %q", got)
	}
}

func TestDoctorBrowserRuntimeReportsReadyWhenChromeAndCDPAreReachable(t *testing.T) {
	got := doctorBrowserRuntimeStatusWithDeps(browserRuntimeDoctorDeps{
		lookPath: func(name string) (string, error) {
			switch name {
			case "google-chrome":
				return "/usr/bin/google-chrome", nil
			default:
				return "", errors.New("not found")
			}
		},
		getenv: func(key string) string {
			if key == "CHROME_REMOTE_DEBUGGING_URL" {
				return "http://127.0.0.1:9222"
			}
			return ""
		},
		probeCDP: func(_ context.Context, endpoint string) error {
			if endpoint != "http://127.0.0.1:9222" {
				t.Fatalf("probe endpoint = %q, want http://127.0.0.1:9222", endpoint)
			}
			return nil
		},
	})

	if got.Status != doctor.StatusPass {
		t.Fatalf("Status = %v, want %v\n%s", got.Status, doctor.StatusPass, got.Format())
	}
	for _, name := range []string{"chrome", "cdp"} {
		item, ok := findItem(got.Items, name)
		if !ok {
			t.Fatalf("missing %s item in %+v", name, got.Items)
		}
		if item.Status != doctor.StatusPass {
			t.Fatalf("%s item = %+v, want PASS", name, item)
		}
	}
	if item, ok := findItem(got.Items, "harness"); ok {
		t.Fatalf("unexpected external harness item after in-process browser runtime migration: %+v", item)
	}
	if !strings.Contains(got.Summary, "local_cdp_ready") {
		t.Fatalf("Summary = %q, want local_cdp_ready", got.Summary)
	}
}

func TestDoctorBrowserRuntimeAcceptsHermesBrowserCDPURL(t *testing.T) {
	got := doctorBrowserRuntimeStatusWithDeps(browserRuntimeDoctorDeps{
		lookPath: func(name string) (string, error) {
			switch name {
			case "google-chrome":
				return "/bin/" + name, nil
			default:
				return "", errors.New("not found")
			}
		},
		getenv: func(key string) string {
			if key == "BROWSER_CDP_URL" {
				return "http://127.0.0.1:9333"
			}
			return ""
		},
		probeCDP: func(_ context.Context, endpoint string) error {
			if endpoint != "http://127.0.0.1:9333" {
				t.Fatalf("probe endpoint = %q, want BROWSER_CDP_URL", endpoint)
			}
			return nil
		},
	})

	if got.Status != doctor.StatusPass {
		t.Fatalf("Status = %v, want %v\n%s", got.Status, doctor.StatusPass, got.Format())
	}
}

func TestDoctorBrowserRuntimeWarnsWhenCDPEndpointIsConfiguredButUnreachable(t *testing.T) {
	got := doctorBrowserRuntimeStatusWithDeps(browserRuntimeDoctorDeps{
		lookPath: func(name string) (string, error) {
			switch name {
			case "google-chrome":
				return "/bin/" + name, nil
			default:
				return "", errors.New("not found")
			}
		},
		getenv: func(key string) string {
			if key == "CHROME_REMOTE_DEBUGGING_URL" {
				return "http://127.0.0.1:9222"
			}
			return ""
		},
		probeCDP: func(context.Context, string) error {
			return errors.New("connection refused")
		},
	})

	if got.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want %v\n%s", got.Status, doctor.StatusWarn, got.Format())
	}
	if !strings.Contains(got.Summary, "cdp_unreachable") {
		t.Fatalf("Summary = %q, want cdp_unreachable", got.Summary)
	}
	cdp, ok := findItem(got.Items, "cdp")
	if !ok {
		t.Fatalf("missing cdp item in %+v", got.Items)
	}
	if !strings.Contains(cdp.Note, "connection refused") || !strings.Contains(cdp.Note, "remote-debugging-port=9222") {
		t.Fatalf("cdp note = %q, want reachability error plus launch guidance", cdp.Note)
	}
}

func TestDoctorBrowserRuntimeOfflineDoesNotProbeConfiguredCDP(t *testing.T) {
	got := doctorBrowserRuntimeStatusWithDeps(browserRuntimeDoctorDeps{
		offline: true,
		lookPath: func(name string) (string, error) {
			if name == "google-chrome" {
				return "/bin/google-chrome", nil
			}
			return "", errors.New("not found")
		},
		getenv: func(key string) string {
			if key == "BROWSER_CDP_URL" {
				return "http://127.0.0.1:9333"
			}
			return ""
		},
		probeCDP: func(context.Context, string) error {
			t.Fatalf("offline browser runtime status must not probe CDP")
			return nil
		},
	})

	if got.Status != doctor.StatusPass {
		t.Fatalf("Status = %v, want PASS for local offline CDP config\n%s", got.Status, got.Format())
	}
	if !strings.Contains(got.Summary, "cdp_configured_offline") {
		t.Fatalf("Summary = %q, want cdp_configured_offline", got.Summary)
	}
	cdp, ok := findItem(got.Items, "cdp")
	if !ok {
		t.Fatalf("missing cdp item in %+v", got.Items)
	}
	if cdp.Status != doctor.StatusSkip || !strings.Contains(cdp.Note, "skipped --offline") {
		t.Fatalf("cdp item = %+v, want SKIP with offline note", cdp)
	}
}
