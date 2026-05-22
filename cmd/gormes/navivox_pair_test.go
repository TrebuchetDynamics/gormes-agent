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

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestNavivoxPairHelpExplainsOneTerminalFlow(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--help")
	if err != nil {
		t.Fatalf("navivox pair --help: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Start a local Navivox bridge, generate a pairing token, write a QR image,",
		"print the localhost URL, then wait for the Android app to connect.",
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
		"base_url":              "http://127.0.0.1:8765",
		"websocket_url":         "ws://127.0.0.1:8765/v1/navivox/stream",
		"status_url":            "http://127.0.0.1:8765/v1/navivox/status",
		"setup_handoff":         "true",
		"setup_mutation_policy": "read_only_handoff",
		"setup_sections":        "provider,model,workspace,channels",
		"setup_entry_screen":    "setup.provider",
		"recommended_path":      "navivox",
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("descriptor %s = %q, want %q in %s", key, got, want, descriptor)
		}
	}
}

func TestNavivoxPairNoWaitCreatesLocalPairingHandoff(t *testing.T) {
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
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--port", strconv.Itoa(port), "--no-wait")
	if err != nil {
		t.Fatalf("navivox pair --no-wait: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Navivox pairing ready.",
		fmt.Sprintf("Local bridge URL: http://127.0.0.1:%d", port),
		fmt.Sprintf("WebSocket URL: ws://127.0.0.1:%d/v1/navivox/stream", port),
		"Pairing token: generated and stored",
		"Pairing QR image: ",
		"Open Navivox on Android and scan the QR.",
		"Waiting for Navivox connection skipped (--no-wait).",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
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
	if strings.Contains(stdout+stderr, cfg.Navivox.Token) {
		t.Fatalf("navivox pair leaked generated token:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	qrPath := filepath.Join(home, "navivox", "pairing.png")
	info, err := os.Stat(qrPath)
	if err != nil {
		t.Fatalf("stat pairing QR image: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("pairing QR mode = %v, want 0600", got)
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
		fmt.Sprintf("Local bridge URL: http://127.0.0.1:%d", port),
		fmt.Sprintf("Local bridge listening: http://127.0.0.1:%d", port),
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
