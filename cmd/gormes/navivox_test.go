package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestNavivoxServeStdioHandshake(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	codec := navivox.NewCodec()
	var input bytes.Buffer
	if err := codec.WriteFrame(&input, navivox.Frame{Header: navivox.Header{
		Type:        navivox.EventHello,
		MessageID:   "hello-cli",
		Timestamp:   "2026-05-05T12:00:00Z",
		ContentType: navivox.ContentTypeJSON,
	}, Payload: []byte(`{"device":{"id":"test-client"},"supported_versions":[1]}`)}); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	cmd.SetIn(&input)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := executeRootCommand(cmd, "navivox", "serve", "--stdio"); err != nil {
		t.Fatalf("navivox serve --stdio: %v\nstderr=%s", err, stderr.String())
	}
	got, err := codec.ReadFrame(&stdout)
	if err != nil {
		t.Fatalf("read stdout frame: %v\nstderr=%s", err, stderr.String())
	}
	if got.Header.Type != navivox.EventServerStatus || got.Header.CorrelationID != "hello-cli" {
		t.Fatalf("response header = %+v, want server.status correlated to hello-cli", got.Header)
	}
}

func TestNavivoxServeRequiresStdioFlag(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	_, stderr, err := executeOneshotFlagCommand(cmd, "navivox", "serve")
	if err == nil {
		t.Fatalf("navivox serve without --stdio succeeded; stderr=%s", stderr)
	}
}

func TestNavivoxPairPrintsScannableDescriptorAndPersistsPendingPairing(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})

	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"navivox", "pair",
		"--host", "100.77.1.2",
		"--port", "2222",
		"--user", "ada",
		"--device-name", "pixel-lab",
		"--qr=false",
	)
	if err != nil {
		t.Fatalf("navivox pair: %v\nstderr=%s", err, stderr)
	}
	for _, want := range []string{
		"Navivox pairing",
		"Scan this QR from the Navivox app.",
		"Host: 100.77.1.2",
		"Port: 2222",
		"User: ada",
		"Command: gormes navivox serve --stdio",
		"Tailscale SSH is recommended",
		"Fallback URI:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("navivox pair output missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(strings.ToLower(stdout), "sudo password:") || strings.Contains(stdout, "PRIVATE KEY") {
		t.Fatalf("navivox pair output leaked credential-shaped text:\n%s", stdout)
	}

	code := regexp.MustCompile(`Pairing code: ([A-Z2-9]{8})`).FindStringSubmatch(stdout)
	if code == nil {
		t.Fatalf("navivox pair output missing bounded pairing code:\n%s", stdout)
	}
	uriText := regexp.MustCompile(`(?m)^navivox://pair\?[^\n]+$`).FindString(stdout)
	if uriText == "" {
		t.Fatalf("navivox pair output missing fallback URI:\n%s", stdout)
	}
	parsed, err := url.Parse(uriText)
	if err != nil {
		t.Fatalf("parse pairing URI %q: %v", uriText, err)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"transport": "ssh",
		"host":      "100.77.1.2",
		"port":      "2222",
		"user":      "ada",
		"command":   "gormes navivox serve --stdio",
		"protocol":  "1",
		"code":      code[1],
		"device":    "pixel-lab",
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("pairing URI %s = %q, want %q in %s", key, got, want, uriText)
		}
	}

	status, err := gateway.NewXDGPairingStore().ReadPairingStatus(context.Background())
	if err != nil {
		t.Fatalf("read pairing status: %v", err)
	}
	if len(status.Pending) != 1 || status.Pending[0].Platform != navivox.PlatformName || status.Pending[0].Code != code[1] || status.Pending[0].UserID != "pixel-lab" {
		t.Fatalf("pending pairing = %+v, want navivox/pixel-lab/%s", status.Pending, code[1])
	}
}

// TestNavivoxPairCommand_JSONEmitsStructuredPairingDescriptor proves
// `gormes navivox pair --json` returns a parseable
// `{build, host, port, user, command, protocol, code, device, expires_at,
// uri, host_source}` document so fleet automation provisioning Navivox
// pairing across machines can ingest the descriptor without scraping the
// "Host: / Port: / Pairing code:" prose. Build provenance leads — same
// convention as the rest of the `--json` arc. The pairing code is the
// data being conveyed (one-time gateway-issued credential), so it is in
// the document by design; long-lived secrets like SSH keys remain
// excluded.
func TestNavivoxPairCommand_JSONEmitsStructuredPairingDescriptor(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})

	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"navivox", "pair",
		"--host", "100.77.1.2",
		"--port", "2222",
		"--user", "ada",
		"--device-name", "pixel-lab",
		"--qr=false",
		"--json",
	)
	if err != nil {
		t.Fatalf("navivox pair --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Host       string `json:"host"`
		Port       int    `json:"port"`
		User       string `json:"user"`
		Command    string `json:"command"`
		Protocol   uint32 `json:"protocol"`
		Code       string `json:"code"`
		Device     string `json:"device"`
		ExpiresAt  string `json:"expires_at"`
		URI        string `json:"uri"`
		HostSource string `json:"host_source"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("navivox pair --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Host != "100.77.1.2" {
		t.Errorf("host = %q, want 100.77.1.2", got.Host)
	}
	if got.Port != 2222 {
		t.Errorf("port = %d, want 2222", got.Port)
	}
	if got.User != "ada" {
		t.Errorf("user = %q, want ada", got.User)
	}
	if got.Device != "pixel-lab" {
		t.Errorf("device = %q, want pixel-lab", got.Device)
	}
	if !regexp.MustCompile(`^[A-Z2-9]{8}$`).MatchString(got.Code) {
		t.Errorf("code = %q, want 8-char base32-ish bounded code", got.Code)
	}
	if !strings.HasPrefix(got.URI, "navivox://pair?") {
		t.Errorf("uri = %q, want navivox://pair? prefix", got.URI)
	}
	if got.Command != "gormes navivox serve --stdio" {
		t.Errorf("command = %q, want default serve command", got.Command)
	}
	// Pending pairing must still be persisted — JSON emit must not skip
	// the gateway side effect.
	status, err := gateway.NewXDGPairingStore().ReadPairingStatus(context.Background())
	if err != nil {
		t.Fatalf("read pairing status: %v", err)
	}
	if len(status.Pending) != 1 || status.Pending[0].Code != got.Code {
		t.Fatalf("pending pairing = %+v, want one with code %s", status.Pending, got.Code)
	}
}

func TestNavivoxSetupHostPlanPrioritizesTailscaleAndSudoSafety(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "navivox", "setup-host", "--plan")
	if err != nil {
		t.Fatalf("navivox setup-host --plan: %v\nstderr=%s", err, stderr)
	}
	for _, want := range []string{
		"Navivox host setup plan",
		"Tailscale is the recommended network path.",
		"curl -fsSL https://tailscale.com/install.sh | sh",
		"sudo tailscale up --ssh",
		"openssh-server",
		"systemctl enable --now ssh",
		"sudo password is prompt-only and never stored",
		"gormes navivox pair",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("setup-host plan missing %q:\n%s", want, stdout)
		}
	}
}
