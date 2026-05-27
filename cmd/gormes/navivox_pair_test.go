package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

func TestNavivoxPairHelpExplainsOneTerminalFlow(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--help")
	if err != nil {
		t.Fatalf("navivox pair --help: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Start a network-reachable Navivox bridge, generate or reuse a pairing token,",
		"print the token for manual entry, open Navivox directly on",
		"Use this after the installer recommends Navivox setup:",
		"gormes navivox pair",
		"Keep the Termux session open after Navivox connects; it owns the local bridge.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help output missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "Hermes") {
		t.Fatalf("navivox pair help should not mention Hermes:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestNavivoxPairDescriptorIncludesSetupContinuationHints(t *testing.T) {
	descriptor := navivoxPairDescriptor(config.NavivoxCfg{
		AuthMode:     config.NavivoxAuthPairingToken,
		ExposureMode: config.NavivoxExposureLocal,
		Token:        "nvbx_test_token",
	}, "http://127.0.0.1:8765", "ws://127.0.0.1:8765/v1/navivox/stream")
	parsed, err := url.Parse(descriptor)
	if err != nil {
		t.Fatalf("parse descriptor: %v", err)
	}
	if parsed.Scheme != "navivox" || parsed.Host != "connect" {
		t.Fatalf("descriptor target = %s://%s, want navivox://connect", parsed.Scheme, parsed.Host)
	}
	values := parsed.Query()
	for key, want := range map[string]string{
		"base_url":                  "http://127.0.0.1:8765",
		"websocket_url":             "ws://127.0.0.1:8765/v1/navivox/stream",
		"status_url":                "http://127.0.0.1:8765/v1/navivox/status",
		"capabilities_url":          "http://127.0.0.1:8765/v1/navivox/capabilities",
		"setup_handoff":             "true",
		"setup_mutation_policy":     "read_only_handoff",
		"setup_sections":            "provider,model,workspace,channels",
		"setup_entry_screen":        "setup.provider",
		"bridge_keepalive_required": "true",
		"bridge_lifecycle":          "termux_pair_command",
		"recommended_path":          "navivox",
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("descriptor %s = %q, want %q in %s", key, got, want, descriptor)
		}
	}
}

func TestNavivoxPairAutoTargetPrefersTailscaleNetworkIP(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "wg0", Kind: vpnhost.KindWireGuard, IPv4: "10.0.0.4"},
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
		}, nil
	}

	target, err := resolveNavivoxPairTarget(context.Background(), "")
	if err != nil {
		t.Fatalf("resolveNavivoxPairTarget: %v", err)
	}
	if target.Host != "100.64.1.2" || target.ExposureMode != config.NavivoxExposureTailscale || target.Source != "tailscale auto-detected" {
		t.Fatalf("target = %+v, want tailscale network IP", target)
	}
}

