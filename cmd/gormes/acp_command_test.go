package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/acp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestACPClientCommandConnectsWithSessionAndJSON(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	ctx := t.Context()
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	if err := smap.Put(ctx, "agent:main:main", "sess-main"); err != nil {
		t.Fatalf("Put session: %v", err)
	}
	if err := smap.Close(); err != nil {
		t.Fatalf("Close session map: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called")
			return nil
		},
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"acp", "client",
		"--session", "agent:main:main",
		"--require-existing",
		"--provenance", "meta+receipt",
		"--cwd", "/repo",
		"--json",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}

	var result acp.ClientResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode JSON: %v\nstdout=%s", err, stdout)
	}
	if !result.OK || result.Code != acp.ClientEvidenceConnected {
		t.Fatalf("result = %+v, want connected", result)
	}
	if result.SessionKey != "agent:main:main" || result.SessionID != "sess-main" {
		t.Fatalf("session = %q/%q, want agent:main:main/sess-main", result.SessionKey, result.SessionID)
	}
	if !strings.Contains(result.Receipt, "signature=sha256:") {
		t.Fatalf("receipt missing signed hash: %q", result.Receipt)
	}
}

func TestACPClientCommandRequireExistingMissingSessionExits1WithEvidence(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called")
			return nil
		},
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"acp", "client",
		"--session", "agent:missing:main",
		"--require-existing",
		"--json",
	)
	if err == nil {
		t.Fatalf("Execute() error = nil, want exit 1\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 1 {
		t.Fatalf("exit code = %d, want 1 (err=%v)\nstderr=%s", code, err, stderr)
	}

	var result acp.ClientResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode JSON: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if result.OK || result.Evidence.Code != acp.ClientEvidenceRowBacked || result.Evidence.Reason != "session_key_not_found" {
		t.Fatalf("result = %+v, want row-backed missing session evidence", result)
	}
}
