package channels

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSlackCommandUsesInjectedManifestOptions(t *testing.T) {
	var got SlackManifestOptions
	cmd := NewSlackCommandWithSeams(SlackCommandSeams{
		Manifest: func(_ *cobra.Command, opts SlackManifestOptions) error {
			got = opts
			return nil
		},
	})
	cmd.SetArgs([]string{"manifest", "--name", "Ops Bot", "--description", "Incident helper", "--slashes-only", "--write=/tmp/slack.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("slack manifest: %v", err)
	}
	if got.BotName != "Ops Bot" || got.Description != "Incident helper" || !got.SlashesOnly {
		t.Fatalf("manifest options = %+v, want name/description/slashes-only", got)
	}
	if !got.WriteChanged || got.WriteTarget != "/tmp/slack.json" {
		t.Fatalf("write options = changed %v target %q, want true /tmp/slack.json", got.WriteChanged, got.WriteTarget)
	}
}

func TestSlackManifestWriteNoOptUsesDefaultSentinel(t *testing.T) {
	var got SlackManifestOptions
	cmd := NewSlackCommandWithSeams(SlackCommandSeams{
		Manifest: func(_ *cobra.Command, opts SlackManifestOptions) error {
			got = opts
			return nil
		},
	})
	cmd.SetArgs([]string{"manifest", "--write"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("slack manifest --write: %v", err)
	}
	if !got.WriteChanged || got.WriteTarget != SlackManifestDefaultWrite {
		t.Fatalf("write options = changed %v target %q, want sentinel %q", got.WriteChanged, got.WriteTarget, SlackManifestDefaultWrite)
	}
}
