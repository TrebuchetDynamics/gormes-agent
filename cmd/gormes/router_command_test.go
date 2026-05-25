package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func runRouterTestCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newRouterCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestRouterDryRunRendersConfigWithoutBindingPort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data"))
	t.Setenv("GORMES_ROUTER_API_KEY", "router-secret-must-not-leak")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	body := `
[router]
enabled = true
listen = "127.0.0.1:9898"
api_key_env = "GORMES_ROUTER_API_KEY"
redact_logs = true
setup_mode = "local_gateway"

[[router.routes]]
name = "primary-provider"
alias = "primary-chat"
provider = "custom"
model = "fake-model"
base_url = "https://llm.example/v1"
api_key_env = "UPSTREAM_KEY"
transport = "chat_completions"
`
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(strings.TrimSpace(body)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runRouterTestCommand(t, "--dry-run")
	if err != nil {
		t.Fatalf("router --dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes Router dry run",
		"enabled=true",
		"listen=127.0.0.1:9898",
		"openai_base_url=http://127.0.0.1:9898/v1",
		"state=missing_credential",
		"dry_run_no_bind=true",
		"model=primary-chat provider=custom status=missing_credential",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{"router-secret-must-not-leak", "UPSTREAM_KEY=", "bind succeeded", "listening on"} {
		if strings.Contains(stdout+stderr, forbidden) {
			t.Fatalf("router dry-run leaked or bound unexpectedly via %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
}

func TestRouterCommandIsRegisteredOnRoot(t *testing.T) {
	root := newRootCommand()
	for _, cmd := range root.Commands() {
		if cmd.Name() == "router" {
			return
		}
	}
	t.Fatalf("root command did not expose router; commands=%v", rootCommandNames(root))
}

func rootCommandNames(root interface{ Commands() []*cobra.Command }) []string {
	commands := root.Commands()
	names := make([]string, 0, len(commands))
	for _, cmd := range commands {
		names = append(names, cmd.Name())
	}
	return names
}
