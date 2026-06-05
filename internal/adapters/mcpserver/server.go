// Package mcpserver provides a Model Context Protocol (MCP) server scaffold
// for gormes-agent. This is a stub implementation that defines the server
// structure and tool surface without live MCP protocol handling.
//
// The MCP server exposes messaging conversation tools matching the Hermes
// 9-tool surface: conversations_list, conversation_get, messages_read,
// attachments_fetch, events_poll, events_wait, messages_send,
// permissions_list_open, permissions_respond.
package mcpserver

import (
	"context"
)

// MCPConfig holds configuration for the MCP server.
type MCPConfig struct {
	Enabled   bool
	SessionDB string
}

// MCPServer is the MCP server instance. It provides access to messaging
// conversations across connected platforms via the MCP tool interface.
type MCPServer struct {
	config  MCPConfig
	running bool
}

// NewMCPServer creates a new MCP server instance with the given configuration.
func NewMCPServer(cfg MCPConfig) *MCPServer {
	return &MCPServer{
		config:  cfg,
		running: false,
	}
}

// Start launches the MCP server. It blocks until the context is cancelled
// or the server encounters a fatal error. This is a stub implementation.
func (s *MCPServer) Start(ctx context.Context) error {
	if !s.config.Enabled {
		return nil
	}
	s.running = true
	<-ctx.Done()
	s.running = false
	return ctx.Err()
}

// Stop gracefully shuts down the MCP server. This is a stub implementation.
func (s *MCPServer) Stop() error {
	s.running = false
	return nil
}
