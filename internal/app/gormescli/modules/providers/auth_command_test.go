package providers

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
)

func TestAuthCommandRoutesSubcommandsThroughInjectedSeams(t *testing.T) {
	var added AuthAddOptions
	cmd := NewAuthCommandWithSeams(AuthSeams{
		RunBare: func(*cobra.Command) error {
			t.Fatal("bare auth must not run for auth add")
			return nil
		},
		RunAdd: func(_ *cobra.Command, opts AuthAddOptions) error {
			added = opts
			return nil
		},
		EmitJSONSubcommandRequired: func(*cobra.Command) error {
			t.Fatal("auth --json handler must not run for auth add")
			return nil
		},
	}, Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"add", "openrouter", "--type", "api-key", "--label", "main", "--api-key", "secret", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth add: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if added.Provider != "openrouter" || added.AuthType != "api-key" || added.Label != "main" || added.APIKey != "secret" {
		t.Fatalf("added = %+v, want parsed auth add options", added)
	}
}

func TestAuthCommandStatusMissingProviderUsesInjectedJSONError(t *testing.T) {
	cmd := NewAuthCommandWithSeams(AuthSeams{
		EmitJSONInputError: func(_ *cobra.Command, action, message string) error {
			if action != "missing_argument" || !strings.Contains(message, "<provider>") {
				t.Fatalf("json input error = %q/%q, want missing provider", action, message)
			}
			return errors.New("json-input-error")
		},
	}, Options{})
	cmd.SetArgs([]string{"status", "--json"})

	err := cmd.Execute()
	if err == nil || err.Error() != "json-input-error" {
		t.Fatalf("auth status --json error = %v, want injected json-input-error", err)
	}
}
