package gormescli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// TestACPClientCommand_JSONIncludesBuildProvenance proves
// `gormes acp client --json` emits a top-level `build` envelope so
// fleet automation bridging ACP sessions across machines can attribute
// each connect/degraded result to the binary version that emitted it.
// Existing top-level ClientResult fields (ok/session_key/etc.) remain
// addressable through struct embedding — additive change.
func TestACPClientCommand_JSONIncludesBuildProvenance(t *testing.T) {
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

	cmd := newACPCommandForTest()
	stdout, stderr, err := executeACPCommandForTest(cmd,
		"client",
		"--session", "agent:main:main",
		"--require-existing",
		"--provenance", "meta+receipt",
		"--cwd", "/repo",
		"--json",
	)
	if err != nil {
		t.Fatalf("acp client --json: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		OK         bool   `json:"ok"`
		SessionKey string `json:"session_key"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if !got.OK || got.SessionKey != "agent:main:main" {
		t.Errorf("ok=%v key=%q, want true/agent:main:main", got.OK, got.SessionKey)
	}
}

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

	cmd := newACPCommandForTest()
	stdout, stderr, err := executeACPCommandForTest(cmd,
		"client",
		"--session", "agent:main:main",
		"--require-existing",
		"--provenance", "meta+receipt",
		"--cwd", "/repo",
		"--json",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}

	var result ACPClientResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode JSON: %v\nstdout=%s", err, stdout)
	}
	if !result.OK || result.Code != ACPClientEvidenceConnected {
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

	cmd := newACPCommandForTest()
	stdout, stderr, err := executeACPCommandForTest(cmd,
		"client",
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

	var result ACPClientResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode JSON: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if result.OK || result.Evidence.Code != ACPClientEvidenceRowBacked || result.Evidence.Reason != "session_key_not_found" {
		t.Fatalf("result = %+v, want row-backed missing session evidence", result)
	}
}

func TestACPServeCommandRunsJSONRPCOverStdio(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	stdout, stderr, err := executeCobraCommandForTest(newACPCommandForTest(), cobraCommandExecutionOptions{Input: strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1}}` + "\n")}, "serve")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want no JSON-RPC transport contamination", stderr)
	}
	var frame map[string]any
	if err := json.Unmarshal(bytes.TrimSpace([]byte(stdout)), &frame); err != nil {
		t.Fatalf("stdout is not one clean JSON-RPC frame: %v\nstdout=%s", err, stdout)
	}
	result := frame["result"].(map[string]any)
	caps := result["agentCapabilities"].(map[string]any)
	if caps["loadSession"] != true {
		t.Fatalf("loadSession = %#v, want true", caps["loadSession"])
	}
}
