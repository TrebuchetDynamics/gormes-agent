package providers

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestUsageCommandFacadeBuildsCommand(t *testing.T) {
	cmd := NewUsageCommand(Options{})
	if cmd.Use != "usage" {
		t.Fatalf("Use = %q, want usage", cmd.Use)
	}
	for _, name := range []string{"provider", "api-key", "base-url", "account-id", "json"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("usage command missing --%s flag", name)
		}
	}
}

func TestRunUsageCommandFacadeAcceptsInvocationType(t *testing.T) {
	var _ = RunUsageCommand
	var _ = UsageInvocation{Provider: "openrouter"}
	var _ *cobra.Command
}
