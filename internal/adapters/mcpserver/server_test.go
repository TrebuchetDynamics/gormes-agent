package mcpserver

import (
	"context"
	"testing"
)

func TestNewMCPServer(t *testing.T) {
	cfg := MCPConfig{
		Enabled:   true,
		SessionDB: "/tmp/test.db",
	}

	srv := NewMCPServer(cfg)
	if srv == nil {
		t.Fatal("NewMCPServer returned nil")
	}
	if srv.config.Enabled != true {
		t.Error("Expected config.Enabled to be true")
	}
	if srv.config.SessionDB != "/tmp/test.db" {
		t.Error("Expected config.SessionDB to be /tmp/test.db")
	}
}

func TestNewMCPServer_Disabled(t *testing.T) {
	cfg := MCPConfig{
		Enabled:   false,
		SessionDB: "",
	}

	srv := NewMCPServer(cfg)
	if srv == nil {
		t.Fatal("NewMCPServer returned nil")
	}
	if srv.config.Enabled != false {
		t.Error("Expected config.Enabled to be false")
	}
}

func TestMCPServer_StartStop(t *testing.T) {
	cfg := MCPConfig{Enabled: true}
	srv := NewMCPServer(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- srv.Start(ctx)
	}()

	cancel()
	<-done

	err := srv.Stop()
	if err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestMCPServer_StartDisabled(t *testing.T) {
	cfg := MCPConfig{Enabled: false}
	srv := NewMCPServer(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
}

func TestMCPTools(t *testing.T) {
	tools := MCPTools()
	if len(tools) != 9 {
		t.Fatalf("Expected 9 tools, got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Name)
		}
		if tool.Schema == nil {
			t.Errorf("Tool %s has nil Schema", tool.Name)
		}
	}

	expectedTools := []string{
		"conversations_list",
		"conversation_get",
		"messages_read",
		"attachments_fetch",
		"events_poll",
		"events_wait",
		"messages_send",
		"permissions_list_open",
		"permissions_respond",
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool %q not found", name)
		}
	}
}
