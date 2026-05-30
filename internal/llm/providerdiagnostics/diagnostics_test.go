package providerdiagnostics

import (
	"strings"
	"testing"
)

func TestBuildRedactsSecretsAndKeepsOperatorEvidence(t *testing.T) {
	diag := Build(Input{
		Provider:         " openrouter ",
		Model:            "anthropic/claude",
		CredentialSource: "env:OPENROUTER_API_KEY=sk-secret-value",
		Classification: Classification{
			Kind:      "auth",
			Class:     "fatal",
			Status:    401,
			Message:   "Bearer sk-secret-value failed",
			Retryable: false,
		},
	})

	if diag.Provider != "openrouter" || diag.Model != "anthropic/claude" {
		t.Fatalf("identity = %+v, want provider/model evidence", diag)
	}
	if !strings.Contains(diag.CredentialSource, "OPENROUTER_API_KEY=") {
		t.Fatalf("credential source = %q, want source name", diag.CredentialSource)
	}
	rendered := diag.CredentialSource + " " + diag.Message
	if strings.Contains(rendered, "sk-secret-value") {
		t.Fatalf("diagnostic leaked secret: %+v", diag)
	}
	if diag.NextAction != "refresh_or_replace_provider_credential" {
		t.Fatalf("NextAction = %q, want auth repair action", diag.NextAction)
	}
	if !diag.Redacted {
		t.Fatalf("Redacted = false, want true")
	}
}

func TestNextActionFallbacks(t *testing.T) {
	if got := NextAction(Classification{Kind: "timeout"}); got != "retry_with_bounded_backoff" {
		t.Fatalf("timeout action = %q", got)
	}
	if got := NextAction(Classification{Class: "fatal"}); got != "inspect_provider_auth_or_request" {
		t.Fatalf("fatal fallback action = %q", got)
	}
	if got := NextAction(Classification{}); got != "inspect_provider_diagnostics" {
		t.Fatalf("unknown fallback action = %q", got)
	}
}
