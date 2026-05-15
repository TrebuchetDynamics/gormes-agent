package main

import (
	"strings"
	"testing"
)

func TestRootHelpGuidesFirstRunOperators(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--help")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}

	for _, want := range []string{
		"Fast paths:",
		"Fresh install:       gormes setup -> gormes chat",
		"Targeted setup:      gormes setup --quick --target terminal",
		"Scripted chat:       gormes chat -q \"summarize this repo\"",
		"Messaging gateway:   gormes gateway status -> gormes gateway -> gormes logs",
		"Operator workflows:",
		"First run and setup",
		"gormes setup                      run guided setup",
		"gormes setup --quick              configure missing setup items only",
		"Provider/auth/debug",
		"gormes debug share",
		"Session and memory",
		"Automation and integrations",
		"Examples:",
		"gormes --offline --profile test",
		"gormes chat -q \"write a release note\"",
		"gormes chat",
		"gormes gateway status --json",
		"gormes config check --json",
		"Config and state:",
		"config:   ~/.gormes/config.toml",
		"secrets:  ~/.gormes/.env",
		"profiles: gormes profile list",
		"Need more detail:",
		"gormes help <command>",
		"gormes completion <shell>",
		"docs: https://docs.gormes.ai",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("root help missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{
		"gormes debug bundle",
		"gormes onboard",
		"gormes login",
		"--oneshot",
		"one-shot",
		"oneshot",
		"\n  onboard     ",
		"\n  login       ",
	} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("root help advertised unimplemented shortcut %q:\n%s", forbidden, stdout)
		}
	}
}

func TestRemovedTopLevelCommandsDoNotResolveThroughHelpCommand(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	for _, topic := range []string{"login", "onboard"} {
		t.Run(topic, func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeOneshotFlagCommand(cmd, "help", topic)
			if err == nil {
				t.Fatalf("help %s error = nil\nstdout=%s\nstderr=%s", topic, stdout, stderr)
			}
			combined := stdout + stderr + err.Error()
			if !strings.Contains(combined, "unknown help topic") {
				t.Fatalf("help %s output = %q, want unknown help topic", topic, combined)
			}
			for _, forbidden := range []string{"Usage:", "Deprecated", "deprecated", "--oneshot", "gormes onboard", "gormes login"} {
				if strings.Contains(combined, forbidden) {
					t.Fatalf("help %s leaked %q:\nstdout=%s\nstderr=%s\nerr=%v", topic, forbidden, stdout, stderr, err)
				}
			}
		})
	}
}

func TestRemovedTopLevelEntrypointsReturnReplacementGuidance(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "login",
			args: []string{"login", "--provider", "plain-secret-provider"},
			want: []string{"unknown command \"login\"", "gormes auth add <provider> --type oauth"},
		},
		{
			name: "onboard",
			args: []string{"onboard", "--json"},
			want: []string{"unknown command \"onboard\"", "gormes setup", "gormes doctor --offline --target terminal --json"},
		},
		{
			name: "oneshot long",
			args: []string{"--oneshot", "hello"},
			want: []string{"unknown flag: --oneshot", "gormes chat -q \"hello\""},
		},
		{
			name: "oneshot short",
			args: []string{"-z", "hello"},
			want: []string{"unknown shorthand flag: -z", "gormes chat -q \"hello\""},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, tc.args...)
			if err == nil {
				t.Fatalf("%v error = nil\nstdout=%s\nstderr=%s", tc.args, stdout, stderr)
			}
			combined := stdout + stderr + err.Error()
			for _, want := range tc.want {
				if !strings.Contains(combined, want) {
					t.Fatalf("%v output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", tc.args, want, stdout, stderr, err)
				}
			}
			for _, forbidden := range []string{"Deprecated", "deprecated", "plain-secret-provider"} {
				if strings.Contains(combined, forbidden) {
					t.Fatalf("%v output leaked %q:\nstdout=%s\nstderr=%s\nerr=%v", tc.args, forbidden, stdout, stderr, err)
				}
			}
		})
	}
}
