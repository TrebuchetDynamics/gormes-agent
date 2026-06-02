package navivoxqr

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCommandForPlatformKeepsPlainPathsStable(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		goos    string
		android bool
		want    string
	}{
		{name: "android", path: "/tmp/pairing.png", goos: "linux", android: true, want: "termux-open /tmp/pairing.png"},
		{name: "linux", path: "/tmp/pairing.png", goos: "linux", want: "xdg-open /tmp/pairing.png"},
		{name: "darwin", path: "/tmp/pairing.png", goos: "darwin", want: "open /tmp/pairing.png"},
		{name: "windows", path: `C:\Temp\pairing.png`, goos: "windows", want: `start "C:\Temp\pairing.png"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OpenCommandForPlatform(tt.path, tt.goos, tt.android); got != tt.want {
				t.Fatalf("OpenCommandForPlatform() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompactPairDescriptorContainsImportFields(t *testing.T) {
	descriptor := CompactPairDescriptor("pairing_token", "local", "nvbx_qr", "http://127.0.0.1:8765", "ws://127.0.0.1:8765/v1/navivox/stream")
	parsed, err := url.Parse(descriptor)
	if err != nil {
		t.Fatalf("parse descriptor: %v", err)
	}
	values := parsed.Query()
	for key, want := range map[string]string{
		"base_url":         "http://127.0.0.1:8765",
		"websocket_url":    "ws://127.0.0.1:8765/v1/navivox/stream",
		"capabilities_url": "http://127.0.0.1:8765/v1/navivox/capabilities",
		"auth_mode":        "pairing_token",
		"exposure_mode":    "local",
		"rest_token":       "nvbx_qr",
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestWritePNGCreatesPrivateQRCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "navivox", "pairing.png")
	if err := WritePNG(path, "navivox://connect?base_url=http://127.0.0.1:8765&rest_token=nvbx_qr"); err != nil {
		t.Fatalf("WritePNG: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat QR: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %v, want 0600", got)
	}
}

func TestForColumnsReportsTooNarrow(t *testing.T) {
	qr, err := ForColumns("navivox://connect?base_url=http://127.0.0.1:8765&rest_token=nvbx_qr", 1)
	if err != nil {
		t.Fatalf("ForColumns: %v", err)
	}
	if !qr.TooNarrow || qr.RequiredWidth <= 1 || qr.Text != "" {
		t.Fatalf("qr = %+v, want too narrow without text", qr)
	}
}
