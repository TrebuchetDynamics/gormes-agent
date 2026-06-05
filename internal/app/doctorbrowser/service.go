package doctorbrowser

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

const DefaultChromeCDPEndpoint = "http://127.0.0.1:9222"

var ChromeExecutableCandidates = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"chrome",
	"brave-browser",
	"microsoft-edge",
	"microsoft-edge-stable",
}

type RuntimeDoctorDeps struct {
	LookPath func(string) (string, error)
	Getenv   func(string) string
	ProbeCDP func(context.Context, string) error
	Offline  bool
}

func RuntimeStatus() doctor.CheckResult {
	return RuntimeStatusWithDeps(RuntimeDoctorDeps{})
}

func RuntimeStatusWithDeps(deps RuntimeDoctorDeps) doctor.CheckResult {
	if deps.LookPath == nil {
		deps.LookPath = exec.LookPath
	}
	if deps.Getenv == nil {
		deps.Getenv = os.Getenv
	}
	if deps.ProbeCDP == nil {
		deps.ProbeCDP = ProbeChromeCDPEndpoint
	}

	chromePath, chromeOK := FirstRuntimePath(deps.LookPath, ChromeExecutableCandidates...)
	endpoint := CDPEndpointFromEnv(deps.Getenv)

	status := doctor.StatusPass
	summary := "local_cdp_ready"
	items := []doctor.ItemInfo{
		chromeDoctorItem(chromePath, chromeOK, endpoint),
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
			Note:   CDPSetupRecommendation(ChromeLaunchCommand(chromePath)),
		})
		return doctor.CheckResult{Name: "Browser runtime", Status: status, Summary: summary, Items: items}
	}

	if deps.Offline {
		items = append(items, doctor.ItemInfo{
			Name:   "cdp",
			Status: doctor.StatusSkip,
			Note:   "configured at " + endpoint + "; reachability probe skipped --offline",
		})
		return doctor.CheckResult{Name: "Browser runtime", Status: status, Summary: "cdp_configured_offline", Items: items}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()
	if err := deps.ProbeCDP(ctx, endpoint); err != nil {
		status = doctor.StatusWarn
		summary = "cdp_unreachable"
		items = append(items, doctor.ItemInfo{
			Name:   "cdp",
			Status: doctor.StatusWarn,
			Note:   fmt.Sprintf("%s; %s", err.Error(), CDPSetupRecommendation(ChromeLaunchCommand(chromePath))),
		})
		return doctor.CheckResult{Name: "Browser runtime", Status: status, Summary: summary, Items: items}
	}

	items = append(items, doctor.ItemInfo{
		Name:   "cdp",
		Status: doctor.StatusPass,
		Note:   "reachable at " + endpoint,
	})
	return doctor.CheckResult{Name: "Browser runtime", Status: status, Summary: summary, Items: items}
}

func FirstRuntimePath(lookPath func(string) (string, error), names ...string) (string, bool) {
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

func chromeDoctorItem(path string, ok bool, endpoint string) doctor.ItemInfo {
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
		Note:   "Install Google Chrome or Chromium, then start it with " + ChromeLaunchCommand(""),
	}
}

func CDPSetupRecommendation(command string) string {
	return "set BROWSER_CDP_URL=" + DefaultChromeCDPEndpoint + " (or CHROME_REMOTE_DEBUGGING_URL) after starting Chrome with " + command
}

func CDPEndpointFromEnv(getenv func(string) string) string {
	return strings.TrimSpace(firstNonEmpty(getenv("BROWSER_CDP_URL"), getenv("CHROME_REMOTE_DEBUGGING_URL")))
}

func ChromeLaunchCommand(chromePath string) string {
	command := strings.TrimSpace(chromePath)
	if command == "" {
		command = "google-chrome"
	}
	return command + " --remote-debugging-port=9222 --user-data-dir=" + filepath.Join(config.GormesHome(), "chrome-debug") + " --no-first-run --no-default-browser-check"
}

func ProbeChromeCDPEndpoint(ctx context.Context, endpoint string) error {
	versionURL, err := ChromeCDPVersionURL(endpoint)
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

func ChromeCDPVersionURL(endpoint string) (string, error) {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
