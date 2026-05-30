package providerstatus

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

const (
	AuthStatusLoggedIn  = "logged_in"
	AuthStatusLoggedOut = "logged_out"
	AuthStatusError     = "error"
)

type AuthStatusOptions struct {
	HermesHome       string
	AnthropicStatus  func(context.Context) (config.AnthropicAuthStatus, error)
	CodexStatus      func() (config.CodexOAuthAuthStatus, error)
	GoogleCLIStatus  func() (config.GoogleOAuthAuthStatus, error)
	CredentialLoader func(provider string) (config.CredentialPoolStatus, config.CredentialPoolEvidence, error)
}

type ProviderAuthStatus struct {
	Provider      string
	AuthType      string
	Status        string
	Reason        string
	Authenticated bool
	Redacted      bool
	Credentials   []config.RedactedCredentialStatus
}

func RenderAuthStatus(ctx context.Context, out io.Writer, providerInput string, opts AuthStatusOptions) (ProviderAuthStatus, error) {
	status, err := ResolveAuthStatus(ctx, providerInput, opts)
	if err != nil {
		return ProviderAuthStatus{}, err
	}
	if out != nil {
		fmt.Fprintf(out, "auth_status provider=%s status=%s auth_type=%s reason=%s credentials=%d redacted=%t\n", status.Provider, status.Status, status.AuthType, status.Reason, len(status.Credentials), status.Redacted)
		for i, entry := range status.Credentials {
			entryStatus := textvalue.FirstNonEmptyTrimmed(entry.LastStatus, config.CredentialStatusOK)
			reason := textvalue.FirstNonEmptyTrimmed(entry.LastErrorReason, "-")
			fmt.Fprintf(out, "  %d. id=%s label=%s auth_type=%s source=%s status=%s reason=%s redacted=%t\n", i+1, entry.ID, entry.Label, entry.AuthType, displayAuthStatusSource(entry.Source), entryStatus, reason, entry.SecretsRedacted)
		}
	}
	return status, nil
}

func ResolveAuthStatus(ctx context.Context, providerInput string, opts AuthStatusOptions) (ProviderAuthStatus, error) {
	provider := strings.TrimSpace(strings.ToLower(providerInput))
	if provider == "" {
		return ProviderAuthStatus{}, fmt.Errorf("auth status: provider is required")
	}
	entry, ok := llm.ResolveProviderManifestEntry(provider)
	if !ok {
		return ProviderAuthStatus{}, fmt.Errorf("auth status: unknown provider %q", providerInput)
	}
	provider = entry.ID
	switch provider {
	case config.CodexOAuthProvider:
		return resolveCodexStatus(opts)
	case config.AnthropicProvider:
		return resolveAnthropicStatus(ctx, opts)
	case "google-gemini-cli":
		return resolveGoogleCLIStatus(opts)
	default:
		return resolveCredentialPoolStatus(provider, entry.AuthType, opts)
	}
}

func resolveCredentialPoolStatus(provider, authType string, opts AuthStatusOptions) (ProviderAuthStatus, error) {
	loader := opts.CredentialLoader
	if loader == nil {
		loader = func(provider string) (config.CredentialPoolStatus, config.CredentialPoolEvidence, error) {
			pool, evidence, err := config.LoadCredentialPool(config.CredentialPoolOptions{HermesHome: opts.HermesHome, Provider: provider})
			if err != nil {
				return config.CredentialPoolStatus{}, evidence, err
			}
			return pool.RedactedStatus(), evidence, nil
		}
	}
	poolStatus, evidence, err := loader(provider)
	if err != nil {
		return ProviderAuthStatus{Provider: provider, AuthType: authType, Status: AuthStatusError, Reason: textvalue.FirstNonEmptyTrimmed(evidence.Code, config.CredentialPoolEvidenceCorrupt), Redacted: true}, err
	}
	if poolStatus.Count == 0 {
		return ProviderAuthStatus{Provider: provider, AuthType: authType, Status: AuthStatusLoggedOut, Reason: textvalue.FirstNonEmptyTrimmed(evidence.Code, config.CredentialPoolEvidenceEmpty), Redacted: true}, nil
	}
	return ProviderAuthStatus{Provider: provider, AuthType: authType, Status: AuthStatusLoggedIn, Authenticated: true, Reason: config.CredentialPoolEvidenceLoaded, Redacted: true, Credentials: poolStatus.Entries}, nil
}