func TestNavivoxPairNoWaitCreatesLocalPairingHandoff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("COLUMNS", "120")
	previousToken, hadPreviousToken := os.LookupEnv("GORMES_NAVIVOX_TOKEN")
	if err := os.Unsetenv("GORMES_NAVIVOX_TOKEN"); err != nil {
		t.Fatalf("unset GORMES_NAVIVOX_TOKEN: %v", err)
	}
	t.Cleanup(func() {
		if hadPreviousToken {
			_ = os.Setenv("GORMES_NAVIVOX_TOKEN", previousToken)
		} else {
			_ = os.Unsetenv("GORMES_NAVIVOX_TOKEN")
		}
	})

	port := freeLocalTCPPort(t)
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--no-wait")
	if err != nil {
		t.Fatalf("navivox pair --no-wait: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	qrPath := filepath.Join(home, "navivox", "pairing.png")
	for _, want := range []string{
		"Navivox pairing ready.",
		fmt.Sprintf("  Bridge: http://127.0.0.1:%d", port),
		fmt.Sprintf("  Stream: ws://127.0.0.1:%d/v1/navivox/stream", port),
		"  Network: operator override",
		"  Handoff: QR fallback saved:\n    " + qrPath,
		"  Scan this QR from Navivox:",
		"  QR payload includes the network bridge URL and pairing token.",
		"  Manual token is printed above for fallback entry.",
		"  Secret: QR embeds the network bridge URL and Navivox token.",
		"  Token source: generated and stored in:\n  " + config.EnvPath(),
		"  Treat token/QR like WhatsApp Web:",
		"  anyone with it can connect while this bridge is online.",
		"  Keep this terminal open for this bridge.",
		"  Text prompt for Navivox manual registration:",
		"    Connect Navivox to this Gormes bridge:",
		"Base URL http://127.0.0.1:",
		"WebSocket ws://127.0.0.1:",
		"Auth mode pairing_token",
		"Token ",
		"Waiting for Navivox connection skipped (--no-wait).",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, removed := range []string{"Next steps", "scan the QR image", "Connection\n", "Pairing\n", "Local bridge listening"} {
		if strings.Contains(stdout, removed) {
			t.Fatalf("stdout still contains noisy pair output %q:\n%s", removed, stdout)
		}
	}
	if !strings.ContainsAny(stdout, "▀▄█") {
		t.Fatalf("terminal QR block missing from pair output:\n%s", stdout)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Navivox.Enabled || cfg.Navivox.BindHost != "127.0.0.1" || cfg.Navivox.Port != port || cfg.Navivox.AuthMode != config.NavivoxAuthPairingToken {
		t.Fatalf("navivox config = %+v, want local pairing_token config", cfg.Navivox)
	}
	if cfg.Navivox.Token == "" {
		t.Fatal("navivox token was not generated")
	}
	if !strings.Contains(stdout, "  Token: "+cfg.Navivox.Token) {
		t.Fatalf("navivox pair should print generated token for manual entry:\nstdout=%s", stdout)
	}
	if strings.Contains(stderr, cfg.Navivox.Token) {
		t.Fatalf("navivox pair leaked generated token to stderr:\nstderr=%s", stderr)
	}

	info, err := os.Stat(qrPath)
	if err != nil {
		t.Fatalf("stat pairing QR image: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("pairing QR mode = %v, want 0600", got)
	}
}

func TestNavivoxPairReusesLiveGatewayRuntimeInsteadOfRefusing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	previous := newNavivoxPairRuntimeStore
	t.Cleanup(func() { newNavivoxPairRuntimeStore = previous })
	newNavivoxPairRuntimeStore = func(string) navivoxPairRuntimeStore {
		return fakeNavivoxPairRuntimeStore{snapshot: gateway.RuntimeStatusSnapshot{
			Status: gateway.RuntimeStatus{PID: 1234},
			Validation: gateway.RuntimeProcessValidation{
				Live: true,
				PID:  1234,
			},
		}}
	}

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	err := ensureNoLiveGatewayForNavivoxPair(cmd.Context())
	if err != nil {
		t.Fatalf("ensureNoLiveGatewayForNavivoxPair returned error for reusable live gateway: %v", err)
	}
}

type fakeNavivoxPairRuntimeStore struct {
	snapshot gateway.RuntimeStatusSnapshot
	err      error
}

func (s fakeNavivoxPairRuntimeStore) ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error) {
	return s.snapshot, s.err
}

func TestNavivoxPairAutoPortsWhenDefaultPortBusy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("COLUMNS", "120")
	previousToken, hadPreviousToken := os.LookupEnv("GORMES_NAVIVOX_TOKEN")
	if err := os.Unsetenv("GORMES_NAVIVOX_TOKEN"); err != nil {
		t.Fatalf("unset GORMES_NAVIVOX_TOKEN: %v", err)
	}
	t.Cleanup(func() {
		if hadPreviousToken {
			_ = os.Setenv("GORMES_NAVIVOX_TOKEN", previousToken)
		} else {
			_ = os.Unsetenv("GORMES_NAVIVOX_TOKEN")
		}
	})

	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy local port: %v", err)
	}
	defer occupied.Close()
	occupiedPort := occupied.Addr().(*net.TCPAddr).Port

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := runNavivoxPair(cmd, navivoxPairOptions{host: "127.0.0.1", port: occupiedPort, noWait: true}); err != nil {
		t.Fatalf("runNavivoxPair auto-port fallback: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, fmt.Sprintf("port %d busy, using", occupiedPort)) {
		t.Fatalf("stdout missing auto-port fallback evidence for occupied port %d:\n%s", occupiedPort, out)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Navivox.Port == occupiedPort || cfg.Navivox.Port == 0 {
		t.Fatalf("persisted navivox port = %d, want fallback port different from occupied %d", cfg.Navivox.Port, occupiedPort)
	}
	if !strings.Contains(out, fmt.Sprintf("Bridge: http://127.0.0.1:%d", cfg.Navivox.Port)) {
		t.Fatalf("stdout bridge did not use persisted fallback port %d:\n%s", cfg.Navivox.Port, out)
	}
}

func TestNavivoxPairNarrowTermuxFallsBackToPNGQRCode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_narrow_termux_token")
	t.Setenv("COLUMNS", "48")

	port := freeLocalTCPPort(t)
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--no-wait")
	if err != nil {
		t.Fatalf("navivox pair --no-wait narrow: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	qrPath := filepath.Join(home, "navivox", "pairing.png")
	for _, want := range []string{
		"  Scan this QR from Navivox:",
		"  Terminal QR: not printed; terminal is too narrow.",
		"  Detected columns: 48; QR needs ",
		"  Termux tip: rotate phone, reduce font size, or open the PNG:",
		"  termux-open " + qrPath,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.ContainsAny(stdout, "▀▄█") {
		t.Fatalf("narrow-terminal fallback should not print a wrapped QR block:\n%s", stdout)
	}
	if !strings.Contains(stdout, "  Token: nvbx_narrow_termux_token") {
		t.Fatalf("narrow-terminal fallback should print token for manual entry:\nstdout=%s", stdout)
	}
	if strings.Contains(stderr, "nvbx_narrow_termux_token") || strings.Contains(stdout+stderr, "rest_token=") {
		t.Fatalf("narrow-terminal fallback leaked descriptor token material:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestNavivoxPairWaitStartsLocalBridgeUntilContextCanceled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	previousToken, hadPreviousToken := os.LookupEnv("GORMES_NAVIVOX_TOKEN")
	if err := os.Unsetenv("GORMES_NAVIVOX_TOKEN"); err != nil {
		t.Fatalf("unset GORMES_NAVIVOX_TOKEN: %v", err)
	}
	t.Cleanup(func() {
		if hadPreviousToken {
			_ = os.Setenv("GORMES_NAVIVOX_TOKEN", previousToken)
		} else {
			_ = os.Unsetenv("GORMES_NAVIVOX_TOKEN")
		}
	})

	port := freeLocalTCPPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := newRootCommandWithRuntime(rootRuntime{})
	cmd.SetContext(ctx)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- executeRootCommand(cmd, "navivox", "pair", "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	}()

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	if err := waitForNavivoxPairHealth(t, healthURL); err != nil {
		cancel()
		<-errCh
		t.Fatalf("navivox pair did not start local bridge at %s: %v\nstdout=%s\nstderr=%s", healthURL, err, stdout.String(), stderr.String())
	}
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("navivox pair after cancel: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("navivox pair did not exit after context cancellation\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		fmt.Sprintf("Bridge: http://127.0.0.1:%d", port),
		"Keep this terminal open for this bridge.",
		"Waiting for Navivox connection... Press Ctrl-C to stop.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestNavivoxPairPrintsConnectedWhenNavivoxStreams(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	previousToken, hadPreviousToken := os.LookupEnv("GORMES_NAVIVOX_TOKEN")
	if err := os.Unsetenv("GORMES_NAVIVOX_TOKEN"); err != nil {
		t.Fatalf("unset GORMES_NAVIVOX_TOKEN: %v", err)
	}
	t.Cleanup(func() {
		if hadPreviousToken {
			_ = os.Setenv("GORMES_NAVIVOX_TOKEN", previousToken)
		} else {
			_ = os.Unsetenv("GORMES_NAVIVOX_TOKEN")
		}
	})

	port := freeLocalTCPPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := newRootCommandWithRuntime(rootRuntime{})
	cmd.SetContext(ctx)
	var stdout, stderr syncBuffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- executeRootCommand(cmd, "navivox", "pair", "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	}()

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	if err := waitForNavivoxPairHealth(t, healthURL); err != nil {
		cancel()
		<-errCh
		t.Fatalf("navivox pair did not start local bridge at %s: %v\nstdout=%s\nstderr=%s", healthURL, err, stdout.String(), stderr.String())
	}
	if err := waitForOutputContains(&stdout, "Navivox pairing ready."); err != nil {
		cancel()
		<-errCh
		t.Fatalf("navivox pair started bridge before printing persisted handoff: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	cfg, err := config.Load(nil)
	if err != nil {
		cancel()
		<-errCh
		t.Fatalf("load generated navivox config: %v", err)
	}
	conn := dialNavivoxPairWebSocket(t, fmt.Sprintf("ws://127.0.0.1:%d/v1/navivox/stream", port), cfg.Navivox.Token)
	defer conn.Close()

	if err := waitForOutputContains(&stdout, "Navivox connected. Continue setup in Navivox."); err != nil {
		cancel()
		<-errCh
		t.Fatalf("navivox pair did not report app connection: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	expectedBridgeLine := fmt.Sprintf("Local bridge remains online: http://127.0.0.1:%d", port)
	if err := waitForOutputContains(&stdout, expectedBridgeLine); err != nil {
		cancel()
		<-errCh
		t.Fatalf("navivox pair did not report persistent bridge URL: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	select {
	case err := <-errCh:
		t.Fatalf("navivox pair exited after app connection before cancellation: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	case <-time.After(100 * time.Millisecond):
	}
	_ = conn.Close()
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("navivox pair after cancel: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("navivox pair did not exit after context cancellation\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func dialNavivoxPairWebSocket(t *testing.T, wsURL, token string) *websocket.Conn {
	t.Helper()
	if token == "" {
		t.Fatal("generated navivox token is empty")
	}
	dialer := websocket.Dialer{Subprotocols: []string{
		"navivox.v1",
		"gormes.navivox.token." + base64.RawURLEncoding.EncodeToString([]byte(token)),
	}}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("websocket dial status=%d err=%v", resp.StatusCode, err)
		}
		t.Fatal(err)
	}
	return conn
}

func waitForOutputContains(buf *syncBuffer, want string) error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), want) {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %q", want)
}

func freeLocalTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitForNavivoxPairHealth(t *testing.T, healthURL string) error {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err != nil {
			lastErr = err
			time.Sleep(25 * time.Millisecond)
			continue
		}
		var payload map[string]any
		decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK && decodeErr == nil && payload["platform"] == "navivox" && payload["status"] == "ok" {
			return nil
		}
		lastErr = fmt.Errorf("status=%d payload=%v decode=%v", resp.StatusCode, payload, decodeErr)
		time.Sleep(25 * time.Millisecond)
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("timed out waiting for health response")
}
