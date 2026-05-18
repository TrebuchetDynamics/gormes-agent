package acp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cmdrunner"
)

const (
	BrowserBootstrapEvidencePlanned          = "acp_setup_browser_plan"
	BrowserBootstrapEvidenceApprovalRequired = "acp_setup_browser_approval_required"
	BrowserBootstrapEvidenceComplete         = "acp_setup_browser_complete"
	BrowserBootstrapEvidenceCommandFailed    = "acp_setup_browser_command_failed"
	BrowserBootstrapEvidenceUnsupported      = "acp_setup_browser_unsupported"

	BrowserBootstrapStepPlanned   = "planned"
	BrowserBootstrapStepSkipped   = "skipped"
	BrowserBootstrapStepSucceeded = "succeeded"
	BrowserBootstrapStepFailed    = "failed"
)

type BrowserBootstrapOptions struct {
	Platform     string
	HomeDir      string
	DryRun       bool
	AssumeYes    bool
	SkipChromium bool
	Runner       cmdrunner.Runner
}

type BrowserBootstrapReport struct {
	OK               bool                   `json:"ok"`
	DryRun           bool                   `json:"dry_run"`
	Executed         bool                   `json:"executed"`
	RequiresApproval bool                   `json:"requires_approval,omitempty"`
	Platform         string                 `json:"platform"`
	HomeDir          string                 `json:"home_dir,omitempty"`
	NodePrefix       string                 `json:"node_prefix,omitempty"`
	Evidence         ClientEvidence         `json:"evidence"`
	Message          string                 `json:"message,omitempty"`
	Steps            []BrowserBootstrapStep `json:"steps"`
}

type BrowserBootstrapStep struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Command []string `json:"command,omitempty"`
	Message string   `json:"message,omitempty"`
}

func PlanBrowserBootstrap(opts BrowserBootstrapOptions) BrowserBootstrapReport {
	platform := normalizeBrowserBootstrapPlatform(opts.Platform)
	homeDir := strings.TrimSpace(opts.HomeDir)
	if homeDir == "" {
		homeDir = defaultBrowserBootstrapHome()
	}
	report := BrowserBootstrapReport{
		OK:         true,
		DryRun:     opts.DryRun,
		Platform:   platform,
		HomeDir:    homeDir,
		NodePrefix: joinBrowserBootstrapPath(platform, homeDir, "node"),
		Evidence:   ClientEvidence{Code: BrowserBootstrapEvidencePlanned},
		Message:    "ACP browser bootstrap plan prepared.",
	}
	if !browserBootstrapPlatformSupported(platform) {
		report.OK = false
		report.Evidence = ClientEvidence{Code: BrowserBootstrapEvidenceUnsupported, Reason: "unsupported_platform"}
		report.Message = fmt.Sprintf("ACP browser bootstrap is not supported on %s.", platform)
		return report
	}

	npm := "npm"
	npx := "npx"
	if platform == "windows" {
		npm = "npm.cmd"
		npx = "npx.cmd"
	}
	report.Steps = append(report.Steps,
		BrowserBootstrapStep{
			Name:    "node",
			Status:  BrowserBootstrapStepPlanned,
			Command: []string{"node", "--version"},
			Message: "Verify Node.js 20+ is available before installing browser tooling.",
		},
		BrowserBootstrapStep{
			Name:   "agent-browser",
			Status: BrowserBootstrapStepPlanned,
			Command: []string{
				npm, "install", "-g", "--prefix", report.NodePrefix, "--silent",
				"agent-browser@^0.26.0", "@askjo/camofox-browser@^1.5.2",
			},
			Message: "Install the Hermes-compatible agent-browser CLI and Camofox package into the Gormes-managed node prefix.",
		},
	)
	chromium := BrowserBootstrapStep{
		Name:    "chromium",
		Status:  BrowserBootstrapStepPlanned,
		Command: []string{npx, "--yes", "playwright", "install", "chromium"},
		Message: "Install Playwright Chromium for agent-browser when no system browser is configured.",
	}
	if opts.SkipChromium {
		chromium.Status = BrowserBootstrapStepSkipped
		chromium.Command = nil
		chromium.Message = "Chromium install skipped by operator request."
	}
	report.Steps = append(report.Steps, chromium)
	return report
}

