package acp

import (
	"context"
	"fmt"
	"log/slog"
)

// ACPConfig holds configuration for the ACP server.
type ACPConfig struct {
	Port       int
	SessionDir string
	Enabled    bool
}

// ServerStatus describes the ACP server's current readiness.
// Surfaces track which ACP protocol surfaces have been implemented
// vs which remain row-backed (planned for future progress.json rows).
type ServerStatus struct {
	Running          bool
	Port             int
	Enabled          bool
	Surfaces         []ServerSurfaceRow
	ImplementedCount int
	RowBackedCount   int
}

// ACPServer is the ACP (Agent Client Protocol) server instance.
// It exposes the agent via the ACP protocol for editor integrations.
// The server side manifest and surface contracts are validated;
// protocol handler implementations are tracked as row-backed surfaces
// awaiting the 5.H client bridge implementation.
type ACPServer struct {
	config     ACPConfig
	sessionDir string
	port       int
	enabled    bool
	manifest   ServerManifest
	running    bool
}

// NewACPServer creates a new ACP server instance with the given configuration.
func NewACPServer(cfg ACPConfig) *ACPServer {
	return &ACPServer{
		config:     cfg,
		sessionDir: cfg.SessionDir,
		port:       cfg.Port,
		enabled:    cfg.Enabled,
		manifest:   DefaultServerManifest(),
	}
}

// Start launches the ACP server. The server side manifest and surface
// contracts are validated; the protocol handler implementations are
// row-backed (see 5.H ACP client bridge mode in progress.json).
// Currently blocks until the context is cancelled, then returns.
func (s *ACPServer) Start(ctx context.Context) error {
	if !s.enabled {
		return fmt.Errorf("acp: server is disabled in config")
	}
	s.running = true
	slog.Info("acp server started",
		"port", s.port,
		"implemented_surfaces", s.countSurfaces(ServerSurfaceStatusImplemented),
		"row_backed_surfaces", s.countSurfaces(ServerSurfaceStatusRowBacked),
	)
	<-ctx.Done()
	s.running = false
	return ctx.Err()
}

// Stop gracefully shuts down the ACP server.
func (s *ACPServer) Stop() error {
	slog.Info("acp server stopping")
	s.running = false
	return nil
}

// Status returns the current ACP server state and surface readiness.
func (s *ACPServer) Status() ServerStatus {
	status := ServerStatus{
		Running:  s.running,
		Port:     s.port,
		Enabled:  s.enabled,
		Surfaces: append([]ServerSurfaceRow(nil), s.manifest.Surfaces...),
	}
	for _, row := range s.manifest.Surfaces {
		switch row.Status {
		case ServerSurfaceStatusImplemented:
			status.ImplementedCount++
		case ServerSurfaceStatusRowBacked:
			status.RowBackedCount++
		}
	}
	return status
}

func (s *ACPServer) countSurfaces(st ServerSurfaceStatus) int {
	n := 0
	for _, row := range s.manifest.Surfaces {
		if row.Status == st {
			n++
		}
	}
	return n
}
