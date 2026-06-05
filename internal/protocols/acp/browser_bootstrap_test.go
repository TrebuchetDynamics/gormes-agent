package acp

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/runtime/cmdrunner"
)

func TestACPSetupBrowserBootstrapPlansLinuxAndWindows(t *testing.T) {
	linux := PlanBrowserBootstrap(BrowserBootstrapOptions{
		Platform: "linux",
		HomeDir:  "/tmp/gormes",
		DryRun:   true,
	})
	if !linux.OK || !linux.DryRun || linux.Evidence.Code != BrowserBootstrapEvidencePlanned {
		t.Fatalf("linux plan = %+v, want OK dry-run planned", linux)
	}
	assertBootstrapStepCommand(t, linux, "node", []string{"node", "--version"})
	assertBootstrapStepCommand(t, linux, "agent-browser", []string{
		"npm", "install", "-g", "--prefix", "/tmp/gormes/node", "--silent",
		"agent-browser@^0.26.0", "@askjo/camofox-browser@^1.5.2",
	})
	assertBootstrapStepCommand(t, linux, "chromium", []string{"npx", "--yes", "playwright", "install", "chromium"})

	windows := PlanBrowserBootstrap(BrowserBootstrapOptions{
		Platform: "windows",
		HomeDir:  `C:\Users\me\.gormes`,
		DryRun:   true,
	})
	if !windows.OK || windows.Platform != "windows" {
		t.Fatalf("windows plan = %+v, want OK windows", windows)
	}
	assertBootstrapStepCommand(t, windows, "agent-browser", []string{
		"npm.cmd", "install", "-g", "--prefix", `C:\Users\me\.gormes\node`, "--silent",
		"agent-browser@^0.26.0", "@askjo/camofox-browser@^1.5.2",
	})
	assertBootstrapStepCommand(t, windows, "chromium", []string{"npx.cmd", "--yes", "playwright", "install", "chromium"})
}

func TestACPSetupBrowserBootstrapRequiresApprovalBeforeExecution(t *testing.T) {
	runner := &cmdrunner.FakeRunner{}
	report := RunBrowserBootstrap(context.Background(), BrowserBootstrapOptions{
		Platform: "linux",
		HomeDir:  "/tmp/gormes",
		Runner:   runner,
	})
	if report.OK || report.Executed || report.Evidence.Code != BrowserBootstrapEvidenceApprovalRequired {
		t.Fatalf("unapproved run = %+v, want approval-required degraded report", report)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("unapproved run executed commands: %+v", runner.Commands)
	}
}

func TestACPSetupBrowserBootstrapExecutesApprovedSteps(t *testing.T) {
	runner := &cmdrunner.FakeRunner{
		Results: []cmdrunner.Result{
			{Stdout: "v22.1.0\n"},
			{Stdout: "npm ok\n"},
			{Stdout: "chromium ok\n"},
		},
	}
	report := RunBrowserBootstrap(context.Background(), BrowserBootstrapOptions{
		Platform:  "linux",
		HomeDir:   "/tmp/gormes",
		AssumeYes: true,
		Runner:    runner,
	})
	if !report.OK || !report.Executed || report.Evidence.Code != BrowserBootstrapEvidenceComplete {
		t.Fatalf("approved run = %+v, want executed complete report", report)
	}
	wantCommands := [][]string{
		{"node", "--version"},
		{"npm", "install", "-g", "--prefix", "/tmp/gormes/node", "--silent", "agent-browser@^0.26.0", "@askjo/camofox-browser@^1.5.2"},
		{"npx", "--yes", "playwright", "install", "chromium"},
	}
	if got := commandArgv(runner.Commands); !reflect.DeepEqual(got, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", got, wantCommands)
	}
	for _, step := range report.Steps {
		if step.Status != BrowserBootstrapStepSucceeded {
			t.Fatalf("step %s status = %q, want succeeded; report=%+v", step.Name, step.Status, report)
		}
	}
}

func TestACPSetupBrowserBootstrapReportsMissingPrerequisite(t *testing.T) {
	runner := &cmdrunner.FakeRunner{
		Results: []cmdrunner.Result{{Err: errors.New("exec: node: executable file not found"), Stderr: "node missing"}},
	}
	report := RunBrowserBootstrap(context.Background(), BrowserBootstrapOptions{
		Platform:  "linux",
		HomeDir:   "/tmp/gormes",
		AssumeYes: true,
		Runner:    runner,
	})
	if report.OK || !report.Executed || report.Evidence.Code != BrowserBootstrapEvidenceCommandFailed {
		t.Fatalf("failed run = %+v, want command-failed report", report)
	}
	step, ok := findBootstrapStep(report, "node")
	if !ok || step.Status != BrowserBootstrapStepFailed || !strings.Contains(step.Message, "node missing") {
		t.Fatalf("node failure step = %+v ok=%v, want failed node missing", step, ok)
	}
}

func TestACPSetupBrowserBootstrapRejectsOldNode(t *testing.T) {
	runner := &cmdrunner.FakeRunner{
		Results: []cmdrunner.Result{{Stdout: "v18.19.0\n"}},
	}
	report := RunBrowserBootstrap(context.Background(), BrowserBootstrapOptions{
		Platform:  "linux",
		HomeDir:   "/tmp/gormes",
		AssumeYes: true,
		Runner:    runner,
	})
	if report.OK || report.Evidence.Code != BrowserBootstrapEvidenceCommandFailed {
		t.Fatalf("old-node run = %+v, want command-failed report", report)
	}
	if got := commandArgv(runner.Commands); len(got) != 1 || !reflect.DeepEqual(got[0], []string{"node", "--version"}) {
		t.Fatalf("old-node commands = %#v, want only node check", got)
	}
	step, ok := findBootstrapStep(report, "node")
	if !ok || step.Status != BrowserBootstrapStepFailed || !strings.Contains(step.Message, "older than v20") {
		t.Fatalf("old-node step = %+v ok=%v, want failed older-than-v20", step, ok)
	}
}

func assertBootstrapStepCommand(t *testing.T, report BrowserBootstrapReport, name string, want []string) {
	t.Helper()
	step, ok := findBootstrapStep(report, name)
	if !ok {
		t.Fatalf("missing step %q in %+v", name, report)
	}
	if !reflect.DeepEqual(step.Command, want) {
		t.Fatalf("step %s command = %#v, want %#v", name, step.Command, want)
	}
}

func findBootstrapStep(report BrowserBootstrapReport, name string) (BrowserBootstrapStep, bool) {
	for _, step := range report.Steps {
		if step.Name == name {
			return step, true
		}
	}
	return BrowserBootstrapStep{}, false
}

func commandArgv(commands []cmdrunner.Command) [][]string {
	out := make([][]string, 0, len(commands))
	for _, command := range commands {
		out = append(out, append([]string{command.Name}, command.Args...))
	}
	return out
}
