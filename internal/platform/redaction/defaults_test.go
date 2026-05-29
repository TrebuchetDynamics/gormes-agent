package redaction

import "testing"

func TestRedactionDefaultOn_NewConfigEnablesRedaction(t *testing.T) {
	cfg := DefaultRedactionConfig()
	if !cfg.Enabled {
		t.Fatal("expected redaction enabled by default")
	}
	if !cfg.IsEnabled() {
		t.Fatal("IsEnabled() returned false for default config")
	}
	if cfg.DisabledReason() != "" {
		t.Fatalf("expected empty DisabledReason, got %q", cfg.DisabledReason())
	}
}

func TestRedactionDefaultOn_ExplicitOptOutHonored(t *testing.T) {
	cfg := RedactionConfig{Enabled: false}
	if cfg.IsEnabled() {
		t.Fatal("IsEnabled() returned true for disabled config")
	}
	if cfg.DisabledReason() == "" {
		t.Fatal("expected non-empty DisabledReason for disabled config")
	}
}

func TestRedactionDefaultOn_ExplicitEnableHonored(t *testing.T) {
	cfg := RedactionConfig{Enabled: true}
	if !cfg.IsEnabled() {
		t.Fatal("IsEnabled() returned false for explicitly enabled config")
	}
	if cfg.DisabledReason() != "" {
		t.Fatalf("expected empty DisabledReason, got %q", cfg.DisabledReason())
	}
}
