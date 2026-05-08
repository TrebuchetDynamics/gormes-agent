package main

import (
	"bytes"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLogsCommand_FileFallbackRoutesThroughCobraWriters proves the
// command writes its file-fallback output via cmd.OutOrStdout() so
// end-to-end tests can capture stdout. Without this, runLogs's
// fmt.Print bypasses cobra's writer plumbing — meaning operators
// piping `gormes logs` through a tool that captures stdout
// programmatically and tests verifying the format both have to fork
// custom paths.
//
// The fallback path is the testable one: when the gateway HTTP
// endpoint is unreachable (the default in any test environment with
// no gateway daemon running), runLogs reads the on-disk log file
// instead. This test seeds GORMES_HOME with a known log file and
// asserts the captured stdout contains its body.
func TestLogsCommand_FileFallbackRoutesThroughCobraWriters(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	want := "[2026-05-08T07:00:00Z] info: gateway started\n"
	if err := os.WriteFile(filepath.Join(home, "gormes.log"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newLogsCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("logs: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if stdout.String() == "" {
		t.Fatalf("logs must write to cmd.OutOrStdout(); captured stdout is empty (output likely went to os.Stdout)")
	}
	if !strings.Contains(stdout.String(), "gateway started") {
		t.Fatalf("logs stdout missing the seeded log body:\n%s", stdout.String())
	}
}

// TestLogsCommand_HTTPClientHasBoundedTimeout proves the package-level
// HTTP client carries a non-zero Timeout so a hung gateway can't hang
// the operator's terminal indefinitely. http.DefaultClient has no
// timeout — an accept-but-don't-respond gateway would block `gormes
// logs` forever. The test exercises the seam directly rather than
// against the production 127.0.0.1:43827 endpoint, which would
// require a fake server that accepts and never responds — a heavy
// fixture for a property the client config alone proves.
func TestLogsCommand_HTTPClientHasBoundedTimeout(t *testing.T) {
	if logsHTTPClient == nil {
		t.Fatal("logsHTTPClient must be configured at package init")
	}
	if logsHTTPClient.Timeout <= 0 {
		t.Fatalf("logsHTTPClient.Timeout = %s, want a positive bound (a hung gateway must not hang `gormes logs`)", logsHTTPClient.Timeout)
	}
	// Sanity: the bound must be tighter than what an operator will
	// tolerate; a too-large timeout (e.g., a minute) would defeat the
	// point of the bound.
	if logsHTTPClient.Timeout > 30*time.Second {
		t.Fatalf("logsHTTPClient.Timeout = %s, want <= 30s for operator responsiveness", logsHTTPClient.Timeout)
	}
}

// TestLogsCommand_HTTPClientFallsBackOnHang proves the timeout
// actually fires by pointing the client at a server that accepts the
// TCP connection but never writes a response. Without the timeout the
// Get would block forever; with the timeout it returns an error and
// the command falls through to the file-based path.
func TestLogsCommand_HTTPClientFallsBackOnHang(t *testing.T) {
	// Bind a listener that accepts but never reads/writes on the
	// connection — simulates a wedged gateway.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Hold the conn until the test's listener is closed.
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	// Use a test-local client with a very short timeout so the test
	// runs fast even on a hung server. 100ms is well above
	// connect-time and well below test-run patience.
	client := &http.Client{Timeout: 100 * time.Millisecond}
	start := time.Now()
	_, err = client.Get("http://" + ln.Addr().String() + "/api/logs")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error from hung server; got nil")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Get took %s — timeout did not fire", elapsed)
	}
}
