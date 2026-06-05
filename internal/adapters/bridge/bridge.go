package bridge

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/httpjson"
)

const (
	DefaultBindHost    = "127.0.0.1"
	DefaultBindPort    = 8765
	DefaultGatewayHost = "127.0.0.1"
	DefaultGatewayPort = 8766
)

type Config struct {
	BindHost    string
	BindPort    int
	GatewayHost string
	GatewayPort int
	GormesBin   string
}

type Option func(*Config)

func WithGatewayAddr(addr string) Option {
	return func(c *Config) {
		c.GatewayHost = ""
		c.GatewayPort = 0
		c.GormesBin = addr
	}
}

func DefaultConfig() Config {
	return Config{
		BindHost:    DefaultBindHost,
		BindPort:    DefaultBindPort,
		GatewayHost: DefaultGatewayHost,
		GatewayPort: DefaultGatewayPort,
		GormesBin:   "gormes",
	}
}

func (c Config) BindAddr() string {
	return fmt.Sprintf("%s:%d", c.BindHost, c.BindPort)
}

func (c Config) GatewayAddr() string {
	return fmt.Sprintf("%s:%d", c.GatewayHost, c.GatewayPort)
}

type Server struct {
	cfg    Config
	mux    *http.ServeMux
	server *http.Server
	proxy  *httputil.ReverseProxy

	mu             sync.Mutex
	gatewayCmd     *exec.Cmd
	gatewayRunning bool
}

func New(cfg Config) *Server {
	mux := http.NewServeMux()
	srv := &Server{
		cfg:   cfg,
		mux:   mux,
		proxy: newGatewayProxy(cfg),
	}

	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/status", srv.handleStatus)
	mux.HandleFunc("/bootstrap/termux", srv.handleBootstrapTermux)

	apiHandler := http.HandlerFunc(srv.handleProxy)
	mux.Handle("/api/", corsMiddleware(panicRecoveryMiddleware(apiHandler)))
	mux.Handle("/sse/", corsMiddleware(panicRecoveryMiddleware(apiHandler)))

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpjson.Write(w, http.StatusNotFound, map[string]string{
			"error": "not found",
			"path":  r.URL.Path,
		})
	}))

	return srv
}

func (s *Server) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:    s.cfg.BindAddr(),
		Handler: s.mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	httpjson.Write(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	gatewayStatus := "stopped"
	if s.probeGateway(r.Context()) {
		gatewayStatus = "running"
	}

	httpjson.Write(w, http.StatusOK, map[string]interface{}{
		"bridge":       "running",
		"gateway":      gatewayStatus,
		"gateway_addr": s.cfg.GatewayAddr(),
	})
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func (s *Server) probeGateway(ctx context.Context) bool {
	addr := s.cfg.GatewayAddr()
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/api/status", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

func (s *Server) stopGateway(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.gatewayRunning || s.gatewayCmd == nil {
		return nil
	}

	if s.gatewayCmd.Process != nil {
		if err := s.gatewayCmd.Process.Signal(os.Interrupt); err != nil {
			_ = s.gatewayCmd.Process.Kill()
		}
	}

	done := make(chan struct{}, 1)
	go func() {
		_ = s.gatewayCmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = s.gatewayCmd.Process.Kill()
	}

	s.gatewayRunning = false
	s.gatewayCmd = nil
	return nil
}

func newGatewayProxy(cfg Config) *httputil.ReverseProxy {
	target := &url.URL{
		Scheme: "http",
		Host:   cfg.GatewayAddr(),
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalErrorHandler := proxy.ErrorHandler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		httpjson.Write(w, http.StatusBadGateway, map[string]interface{}{
			"error":  "gateway unreachable",
			"detail": err.Error(),
		})
	}
	_ = originalErrorHandler

	return proxy
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func panicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				httpjson.Write(w, http.StatusInternalServerError, map[string]string{
					"error": "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
