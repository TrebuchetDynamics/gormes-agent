package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestACPSetupBrowserDryRunJSON(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "acp", "--setup-browser", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("acp --setup-browser --dry-run --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action   string `json:"action"`
		OK       bool   `json:"ok"`
		DryRun   bool   `json:"dry_run"`
		Evidence struct {
			Code string `json:"code"`
		} `json:"evidence"`
		Steps []struct {
			Name    string   `json:"name"`
			Command []string `json:"command"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode stdout: %v\nstdout=%s", err, stdout)
	}
	if got.Build.Version != Version || got.Action != "acp_setup_browser" || !got.OK || !got.DryRun || got.Evidence.Code != "acp_setup_browser_plan" {
		t.Fatalf("report = %+v, want build/action/ok/dry-run/planned", got)
	}
	if !containsStepCommand(got.Steps, "agent-browser", "agent-browser@^0.26.0") {
		t.Fatalf("dry-run report missing agent-browser install command: %+v", got.Steps)
	}
	if strings.Contains(stdout, "sk-") || strings.Contains(stdout, "TOKEN=") || strings.Contains(stdout, "HERMES_HOME") {
		t.Fatalf("dry-run report leaked secret-looking or Hermes-owned state:\n%s", stdout)
	}
}

func TestACPSetupBrowserRequiresApprovalWithoutYes(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "acp", "--setup-browser", "--json")
	if err == nil {
		t.Fatalf("acp --setup-browser without --yes err=nil, want approval failure\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 1 {
		t.Fatalf("exit code = %d, want 1 (err=%v)\nstdout=%s\nstderr=%s", code, err, stdout, stderr)
	}
	var got struct {
		OK       bool `json:"ok"`
		Executed bool `json:"executed"`
		Evidence struct {
			Code string `json:"code"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode stdout: %v\nstdout=%s", err, stdout)
	}
	if got.OK || got.Executed || got.Evidence.Code != "acp_setup_browser_approval_required" {
		t.Fatalf("approval report = %+v, want not OK, not executed, approval-required", got)
	}
}

func containsStepCommand(steps []struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
}, name string, want string) bool {
	for _, step := range steps {
		if step.Name != name {
			continue
		}
		for _, part := range step.Command {
			if part == want {
				return true
			}
		}
	}
	return false
}
