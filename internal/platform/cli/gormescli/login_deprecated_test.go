package gormescli

import (
	"bytes"
	"strings"
	"testing"
)

// TestDeprecatedLoginCommand_ExitsZeroWithDeprecationMessage verifies that
// gormes login is a registered hidden command (not an "unknown command" typo)
// that prints the Hermes-aligned three-alternative deprecation message to
// stderr and returns nil (exit 0).
//
// Hermes contract: hermes_cli/auth.py:6658 login_command raises SystemExit(0)
// after printing the three-line message; hermes_cli/subcommands/login.py
// registers the subparser without a help= string (hidden from --help) and
// accepts all OAuth flags so old scripts don't get argparse errors.
func TestDeprecatedLoginCommand_ExitsZeroWithDeprecationMessage(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"bare", []string{}},
		{"with-provider", []string{"--provider", "openai-codex"}},
		{"with-all-flags", []string{"--provider", "openai-codex", "--no-browser", "--timeout", "120", "--ca-bundle", "/etc/ssl/certs/ca.pem", "--insecure"}},
		{"with-portal-flags", []string{"--portal-url", "https://portal.example.com", "--inference-url", "https://api.example.com", "--client-id", "myclient", "--scope", "openid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			cmd := NewDeprecatedLoginCommand()
			cmd.SetErr(&stderr)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("gormes login %v must exit 0 (return nil); got error: %v\nstderr: %s", tc.args, err, stderr.String())
			}
			out := stderr.String()
			if !strings.Contains(out, "gormes login") {
				t.Errorf("stderr missing 'gormes login'; got: %q", out)
			}
			if !strings.Contains(out, "gormes auth") {
				t.Errorf("stderr missing 'gormes auth'; got: %q", out)
			}
			if !strings.Contains(out, "gormes model") {
				t.Errorf("stderr missing 'gormes model'; got: %q", out)
			}
			if !strings.Contains(out, "gormes setup") {
				t.Errorf("stderr missing 'gormes setup'; got: %q", out)
			}
		})
	}
}

func TestDeprecatedLoginCommand_IsHiddenFromHelp(t *testing.T) {
	cmd := NewDeprecatedLoginCommand()
	if !cmd.Hidden {
		t.Fatal("gormes login must have Hidden=true so it is omitted from gormes --help")
	}
}

func TestDeprecatedLoginCommand_NotInRootHelp(t *testing.T) {
	factories := stubRootFactories()
	factories["login"] = NewDeprecatedLoginCommand
	root := NewRootCommand(RootOptions{Version: "test"}, factories)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--help"})
	_ = root.Execute()
	if strings.Contains(out.String(), "\n  login") {
		t.Fatalf("gormes --help must not list 'login'; got:\n%s", out.String())
	}
}
