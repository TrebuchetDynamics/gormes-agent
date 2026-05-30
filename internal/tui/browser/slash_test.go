package browser

import (
	"strings"
	"testing"
)

func TestStatusMessage(t *testing.T) {
	got := StatusMessage("")
	if !strings.Contains(got, "browser: not connected") || !strings.Contains(got, "/browser connect "+DefaultCDPURL) {
		t.Fatalf("StatusMessage(empty) = %q, want setup guidance", got)
	}
	if got := StatusMessage(" http://127.0.0.1:9333 "); got != "browser: connected http://127.0.0.1:9333" {
		t.Fatalf("StatusMessage(connected) = %q", got)
	}
}

func TestValidateCDPURL(t *testing.T) {
	for _, endpoint := range []string{"http://127.0.0.1:9222", "https://browser.example", "ws://127.0.0.1:9222", "wss://browser.example"} {
		if err := ValidateCDPURL(endpoint); err != nil {
			t.Fatalf("ValidateCDPURL(%q) = %v, want nil", endpoint, err)
		}
	}
	if err := ValidateCDPURL("file:///tmp/browser"); err == nil {
		t.Fatal("ValidateCDPURL(file) = nil, want invalid scheme")
	}
}

func TestHandleSlashStatusAndConnect(t *testing.T) {
	env := map[string]string{"CHROME_REMOTE_DEBUGGING_URL": " http://127.0.0.1:9444 "}
	getenv := func(key string) string { return env[key] }
	setenv := func(key, value string) error {
		env[key] = value
		return nil
	}

	if got := HandleSlash("/browser status", getenv, setenv); got != "browser: connected http://127.0.0.1:9444" {
		t.Fatalf("HandleSlash(status) = %q", got)
	}
	if got := HandleSlash("/browser connect ws://127.0.0.1:9222", getenv, setenv); got != "browser: connected ws://127.0.0.1:9222" {
		t.Fatalf("HandleSlash(connect) = %q", got)
	}
	if env["BROWSER_CDP_URL"] != "ws://127.0.0.1:9222" || env["CHROME_REMOTE_DEBUGGING_URL"] != "ws://127.0.0.1:9222" {
		t.Fatalf("HandleSlash(connect) env = %#v", env)
	}
	if got := HandleSlash("/browser connect file:///tmp/browser", getenv, setenv); !strings.Contains(got, "browser: invalid CDP URL") {
		t.Fatalf("HandleSlash(invalid connect) = %q", got)
	}
}
