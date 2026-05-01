package acp

import (
	"context"
	"log"
)

// ACPConfig holds configuration for the ACP server.
type ACPConfig struct {
	Port       int
	SessionDir string
	Enabled    bool
}

// ACPServer is the ACP (Agent Client Protocol) server instance.
// It exposes the agent via the ACP protocol for editor integrations.
type ACPServer struct {
	config     ACPConfig
	sessionDir string
	port       int
	enabled    bool
}

// NewACPServer creates a new ACP server instance with the given configuration.
func NewACPServer(cfg ACPConfig) *ACPServer {
	return &ACPServer{
		config:     cfg,
		sessionDir: cfg.SessionDir,
		port:       cfg.Port,
		enabled:    cfg.Enabled,
	}
}

// Start launches the ACP server. It blocks until the context is cancelled
// or the server encounters a fatal error.
func (s *ACPServer) Start(ctx context.Context) error {
	log.Printf("ACP server starting on port %d", s.port)
	// TODO: implement protocol handler
	<-ctx.Done()
	return ctx.Err()
}

// Stop gracefully shuts down the ACP server.
func (s *ACPServer) Stop() error {
	log.Printf("ACP server stopping")
	// TODO: implement graceful shutdown
	return nil
}
