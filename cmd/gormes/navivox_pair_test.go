package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

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
