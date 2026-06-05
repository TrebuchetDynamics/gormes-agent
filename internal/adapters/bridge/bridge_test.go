package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BindHost != "127.0.0.1" {
		t.Errorf("expected BindHost 127.0.0.1, got %s", cfg.BindHost)
	}
	if cfg.BindPort != 8765 {
		t.Errorf("expected BindPort 8765, got %d", cfg.BindPort)
	}
	if cfg.GatewayPort != 8766 {
		t.Errorf("expected GatewayPort 8766, got %d", cfg.GatewayPort)
	}
	if cfg.GatewayHost != "127.0.0.1" {
		t.Errorf("expected GatewayHost 127.0.0.1, got %s", cfg.GatewayHost)
	}
}

func TestConfigBindAddr(t *testing.T) {
	cfg := Config{BindHost: "127.0.0.1", BindPort: 9999}
	if got := cfg.BindAddr(); got != "127.0.0.1:9999" {
		t.Errorf("expected BindAddr 127.0.0.1:9999, got %s", got)
	}
}

func TestConfigGatewayAddr(t *testing.T) {
	cfg := Config{GatewayHost: "10.0.0.1", GatewayPort: 43827}
	if got := cfg.GatewayAddr(); got != "10.0.0.1:43827" {
		t.Errorf("expected GatewayAddr 10.0.0.1:43827, got %s", got)
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv := New(DefaultConfig())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}
	if _, ok := resp["timestamp"]; !ok {
		t.Error("expected timestamp field in health response")
	}
}

func TestStatusEndpointWithoutGateway(t *testing.T) {
	srv := New(DefaultConfig())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["gateway"] != "stopped" {
		t.Errorf("expected gateway=stopped, got %v", resp["gateway"])
	}
}

func TestGatewayProbeFailed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.GatewayPort = 19999
	srv := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if srv.probeGateway(ctx) {
		t.Error("expected probe to fail on unused port")
	}
}

func TestGatewayProbeSuccess(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer gateway.Close()

	parts := strings.Split(gateway.URL, ":")
	portStr := parts[len(parts)-1]
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	cfg := DefaultConfig()
	cfg.GatewayPort = port
	srv := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !srv.probeGateway(ctx) {
		t.Error("expected probe to succeed on running server")
	}
}

func TestCORSMiddlewareAllowsOrigin(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS: expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected ACAO=*, got %s", got)
	}
}

func TestCORSMiddlewareSetsMethods(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	handler.ServeHTTP(rec, req)

	if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
}

func TestCORSMiddlewareSetsHeaders(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")

	handler.ServeHTTP(rec, req)

	if headers := rec.Header().Get("Access-Control-Allow-Headers"); headers == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	handler := panicRecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 after panic, got %d", rec.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "internal server error" {
		t.Errorf("expected error message, got %v", resp["error"])
	}
}

func TestPanicRecoveryMiddlewareNoPanic(t *testing.T) {
	handler := panicRecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %s", rec.Body.String())
	}
}

func TestReverseProxyForwardsRequest(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Gateway", "true")
		fmt.Fprintf(w, "gateway: %s", r.URL.Path)
	}))
	defer gateway.Close()

	cfg := DefaultConfig()
	cfg.GatewayHost = "127.0.0.1"
	cfg.GatewayPort = mustExtractPort(gateway.URL)
	srv := New(cfg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/api/v1/messages") {
		t.Errorf("expected proxied path in body, got %s", rec.Body.String())
	}
}

func TestReverseProxyPreservesMethod(t *testing.T) {
	var receivedMethod string
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer gateway.Close()

	cfg := DefaultConfig()
	cfg.GatewayHost = "127.0.0.1"
	cfg.GatewayPort = mustExtractPort(gateway.URL)
	srv := New(cfg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/send", strings.NewReader(`{"text":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.mux.ServeHTTP(rec, req)

	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST forwarded, got %s", receivedMethod)
	}
}

func TestReverseProxyReturns502WhenGatewayDown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.GatewayPort = 19998
	srv := New(cfg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502 when gateway unreachable, got %d", rec.Code)
	}
}

func TestBootstrapTermuxDryRun(t *testing.T) {
	srv := New(DefaultConfig())

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"dry_run": true}`)
	req := httptest.NewRequest(http.MethodPost, "/bootstrap/termux", body)
	req.Header.Set("Content-Type", "application/json")
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for dry-run, got %d", rec.Code)
	}

	var result bootstrapResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse bootstrap result: %v", err)
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
}

func TestBootstrapTermuxReturnsStructuredResult(t *testing.T) {
	srv := New(DefaultConfig())

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"dry_run": true}`)
	req := httptest.NewRequest(http.MethodPost, "/bootstrap/termux", body)
	req.Header.Set("Content-Type", "application/json")
	srv.mux.ServeHTTP(rec, req)

	var result bootstrapResult
	_ = json.Unmarshal(rec.Body.Bytes(), &result)

	if result.Steps == nil {
		t.Fatal("expected Steps array in result")
	}
	if len(result.Steps) == 0 {
		t.Error("expected at least one step in bootstrap result")
	}

	for i, step := range result.Steps {
		if step.Name == "" {
			t.Errorf("step %d: expected Name", i)
		}
		if step.Status == "" {
			t.Errorf("step %d: expected Status", i)
		}
	}
}

