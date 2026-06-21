package gormescli_test

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/security"
)

// TestAdvisoryStartupBannerEmitsOnCompromisedPackage proves the helper
// assembles hits, acks, and cache and returns a SECURITY ADVISORY banner when
// a compromised package version is detected (parity intent:
// hermes_cli/security_advisories.py startup_banner / short_banner_lines).
func TestAdvisoryStartupBannerEmitsOnCompromisedPackage(t *testing.T) {
	home := t.TempDir()
	installed := func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6" // the compromised PyPI release
		}
		return ""
	}
	banner := gormescli.AdvisoryStartupBanner(home, installed, time.Now())
	if !strings.Contains(banner, "SECURITY ADVISORY") {
		t.Fatalf("banner = %q, want SECURITY ADVISORY", banner)
	}
	if !strings.Contains(banner, "shai-hulud-2026-05") {
		t.Fatalf("banner = %q, want advisory ID shai-hulud-2026-05", banner)
	}
	if !strings.Contains(banner, "gormes doctor") {
		t.Fatalf("banner = %q, want gormes doctor remediation pointer", banner)
	}
}

// TestAdvisoryStartupBannerSilentOnCleanRuntime proves no banner is emitted
// when the runtime has no installed compromised packages (the normal Go-binary
// case where NoInstalledPackages is the seam).
func TestAdvisoryStartupBannerSilentOnCleanRuntime(t *testing.T) {
	home := t.TempDir()
	banner := gormescli.AdvisoryStartupBanner(home, security.NoInstalledPackages, time.Now())
	if banner != "" {
		t.Fatalf("banner = %q, want empty for clean runtime", banner)
	}
}

// TestAdvisoryStartupBannerSuppressedWhenAcked proves that an acknowledged
// advisory suppresses the banner for that ID.
func TestAdvisoryStartupBannerSuppressedWhenAcked(t *testing.T) {
	home := t.TempDir()
	installed := func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	}
	// Ack the advisory via the store directly.
	store := security.NewAckStore(home)
	if err := store.Ack("shai-hulud-2026-05"); err != nil {
		t.Fatalf("ack: %v", err)
	}
	banner := gormescli.AdvisoryStartupBanner(home, installed, time.Now())
	if banner != "" {
		t.Fatalf("banner = %q, want empty after ack", banner)
	}
}

// TestAdvisoryStartupBannerRepeatSuppression proves the repeat-window cache
// suppresses the banner within the 24h window.
func TestAdvisoryStartupBannerRepeatSuppression(t *testing.T) {
	home := t.TempDir()
	installed := func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	}
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)

	// First call: banner shown and cache stamped.
	first := gormescli.AdvisoryStartupBanner(home, installed, now)
	if !strings.Contains(first, "SECURITY ADVISORY") {
		t.Fatalf("first call: banner = %q, want SECURITY ADVISORY", first)
	}
	// Second call within 24h: suppressed.
	second := gormescli.AdvisoryStartupBanner(home, installed, now.Add(time.Hour))
	if second != "" {
		t.Fatalf("second call within 24h: banner = %q, want empty (repeat suppressed)", second)
	}
	// Third call after 25h: re-shown.
	third := gormescli.AdvisoryStartupBanner(home, installed, now.Add(25*time.Hour))
	if !strings.Contains(third, "SECURITY ADVISORY") {
		t.Fatalf("third call after 25h: banner = %q, want SECURITY ADVISORY (re-shown)", third)
	}
}
