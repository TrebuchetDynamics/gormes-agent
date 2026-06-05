package acp

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestACPStdioLogsRouteToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	diag := NewStdioDiagnostics(&stderr)

	diag.Info("acp_stdio_start", "server ready")

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no diagnostic output", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "acp_stdio_start") || !strings.Contains(got, "server ready") {
		t.Fatalf("stderr = %q, want startup diagnostic", got)
	}
}

func TestACPBenignPingSuppressed(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := NewJSONRPCServer(NewSessionRuntime(SessionRuntimeConfig{}))
	server.Diagnostics = NewStdioDiagnostics(&stderr)

	err := server.Handle(context.Background(), strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"ping","params":{"prompt":"secret prompt"}}`+"\n",
	), &stdout)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	frames := decodeJSONRPCFrames(t, stdout.String())
	if len(frames) != 1 {
		t.Fatalf("frames = %d, want 1\nstdout=%s", len(frames), stdout.String())
	}
	errObj := errorMap(t, frames[0])
	if got := int(errObj["code"].(float64)); got != -32601 {
		t.Fatalf("error code = %d, want -32601", got)
	}
	data := errObj["data"].(map[string]any)
	if data["method"] != "ping" {
		t.Fatalf("error data = %#v, want method ping", data)
	}
	if strings.Contains(stdout.String(), "acp_benign_probe_suppressed") {
		t.Fatalf("stdout contains diagnostic evidence:\n%s", stdout.String())
	}
	logs := stderr.String()
	if !strings.Contains(logs, "acp_benign_probe_suppressed") || !strings.Contains(logs, "method=ping") {
		t.Fatalf("stderr = %q, want benign ping suppression evidence", logs)
	}
	if strings.Contains(logs, "Background task failed") || strings.Contains(strings.ToLower(logs), "traceback") {
		t.Fatalf("stderr contains traceback-like probe noise:\n%s", logs)
	}
	if strings.Contains(logs, "secret prompt") {
		t.Fatalf("stderr leaked raw params:\n%s", logs)
	}
}

func TestACPUnknownNonProbeStillVisible(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := NewJSONRPCServer(NewSessionRuntime(SessionRuntimeConfig{}))
	server.Diagnostics = NewStdioDiagnostics(&stderr)

	err := server.Handle(context.Background(), strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"session/custom","params":{}}`+"\n",
	), &stdout)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	errObj := errorMap(t, decodeJSONRPCFrames(t, stdout.String())[0])
	if got := int(errObj["code"].(float64)); got != -32601 {
		t.Fatalf("error code = %d, want -32601", got)
	}
	logs := stderr.String()
	if !strings.Contains(logs, "acp_stdio_error") || !strings.Contains(logs, "method=session/custom") {
		t.Fatalf("stderr = %q, want visible non-probe diagnostic", logs)
	}
	if strings.Contains(logs, "acp_benign_probe_suppressed") {
		t.Fatalf("stderr incorrectly suppressed non-probe unknown method:\n%s", logs)
	}
}

func TestACPMalformedJSONStillVisible(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := NewJSONRPCServer(NewSessionRuntime(SessionRuntimeConfig{}))
	server.Diagnostics = NewStdioDiagnostics(&stderr)

	err := server.Handle(context.Background(), strings.NewReader("{not json}\n"), &stdout)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	errObj := errorMap(t, decodeJSONRPCFrames(t, stdout.String())[0])
	if got := int(errObj["code"].(float64)); got != -32700 {
		t.Fatalf("error code = %d, want -32700", got)
	}
	logs := stderr.String()
	if !strings.Contains(logs, "acp_stdio_error") || !strings.Contains(logs, "reason=parse_error") {
		t.Fatalf("stderr = %q, want parse error diagnostic", logs)
	}
	if strings.Contains(logs, "acp_benign_probe_suppressed") {
		t.Fatalf("stderr incorrectly suppressed malformed JSON:\n%s", logs)
	}
}

func TestACPProbeSuppressionIsBounded(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := NewJSONRPCServer(NewSessionRuntime(SessionRuntimeConfig{}))
	server.Diagnostics = NewStdioDiagnostics(&stderr)

	raw := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"health","params":{"cwd":"/repo/secret","tool_args":{"token":"super-secret-token"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"healthcheck","params":{"prompt":"private prompt body"}}`,
	}, "\n") + "\n"
	err := server.Handle(context.Background(), strings.NewReader(raw), &stdout)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	frames := decodeJSONRPCFrames(t, stdout.String())
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2\nstdout=%s", len(frames), stdout.String())
	}
	logs := stderr.String()
	for _, want := range []string{"method=health", "method=healthcheck", "count=1"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("stderr = %q, missing %q", logs, want)
		}
	}
	for _, leak := range []string{"/repo/secret", "super-secret-token", "private prompt body", "tool_args"} {
		if strings.Contains(logs, leak) {
			t.Fatalf("stderr leaked %q:\n%s", leak, logs)
		}
	}
}

func errorMap(t *testing.T, frame map[string]any) map[string]any {
	t.Helper()
	errObj, ok := frame["error"].(map[string]any)
	if !ok {
		t.Fatalf("frame error = %#v, want object; frame=%#v", frame["error"], frame)
	}
	return errObj
}