func resolveCodexStatus(opts AuthStatusOptions) (ProviderAuthStatus, error) {
	check := opts.CodexStatus
	if check == nil {
		store := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{HermesHome: opts.HermesHome})
		check = store.CheckAuth
	}
	status, err := check()
	if err != nil {
		return ProviderAuthStatus{Provider: config.CodexOAuthProvider, AuthType: "oauth_external", Status: AuthStatusError, Reason: textvalue.FirstNonEmptyTrimmed(status.Code, config.CodexOAuthStatusCorrupt), Redacted: true}, err
	}
	out := ProviderAuthStatus{Provider: config.CodexOAuthProvider, AuthType: "oauth_external", Reason: textvalue.FirstNonEmptyTrimmed(status.Code, status.Evidence), Redacted: true}
	if status.Authenticated {
		out.Status = AuthStatusLoggedIn
		out.Authenticated = true
		out.Credentials = []config.RedactedCredentialStatus{{ID: status.AccountID, Label: status.Label, AuthType: config.CredentialAuthOAuth, Source: status.Source, LastStatus: config.CredentialStatusOK, SecretsRedacted: true}}
	} else {
		out.Status = AuthStatusLoggedOut
	}
	return out, nil
}

func resolveAnthropicStatus(ctx context.Context, opts AuthStatusOptions) (ProviderAuthStatus, error) {
	check := opts.AnthropicStatus
	if check == nil {
		store := config.NewAnthropicAuthStateStore(config.AnthropicAuthStateStoreOptions{})
		check = store.CheckAuth
	}
	status, err := check(ctx)
	if err != nil {
		return ProviderAuthStatus{Provider: config.AnthropicProvider, AuthType: config.CredentialAuthOAuth, Status: AuthStatusError, Reason: textvalue.FirstNonEmptyTrimmed(status.Code, config.AnthropicAuthStatusCorrupt), Redacted: true}, err
	}
	out := ProviderAuthStatus{Provider: config.AnthropicProvider, AuthType: config.CredentialAuthOAuth, Reason: textvalue.FirstNonEmptyTrimmed(status.Code, status.Evidence), Redacted: true}
	if status.Authenticated {
		out.Status = AuthStatusLoggedIn
		out.Authenticated = true
		out.Credentials = []config.RedactedCredentialStatus{{ID: config.AnthropicProvider, Label: config.AnthropicProvider, AuthType: config.CredentialAuthOAuth, Source: status.Source, LastStatus: config.CredentialStatusOK, SecretsRedacted: true}}
	} else {
		out.Status = AuthStatusLoggedOut
	}
	return out, nil
}

func resolveGoogleCLIStatus(opts AuthStatusOptions) (ProviderAuthStatus, error) {
	check := opts.GoogleCLIStatus
	if check == nil {
		return resolveCredentialPoolStatus("google-gemini-cli", "oauth_external", opts)
	}
	status, err := check()
	if err != nil {
		return ProviderAuthStatus{Provider: "google-gemini-cli", AuthType: "oauth_external", Status: AuthStatusError, Reason: textvalue.FirstNonEmptyTrimmed(status.Code, config.GoogleOAuthStatusCorrupt), Redacted: true}, err
	}
	out := ProviderAuthStatus{Provider: "google-gemini-cli", AuthType: "oauth_external", Reason: textvalue.FirstNonEmptyTrimmed(status.Code, status.Evidence), Redacted: true}
	if status.Authenticated {
		out.Status = AuthStatusLoggedIn
		out.Authenticated = true
	} else {
		out.Status = AuthStatusLoggedOut
	}
	return out, nil
}

func displayAuthStatusSource(source string) string {
	return strings.TrimPrefix(strings.TrimSpace(source), "manual:")
}
