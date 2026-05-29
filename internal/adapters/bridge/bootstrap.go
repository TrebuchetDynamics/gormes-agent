package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

type bootstrapStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type bootstrapResult struct {
	Status  string          `json:"status"`
	DryRun  bool            `json:"dry_run"`
	Message string          `json:"message,omitempty"`
	Steps   []bootstrapStep `json:"steps"`
}

type bootstrapStepDef struct {
	name string
	fn   func(context.Context, Config, bool) bootstrapStep
}

func (s *Server) handleBootstrapTermux(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "method not allowed, use POST",
		})
		return
	}

	var req struct {
		DryRun bool `json:"dry_run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid JSON body",
		})
		return
	}

	stream := r.URL.Query().Get("stream") == "true" || r.Header.Get("Accept") == "text/event-stream"

	if stream {
		s.handleBootstrapTermuxSSE(w, r, req.DryRun)
		return
	}

	result := RunBootstrapTermux(r.Context(), s.cfg, req.DryRun)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleBootstrapTermuxSSE(w http.ResponseWriter, r *http.Request, dryRun bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	steps := s.bootstrapSteps()
	for _, stepDef := range steps {
		step := stepDef.fn(r.Context(), s.cfg, dryRun)

		eventData, _ := json.Marshal(step)
		fmt.Fprintf(w, "event: step\ndata: %s\n\n", eventData)
		flusher.Flush()

		if step.Status == "error" && !dryRun {
			break
		}
	}

	finalStatus := "success"
	if dryRun {
		finalStatus = "dry_run_complete"
	}

	finalData, _ := json.Marshal(map[string]interface{}{
		"status":  finalStatus,
		"dry_run": dryRun,
	})
	fmt.Fprintf(w, "event: complete\ndata: %s\n\n", finalData)
	flusher.Flush()
}

func (s *Server) bootstrapSteps() []bootstrapStepDef {
	return []bootstrapStepDef{
		{"check_platform", stepCheckPlatform},
		{"check_termux", stepCheckTermux},
		{"check_termux_api", stepCheckTermuxAPI},
		{"check_gateway_port", stepCheckGatewayPort},
		{"install_gormes", stepInstallGormes},
		{"setup_termux_boot", stepSetupTermuxBoot},
		{"start_gateway", stepStartGateway},
		{"verify_gateway", stepVerifyGateway},
	}
}

func RunBootstrapTermux(ctx context.Context, cfg Config, dryRun bool) bootstrapResult {
	steps := []bootstrapStepDef{
		{"check_platform", stepCheckPlatform},
		{"check_termux", stepCheckTermux},
		{"check_termux_api", stepCheckTermuxAPI},
		{"check_gateway_port", stepCheckGatewayPort},
		{"install_gormes", stepInstallGormes},
		{"setup_termux_boot", stepSetupTermuxBoot},
		{"start_gateway", stepStartGateway},
		{"verify_gateway", stepVerifyGateway},
	}

	var resultSteps []bootstrapStep
	for _, stepDef := range steps {
		step := stepDef.fn(ctx, cfg, dryRun)
		resultSteps = append(resultSteps, step)
		if step.Status == "error" && !dryRun {
			return bootstrapResult{
				Status:  "failed",
				DryRun:  dryRun,
				Message: fmt.Sprintf("bootstrap failed at step: %s", step.Name),
				Steps:   resultSteps,
			}
		}
	}

	status := "success"
	if dryRun {
		status = "dry_run_complete"
	}

	return bootstrapResult{
		Status:  status,
		DryRun:  dryRun,
		Message: "bootstrap completed successfully",
		Steps:   resultSteps,
	}
}

func stepCheckPlatform(ctx context.Context, cfg Config, dryRun bool) bootstrapStep {
	if runtime.GOOS != "android" && runtime.GOOS != "linux" {
		return bootstrapStep{
			Name:   "check_platform",
			Status: "error",
			Detail: fmt.Sprintf("unsupported platform: %s", runtime.GOOS),
		}
	}
	return bootstrapStep{
		Name:   "check_platform",
		Status: "ok",
		Detail: fmt.Sprintf("platform: %s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func stepCheckTermux(ctx context.Context, cfg Config, dryRun bool) bootstrapStep {
	if _, err := exec.LookPath("termux-setup-storage"); err != nil {
		return bootstrapStep{
			Name:   "check_termux",
			Status: "error",
			Detail: "Termux not detected: termux-setup-storage not found",
		}
	}
	return bootstrapStep{
		Name:   "check_termux",
		Status: "ok",
		Detail: "Termux detected",
	}
}

func stepCheckTermuxAPI(ctx context.Context, cfg Config, dryRun bool) bootstrapStep {
	if _, err := exec.LookPath("termux-notification"); err != nil {
		return bootstrapStep{
			Name:   "check_termux_api",
			Status: "warning",
			Detail: "Termux:API not detected (optional for notifications)",
		}
	}
	return bootstrapStep{
		Name:   "check_termux_api",
		Status: "ok",
		Detail: "Termux:API detected",
	}
}

func stepCheckGatewayPort(ctx context.Context, cfg Config, dryRun bool) bootstrapStep {
	srv := &Server{cfg: cfg}
	if srv.probeGateway(ctx) {
		return bootstrapStep{
			Name:   "check_gateway_port",
			Status: "already_running",
			Detail: fmt.Sprintf("Gateway already running on port %d", cfg.GatewayPort),
		}
	}
	return bootstrapStep{
		Name:   "check_gateway_port",
		Status: "ok",
		Detail: fmt.Sprintf("Port %d available", cfg.GatewayPort),
	}
}

func stepInstallGormes(ctx context.Context, cfg Config, dryRun bool) bootstrapStep {
	if dryRun {
		return bootstrapStep{
			Name:   "install_gormes",
			Status: "skipped",
			Detail: "Dry run - would install/update gormes binary",
		}
	}

	bin := cfg.GormesBin
	if bin == "" {
		bin = "gormes"
	}

	if _, err := exec.LookPath(bin); err == nil {
		return bootstrapStep{
			Name:   "install_gormes",
			Status: "ok",
			Detail: "Gormes binary already installed",
		}
	}

	return bootstrapStep{
		Name:   "install_gormes",
		Status: "error",
		Detail: "Gormes binary not found and auto-install not yet implemented",
	}
}

func stepSetupTermuxBoot(ctx context.Context, cfg Config, dryRun bool) bootstrapStep {
	if dryRun {
		return bootstrapStep{
			Name:   "setup_termux_boot",
			Status: "skipped",
			Detail: "Dry run - would configure Termux:Boot",
		}
	}

	bootDir := os.Getenv("PREFIX")
	if bootDir == "" {
		bootDir = "/data/data/com.termux/files/usr"
	}
	bootScript := bootDir + "/etc/termux-boot/gormes.sh"

	if _, err := os.Stat(bootScript); err == nil {
		return bootstrapStep{
			Name:   "setup_termux_boot",
			Status: "ok",
			Detail: "Termux:Boot script already configured",
		}
	}

	return bootstrapStep{
		Name:   "setup_termux_boot",
		Status: "skipped",
		Detail: "Termux:Boot not configured (manual setup required)",
	}
}

func stepStartGateway(ctx context.Context, cfg Config, dryRun bool) bootstrapStep {
	if dryRun {
		return bootstrapStep{
			Name:   "start_gateway",
			Status: "skipped",
			Detail: "Dry run - would start gormes gateway",
		}
	}

	srv := &Server{cfg: cfg}
	if srv.probeGateway(ctx) {
		return bootstrapStep{
			Name:   "start_gateway",
			Status: "ok",
			Detail: "Gateway already running",
		}
	}

	return bootstrapStep{
		Name:   "start_gateway",
		Status: "skipped",
		Detail: "Gateway not started (requires manual start or Termux:Boot)",
	}
}

func stepVerifyGateway(ctx context.Context, cfg Config, dryRun bool) bootstrapStep {
	if dryRun {
		return bootstrapStep{
			Name:   "verify_gateway",
			Status: "skipped",
			Detail: "Dry run - would verify gateway health",
		}
	}

	srv := &Server{cfg: cfg}
	if srv.probeGateway(ctx) {
		return bootstrapStep{
			Name:   "verify_gateway",
			Status: "ok",
			Detail: "Gateway health check passed",
		}
	}

	return bootstrapStep{
		Name:   "verify_gateway",
		Status: "warning",
		Detail: "Gateway not responding (may need manual start)",
	}
}
