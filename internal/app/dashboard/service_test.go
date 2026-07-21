package dashboard

import "testing"

func TestNewCommandPreservesDashboardFlagsWithoutPortShorthand(t *testing.T) {
	cmd := NewCommand(CommandOptions{})
	if cmd.Use != "dashboard" {
		t.Fatalf("Use = %q, want dashboard", cmd.Use)
	}
	host := cmd.Flags().Lookup("host")
	if host == nil {
		t.Fatalf("missing --host flag")
	}
	if got := host.DefValue; got != "127.0.0.1" {
		t.Fatalf("--host default = %q, want 127.0.0.1", got)
	}
	port := cmd.Flags().Lookup("port")
	if port == nil {
		t.Fatalf("missing --port flag")
	}
	if port.Shorthand != "" {
		t.Fatalf("--port shorthand = %q, want empty to avoid root --profile collision", port.Shorthand)
	}
	if got := port.DefValue; got != "43827" {
		t.Fatalf("--port default = %q, want 43827", got)
	}
	if cmd.Flags().Lookup("no-open") == nil {
		t.Fatalf("missing --no-open flag")
	}
}

func TestDefaultCommandOptionsReadsDashboardEnvAndBuildInfo(t *testing.T) {
	t.Setenv("GORMES_DASHBOARD_API_KEY", "dashboard-key")
	opts := DefaultCommandOptions("v-test", "abc", true)
	if opts.APIKey != "dashboard-key" || opts.Version != "v-test" || opts.GitCommit != "abc" || !opts.GitDirty {
		t.Fatalf("options = %+v", opts)
	}
	if opts.GoVersion == "" || opts.Stderr == nil || opts.OpenURL == nil {
		t.Fatalf("runtime defaults not populated: %+v", opts)
	}
}
