package main

import (
	"bytes"
	"context"
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
