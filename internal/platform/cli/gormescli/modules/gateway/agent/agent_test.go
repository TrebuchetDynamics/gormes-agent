package agent

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAgentCommandUsesInjectedResetOptions(t *testing.T) {
	var got AgentResetOptions
	cmd := NewAgentCommandWithSeams(AgentCommandSeams{
		Reset: func(_ *cobra.Command, opts AgentResetOptions) error {
			got = opts
			return nil
		},
	}, AgentCommandOptions{DefaultResetTarget: "/default/gormes"})
	cmd.SetArgs([]string{"reset", "--target", "/tmp/gormes-agent", "--force", "--dry-run", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent reset: %v", err)
	}
	if got.Target != "/tmp/gormes-agent" || !got.Force || !got.DryRun || !got.JSON {
		t.Fatalf("reset options = %+v, want target/force/dry-run/json", got)
	}
}

func TestAgentCommandUsesInjectedBindingOptions(t *testing.T) {
	var gotAgentID string
	var gotMatch AgentBindingMatch
	var gotJSON bool
	cmd := NewAgentCommandWithSeams(AgentCommandSeams{
		Bind: func(_ *cobra.Command, agentID string, match AgentBindingMatch, asJSON bool) error {
			gotAgentID = agentID
			gotMatch = match
			gotJSON = asJSON
			return nil
		},
	}, AgentCommandOptions{})
	cmd.SetArgs([]string{"bind", "research", "--channel", "telegram", "--peer-kind", "direct", "--peer-id", "42", "--thread-id", "topic-a", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent bind: %v", err)
	}
	if gotAgentID != "research" || !gotJSON {
		t.Fatalf("bind agent/json = %q/%v, want research/true", gotAgentID, gotJSON)
	}
	want := AgentBindingMatch{Channel: "telegram", PeerKind: "direct", PeerID: "42", ThreadID: "topic-a"}
	if gotMatch != want {
		t.Fatalf("bind match = %+v, want %+v", gotMatch, want)
	}
}

func TestAgentCommandExposesExpectedSubcommands(t *testing.T) {
	cmd := NewAgentCommandWithSeams(AgentCommandSeams{}, AgentCommandOptions{})
	for _, name := range []string{"reset", "spawn", "list", "bind", "unbind", "inspect"} {
		if child, _, err := cmd.Find([]string{name}); err != nil || child == nil || child.Name() != name {
			t.Fatalf("agent subcommand %q missing: child=%v err=%v", name, child, err)
		}
	}
}