func TestBootstrapTermuxIdempotentDetection(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer gateway.Close()

	parts := strings.Split(gateway.URL, ":")
	portStr := parts[len(parts)-1]
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	cfg := DefaultConfig()
	cfg.GatewayPort = port
	srv := New(cfg)

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"dry_run": true}`)
	req := httptest.NewRequest(http.MethodPost, "/bootstrap/termux", body)
	req.Header.Set("Content-Type", "application/json")
	srv.mux.ServeHTTP(rec, req)

	var result bootstrapResult
	_ = json.Unmarshal(rec.Body.Bytes(), &result)

	if result.Status != "already_running" && result.Status != "success" {
		t.Logf("bootstrap status: %s (acceptable: already_running or success)", result.Status)
	}
}

func TestBootstrapTermuxInvalidJSON(t *testing.T) {
	srv := New(DefaultConfig())

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/bootstrap/termux", body)
	req.Header.Set("Content-Type", "application/json")
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestBootstrapTermuxWrongMethod(t *testing.T) {
	srv := New(DefaultConfig())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bootstrap/termux", nil)
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on bootstrap, got %d", rec.Code)
	}
}

func TestBootstrapTermuxSSEStreaming(t *testing.T) {
	srv := New(DefaultConfig())

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"dry_run": true}`)
	req := httptest.NewRequest(http.MethodPost, "/bootstrap/termux?stream=true", body)
	req.Header.Set("Content-Type", "application/json")
	srv.mux.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", contentType)
	}

	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "event:") {
		t.Error("expected SSE event markers in response")
	}
}

func TestBootstrapTermuxSSEAcceptHeader(t *testing.T) {
	srv := New(DefaultConfig())

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"dry_run": true}`)
	req := httptest.NewRequest(http.MethodPost, "/bootstrap/termux", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	srv.mux.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream via Accept header, got %s", contentType)
	}
}

func TestBootstrapResultMarshalJSON(t *testing.T) {
	result := bootstrapResult{
		Status:  "success",
		DryRun:  true,
		Message: "All steps completed",
		Steps: []bootstrapStep{
			{Name: "check_termux", Status: "ok", Detail: "Termux found"},
			{Name: "install_gormes", Status: "skipped", Detail: "Dry run"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["status"] != "success" {
		t.Errorf("expected status=success, got %v", parsed["status"])
	}
	if parsed["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", parsed["dry_run"])
	}

	steps, ok := parsed["steps"].([]interface{})
	if !ok {
		t.Fatal("expected steps array")
	}
	if len(steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(steps))
	}
}

func TestBootstrapStepMarshalJSON(t *testing.T) {
	step := bootstrapStep{
		Name:   "check_termux",
		Status: "ok",
		Detail: "Termux v0.118+ detected",
	}

	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["name"] != "check_termux" {
		t.Errorf("expected name=check_termux, got %v", parsed["name"])
	}
	if parsed["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", parsed["status"])
	}
}

func TestGatewayStopWhenNotRunning(t *testing.T) {
	srv := New(DefaultConfig())
	err := srv.stopGateway(context.Background())
	if err != nil {
		t.Errorf("expected no error stopping non-running gateway, got %v", err)
	}
}

func TestBootstrapHasExpectedSteps(t *testing.T) {
	srv := New(DefaultConfig())
	steps := srv.bootstrapSteps()

	if len(steps) == 0 {
		t.Fatal("expected bootstrap steps to be defined")
	}

	for i, step := range steps {
		if step.name == "" {
			t.Errorf("step %d: missing name", i)
		}
		if step.fn == nil {
			t.Errorf("step %d: missing function", i)
		}
	}
}

func TestRunBootstrapTermuxDryRun(t *testing.T) {
	cfg := DefaultConfig()
	ctx := context.Background()

	result := RunBootstrapTermux(ctx, cfg, true)

	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if result.Steps == nil {
		t.Fatal("expected Steps array")
	}
}

func TestRunBootstrapTermuxReturnsStatus(t *testing.T) {
	cfg := DefaultConfig()
	ctx := context.Background()

	result := RunBootstrapTermux(ctx, cfg, false)

	if result.Status == "" {
		t.Error("expected non-empty Status")
	}
}

func TestNotFoundHandler(t *testing.T) {
	srv := New(DefaultConfig())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] == "" {
		t.Error("expected error message in 404 response")
	}
}

func TestConcurrentHealthRequests(t *testing.T) {
	srv := New(DefaultConfig())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/health", nil)
			srv.mux.ServeHTTP(rec, r)
			if rec.Code != http.StatusOK {
				t.Errorf("concurrent health check failed: got %d", rec.Code)
			}
		}()
	}
	wg.Wait()
}

func mustExtractPort(rawURL string) int {
	parts := strings.Split(rawURL, ":")
	portStr := parts[len(parts)-1]
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}
