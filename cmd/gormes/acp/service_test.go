package acp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunSetupBrowserDryRunJSONIncludesBuildProvenance(t *testing.T) {
	var out bytes.Buffer
	err := RunSetupBrowser(context.Background(), &out, BrowserBootstrapOptions{DryRun: true}, true, BuildProvenance{Version: "v-test", GitCommit: "abc123"})
	if err != nil {
		t.Fatalf("RunSetupBrowser dry-run: %v\nstdout=%s", err, out.String())
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action   string `json:"action"`
		OK       bool   `json:"ok"`
		DryRun   bool   `json:"dry_run"`
		Evidence struct {
			Code string `json:"code"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout: %v\nstdout=%s", err, out.String())
	}
	if got.Build.Version != "v-test" || got.Build.GitCommit != "abc123" {
		t.Fatalf("build = %+v, want v-test/abc123", got.Build)
	}
	if got.Action != "acp_setup_browser" || !got.OK || !got.DryRun || got.Evidence.Code != "acp_setup_browser_plan" {
		t.Fatalf("report = %+v, want plan dry-run OK", got)
	}
}

func TestRunSetupBrowserApprovalRequiredMapsToExitCode(t *testing.T) {
	var out bytes.Buffer
	err := RunSetupBrowser(context.Background(), &out, BrowserBootstrapOptions{}, true, BuildProvenance{})
	if err == nil {
		t.Fatalf("RunSetupBrowser without approval err=nil, want approval error\nstdout=%s", out.String())
	}
	if code := ExitCode(err); code != 1 {
		t.Fatalf("ExitCode() = %d, want 1 for approval-required error", code)
	}
	var got struct {
		OK       bool `json:"ok"`
		Executed bool `json:"executed"`
		Evidence struct {
			Code string `json:"code"`
		} `json:"evidence"`
	}
	if decodeErr := json.Unmarshal(out.Bytes(), &got); decodeErr != nil {
		t.Fatalf("decode stdout: %v\nstdout=%s", decodeErr, out.String())
	}
	if got.OK || got.Executed || got.Evidence.Code != "acp_setup_browser_approval_required" {
		t.Fatalf("approval report = %+v, want not OK, not executed, approval required", got)
	}
}

func TestDefaultClientOptionsUsesGormesServerCommand(t *testing.T) {
	if got := DefaultClientOptions().ServerCommand; got != "gormes" {
		t.Fatalf("ServerCommand = %q, want gormes", got)
	}
}

func TestParseProvenanceModeRejectsUnknownValue(t *testing.T) {
	_, err := ParseProvenanceMode("not-a-mode")
	if err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("ParseProvenanceMode() error = %v, want provenance validation error", err)
	}
}
