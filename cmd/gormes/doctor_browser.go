package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
)

const defaultChromeCDPEndpoint = "http://127.0.0.1:9222"

var chromeExecutableCandidates = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"chrome",
	"brave-browser",
	"microsoft-edge",
	"microsoft-edge-stable",
}

type browserRuntimeDoctorDeps struct {
	lookPath func(string) (string, error)
	getenv   func(string) string
	probeCDP func(context.Context, string) error
}

func doctorBrowserRuntimeStatus() doctor.CheckResult {
	return doctorBrowserRuntimeStatusWithDeps(browserRuntimeDoctorDeps{})
}

func doctorBrowserRuntimeStatusWithDeps(deps browserRuntimeDoctorDeps) doctor.CheckResult {
	if deps.lookPath == nil {
		deps.lookPath = exec.LookPath
	}
	if deps.getenv == nil {
		deps.getenv = os.Getenv
	}
	if deps.probeCDP == nil {
		deps.probeCDP = probeChromeCDPEndpoint
	}

	harnessPath, harnessOK := firstBrowserRuntimePath(deps.lookPath, "go-browser-harness")
	chromePath, chromeOK := firstBrowserRuntimePath(deps.lookPath, chromeExecutableCandidates...)
	endpoint := browserCDPEndpointFromEnv(deps.getenv)

	status := doctor.StatusPass
	summary := "local_cdp_ready"
	items := []doctor.ItemInfo{
		browserHarnessDoctorItem(harnessPath, harnessOK),
		browserChromeDoctorItem(chromePath, chromeOK, endpoint),
	}

	if endpoint == "" {
		status = doctor.StatusWarn
		if chromeOK {
			summary = "cdp_not_configured"
		} else {
			summary = "chrome_unavailable"
		}
		items = append(items, doctor.ItemInfo{
			Name:   "cdp",
			Status: doctor.StatusWarn,
			Note:   browserCDPSetupRecommendation(chromeLaunchCommand(chromePath)),
		})
		return doctor.CheckResult{Name: "Browser runtime", Status: status, Summary: summary, Items: items}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()
	if err := deps.probeCDP(ctx, endpoint); err != nil {
		status = doctor.StatusWarn
		summary = "cdp_unreachable"
		items = append(items, doctor.ItemInfo{
			Name:   "cdp",
			Status: doctor.StatusWarn,
			Note:   fmt.Sprintf("%s; %s", err.Error(), browserCDPSetupRecommendation(chromeLaunchCommand(chromePath))),
		})
		return doctor.CheckResult{Name: "Browser runtime", Status: status, Summary: summary, Items: items}
	}

	items = append(items, doctor.ItemInfo{
		Name:   "cdp",
		Status: doctor.StatusPass,
		Note:   "reachable at " + endpoint,
	})
	if !harnessOK {
		status = doctor.StatusWarn
		summary = "harness_unavailable"
	}
	return doctor.CheckResult{Name: "Browser runtime", Status: status, Summary: summary, Items: items}
}

func firstBrowserRuntimePath(lookPath func(string) (string, error), names ...string) (string, bool) {
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if resolved, err := lookPath(name); err == nil && strings.TrimSpace(resolved) != "" {
			return resolved, true
		}
	}
	return "", false
}

func browserHarnessDoctorItem(path string, ok bool) doctor.ItemInfo {
	if ok {
		return doctor.ItemInfo{Name: "harness", Status: doctor.StatusPass, Note: "go-browser-harness at " + path}
	}
	return doctor.ItemInfo{
		Name:   "harness",
		Status: doctor.StatusWarn,
		Note:   "go-browser-harness not found on PATH; build/install ../go-browser-harness so browser_* and CDP web_extract can run",
	}
}

func browserChromeDoctorItem(path string, ok bool, endpoint string) doctor.ItemInfo {
	if ok {
		return doctor.ItemInfo{Name: "chrome", Status: doctor.StatusPass, Note: "Chrome-compatible browser at " + path}
	}
	if strings.TrimSpace(endpoint) != "" {
		return doctor.ItemInfo{
			Name:   "chrome",
			Status: doctor.StatusPass,
			Note:   "local Chrome/Chromium not found; using configured CDP endpoint if reachable",
		}
	}
	return doctor.ItemInfo{
		Name:   "chrome",
		Status: doctor.StatusWarn,
		Note:   "Install Google Chrome or Chromium, then start it with " + chromeLaunchCommand(""),
	}
}

func browserCDPSetupRecommendation(command string) string {
	return "set BROWSER_CDP_URL=" + defaultChromeCDPEndpoint + " (or CHROME_REMOTE_DEBUGGING_URL) after starting Chrome with " + command
}

func browserCDPEndpointFromEnv(getenv func(string) string) string {
	return strings.TrimSpace(firstNonEmpty(getenv("BROWSER_CDP_URL"), getenv("CHROME_REMOTE_DEBUGGING_URL")))
}

func chromeLaunchCommand(chromePath string) string {
	command := strings.TrimSpace(chromePath)
	if command == "" {
		command = "google-chrome"
	}
	return command + " --remote-debugging-port=9222 --user-data-dir=$HOME/.gormes/chrome-debug --no-first-run --no-default-browser-check"
}

func probeChromeCDPEndpoint(ctx context.Context, endpoint string) error {
	versionURL, err := chromeCDPVersionURL(endpoint)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("CDP version endpoint returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func chromeCDPVersionURL(endpoint string) (string, error) {
	raw := strings.TrimSpace(endpoint)
	if raw == "" {
		return "", fmt.Errorf("CDP endpoint is empty")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse CDP endpoint: %w", err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("CDP endpoint must include a host")
	}
	switch parsed.Scheme {
	case "ws":
		parsed.Scheme = "http"
	case "wss":
		parsed.Scheme = "https"
	case "http", "https":
	default:
		return "", fmt.Errorf("CDP endpoint must use http, https, ws, or wss")
	}
	parsed.Path = "/json/version"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
