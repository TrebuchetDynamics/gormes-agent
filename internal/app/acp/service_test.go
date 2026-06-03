package acpapp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/acp"
)

func TestRunSetupBrowserDryRunJSONIncludesBuildAndNoSecretState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	var stdout bytes.Buffer
	err := RunSetupBrowser(t.Context(), &stdout, acp.BrowserBootstrapOptions{DryRun: true}, true, BuildProvenance{Version: "test", GitCommit: "abc"})
	if err != nil {
		t.Fatalf("RunSetupBrowser: %v", err)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action string `json:"action"`
		OK     bool   `json:"ok"`
		DryRun bool   `json:"dry_run"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout.String())
	}
	if got.Build.Version != "test" || got.Action != "acp_setup_browser" || !got.OK || !got.DryRun {
		t.Fatalf("report = %+v", got)
	}
	if strings.Contains(stdout.String(), "HERMES_HOME") || strings.Contains(stdout.String(), "TOKEN=") {
		t.Fatalf("report leaked state: %s", stdout.String())
	}
}

func TestRunSetupBrowserApprovalRequiredReturnsExit1(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	var stdout bytes.Buffer
	err := RunSetupBrowser(t.Context(), &stdout, acp.BrowserBootstrapOptions{}, true, BuildProvenance{})
	if err == nil {
		t.Fatalf("err=nil, want approval error")
	}
	if got := ExitCode(err); got != 1 {
		t.Fatalf("ExitCode = %d, want 1 (err=%v)", got, err)
	}
	if !strings.Contains(stdout.String(), "acp_setup_browser_approval_required") {
		t.Fatalf("stdout missing evidence: %s", stdout.String())
	}
}

func TestRunServeProducesCleanJSONRPCFrame(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)
	if _, err := config.Load(nil); err != nil {
		t.Fatalf("load config: %v", err)
	}
	var stdout, stderr bytes.Buffer
	err := RunServe(t.Context(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1}}`+"\n"), &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunServe: %v\nstderr=%s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	var frame map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &frame); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
}
