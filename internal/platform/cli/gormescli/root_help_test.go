package gormescli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootHelpGuidesFirstRunOperators(t *testing.T) {
	cmd := newRootHelpCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "--help")
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
	for _, topic := range []string{"login", "onboard"} {
		t.Run(topic, func(t *testing.T) {
			cmd := newRootHelpCommandForTest()
			stdout, stderr, err := executeRootCommandForTest(cmd, "help", topic)
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

func newRootHelpCommandForTest() *cobra.Command {
	return NewRootCommand(RootOptions{
		Version: Version,
		Finalizers: []func(*cobra.Command){
			func(cmd *cobra.Command) {
				InstallVisibleHelpCommand(cmd, VisibleHelpCommandOptions{ExitCodeError: NewExitCodeError})
			},
			InstallRootHelpRenderer,
		},
	}, stubRootFactories())
}
