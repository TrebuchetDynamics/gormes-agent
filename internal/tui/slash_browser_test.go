package tui

import (
	"os"
	"strings"
	"testing"
)

func TestBrowserSlashStatusReportsMissingConnection(t *testing.T) {
	t.Setenv("BROWSER_CDP_URL", "")
	t.Setenv("CHROME_REMOTE_DEBUGGING_URL", "")

	res := browserSlashHandler("/browser status", nil)
	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if !strings.Contains(res.StatusMessage, "browser: not connected") || !strings.Contains(res.StatusMessage, "/browser connect http://127.0.0.1:9222") {
		t.Fatalf("StatusMessage = %q, want setup guidance", res.StatusMessage)
	}
}

func TestBrowserSlashConnectSetsBothCDPAliases(t *testing.T) {
	t.Setenv("BROWSER_CDP_URL", "")
	t.Setenv("CHROME_REMOTE_DEBUGGING_URL", "")

	res := browserSlashHandler("/browser connect http://127.0.0.1:9333", nil)
	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "browser: connected http://127.0.0.1:9333" {
		t.Fatalf("StatusMessage = %q, want connected evidence", res.StatusMessage)
	}
	if got := os.Getenv("BROWSER_CDP_URL"); got != "http://127.0.0.1:9333" {
		t.Fatalf("BROWSER_CDP_URL = %q, want configured endpoint", got)
	}
	if got := os.Getenv("CHROME_REMOTE_DEBUGGING_URL"); got != "http://127.0.0.1:9333" {
		t.Fatalf("CHROME_REMOTE_DEBUGGING_URL = %q, want configured endpoint", got)
	}
}

func TestBrowserSlashRejectsInvalidConnectURL(t *testing.T) {
	res := browserSlashHandler("/browser connect file:///tmp/browser", nil)
	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if !strings.Contains(res.StatusMessage, "browser: invalid CDP URL") {
		t.Fatalf("StatusMessage = %q, want invalid URL evidence", res.StatusMessage)
	}
}

func TestDefaultSlashRegistryRoutesBrowser(t *testing.T) {
	t.Setenv("BROWSER_CDP_URL", "http://127.0.0.1:9222")
	res := NewDefaultSlashRegistry().Dispatch("/browser status", nil)
	if !res.Handled {
		t.Fatal("Default registry did not route /browser")
	}
	if res.StatusMessage != "browser: connected http://127.0.0.1:9222" {
		t.Fatalf("StatusMessage = %q, want connected status", res.StatusMessage)
	}
}
