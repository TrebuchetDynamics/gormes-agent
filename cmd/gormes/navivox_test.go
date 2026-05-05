package main

import (
	"bytes"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/navivox"
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
