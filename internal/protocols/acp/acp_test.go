package acp

import (
	"context"
	"testing"
	"time"
)

func TestNewACPServer(t *testing.T) {
	cfg := ACPConfig{
		Port:       8080,
		SessionDir: "/tmp/sessions",
		Enabled:    true,
	}

	server := NewACPServer(cfg)
	if server == nil {
		t.Fatal("NewACPServer returned nil")
	}
	if server.port != cfg.Port {
		t.Errorf("expected port %d, got %d", cfg.Port, server.port)
	}
	if server.sessionDir != cfg.SessionDir {
		t.Errorf("expected sessionDir %s, got %s", cfg.SessionDir, server.sessionDir)
	}
	if server.enabled != cfg.Enabled {
		t.Errorf("expected enabled %v, got %v", cfg.Enabled, server.enabled)
	}
}

func TestACPServerStartStop(t *testing.T) {
	cfg := ACPConfig{
		Port:       8080,
		SessionDir: "/tmp/sessions",
		Enabled:    true,
	}

	server := NewACPServer(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	err := <-errCh
	if err == nil {
		t.Error("expected context error on Start, got nil")
	}
	if err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("expected context error, got %v", err)
	}

	if err := server.Stop(); err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
}

func TestACPToolKindConstants(t *testing.T) {
	kinds := []ACPToolKind{
		ACPToolKindRead,
		ACPToolKindEdit,
		ACPToolKindSearch,
		ACPToolKindExecute,
		ACPToolKindFetch,
		ACPToolKindThink,
	}

	expected := []string{"read", "edit", "search", "execute", "fetch", "think"}

	for i, kind := range kinds {
		if string(kind) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], kind)
		}
	}
}

func TestACPSession(t *testing.T) {
	now := time.Now()
	session := ACPSession{
		ID:        "test-session-1",
		CreatedAt: now,
		Model:     "claude-3-5-sonnet",
		Platform:  "zed",
	}

	if session.ID != "test-session-1" {
		t.Errorf("expected ID test-session-1, got %s", session.ID)
	}
	if !session.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, session.CreatedAt)
	}
	if session.Model != "claude-3-5-sonnet" {
		t.Errorf("expected Model claude-3-5-sonnet, got %s", session.Model)
	}
	if session.Platform != "zed" {
		t.Errorf("expected Platform zed, got %s", session.Platform)
	}
}