func RunBrowserBootstrap(ctx context.Context, opts BrowserBootstrapOptions) BrowserBootstrapReport {
	report := PlanBrowserBootstrap(opts)
	if !report.OK || opts.DryRun {
		report.DryRun = opts.DryRun
		return report
	}
	if !opts.AssumeYes {
		report.OK = false
		report.RequiresApproval = true
		report.Evidence = ClientEvidence{Code: BrowserBootstrapEvidenceApprovalRequired}
		report.Message = "ACP browser bootstrap installs external packages; rerun with --yes to execute or --dry-run to preview."
		return report
	}
	runner := opts.Runner
	if runner == nil {
		runner = cmdrunner.ExecRunner{}
	}
	report.Executed = true
	report.Evidence = ClientEvidence{Code: BrowserBootstrapEvidenceComplete}
	report.Message = "ACP browser bootstrap completed."
	for i := range report.Steps {
		step := &report.Steps[i]
		if step.Status == BrowserBootstrapStepSkipped || len(step.Command) == 0 {
			continue
		}
		result := runner.Run(ctx, cmdrunner.Command{Name: step.Command[0], Args: step.Command[1:]})
		if result.Err != nil {
			step.Status = BrowserBootstrapStepFailed
			step.Message = firstNonEmptyString(strings.TrimSpace(result.Stderr), result.Err.Error())
			report.OK = false
			report.Evidence = ClientEvidence{Code: BrowserBootstrapEvidenceCommandFailed, Reason: step.Name}
			report.Message = fmt.Sprintf("ACP browser bootstrap failed during %s.", step.Name)
			return report
		}
		if step.Name == "node" {
			version := strings.TrimSpace(result.Stdout)
			if major, ok := nodeMajorVersion(version); ok && major < 20 {
				step.Status = BrowserBootstrapStepFailed
				step.Message = fmt.Sprintf("Node.js %s is older than v20.", version)
				report.OK = false
				report.Evidence = ClientEvidence{Code: BrowserBootstrapEvidenceCommandFailed, Reason: step.Name}
				report.Message = "ACP browser bootstrap requires Node.js v20 or newer."
				return report
			}
		}
		step.Status = BrowserBootstrapStepSucceeded
		if msg := strings.TrimSpace(result.Stdout); msg != "" {
			step.Message = msg
		}
	}
	return report
}

func normalizeBrowserBootstrapPlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "":
		return normalizeBrowserBootstrapPlatform(runtime.GOOS)
	case "windows", "win32":
		return "windows"
	case "darwin", "macos", "mac":
		return "darwin"
	case "linux":
		return "linux"
	default:
		return strings.ToLower(strings.TrimSpace(platform))
	}
}

func browserBootstrapPlatformSupported(platform string) bool {
	return platform == "linux" || platform == "darwin" || platform == "windows"
}

func defaultBrowserBootstrapHome() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ".gormes"
	}
	return filepath.Join(home, ".gormes")
}

func joinBrowserBootstrapPath(platform string, parts ...string) string {
	sep := string(os.PathSeparator)
	if platform == "windows" {
		sep = `\`
	} else if platform == "linux" || platform == "darwin" {
		sep = "/"
	}
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(cleaned) == 0 {
			cleaned = append(cleaned, strings.TrimRight(part, `/\`))
		} else {
			cleaned = append(cleaned, strings.Trim(part, `/\`))
		}
	}
	return strings.Join(cleaned, sep)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nodeMajorVersion(version string) (int, bool) {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if version == "" {
		return 0, false
	}
	major := 0
	readDigit := false
	for i := 0; i < len(version); i++ {
		if version[i] < '0' || version[i] > '9' {
			break
		}
		readDigit = true
		major = major*10 + int(version[i]-'0')
	}
	return major, readDigit
}
