package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const openRouterBaseURL = "https://openrouter.ai/api/v1"

var (
	authBareAWSIdentityProbe          func() (string, error)
	errAuthBareAWSIdentityUnavailable = errors.New("aws_identity_unavailable")
)

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "auth",
		Short:        "Manage Hermes-compatible provider credentials",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAuthBareCommand(cmd)
		},
	}
	cmd.AddCommand(newAuthAddCommand())
	cmd.AddCommand(newAuthListCommand())
	cmd.AddCommand(newAuthRemoveCommand())
	cmd.AddCommand(newAuthResetCommand())
	cmd.AddCommand(newAuthStatusCommand())
	cmd.AddCommand(newAuthLogoutCommand())
	return cmd
}

func newAuthAddCommand() *cobra.Command {
	var authType string
	var label string
	var apiKey string
	var inferenceURL string
	var portalURL string
	var clientID string
	var scope string
	var noBrowser bool
	var timeout string
	var insecure bool
	var caBundle string
	var emergencyImportFromCodexCLI string

	cmd := &cobra.Command{
		Use:   "add <provider>",
		Short: "Add a provider credential to the Hermes-compatible credential pool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthAddCommand(cmd, authAddOptions{
				Provider:                    args[0],
				AuthType:                    authType,
				Label:                       label,
				APIKey:                      apiKey,
				InferenceURL:                inferenceURL,
				PortalURL:                   portalURL,
				ClientID:                    clientID,
				Scope:                       scope,
				NoBrowser:                   noBrowser,
				Timeout:                     timeout,
				Insecure:                    insecure,
				CABundle:                    caBundle,
				EmergencyImportFromCodexCLI: emergencyImportFromCodexCLI,
			})
		},
	}
	cmd.Flags().StringVar(&authType, "type", "", "credential type: api-key, api_key, or oauth")
	cmd.Flags().StringVar(&label, "label", "", "credential label")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key to store; omitted values are not echoed")
	cmd.Flags().StringVar(&inferenceURL, "inference-url", "", "provider inference base URL override")
	cmd.Flags().StringVar(&portalURL, "portal-url", "", "OAuth portal URL")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&scope, "scope", "", "OAuth scope")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open a browser for OAuth")
	cmd.Flags().StringVar(&timeout, "timeout", "", "OAuth timeout")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "disable OAuth TLS verification")
	cmd.Flags().StringVar(&caBundle, "ca-bundle", "", "OAuth CA bundle")
	cmd.Flags().StringVar(&emergencyImportFromCodexCLI, "emergency-import-from-codex-cli", "", "explicitly import Codex CLI auth.json after accepting the refresh-token race envelope")
	return cmd
}

func newAuthListCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "list [provider]",
		Short:        "List provider credentials with secrets redacted",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := ""
			if len(args) > 0 {
				provider = args[0]
			}
			return runAuthListCommand(cmd, provider, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, provider, credentials: [...]}` JSON document with the same redacted fields the human row prints (suitable for fleet credential-health monitoring)")
	return cmd
}

func newAuthRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "remove <provider> <target>",
		Short:        "Remove a provider credential by index, id, or label",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthRemoveCommand(cmd, args[0], args[1])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action, provider, removed: {id, label}, redacted}`")
	return cmd
}

func newAuthResetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "reset <provider>",
		Short:        "Reset provider credential cooldown/exhaustion state",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthResetCommand(cmd, args[0])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action, provider, count, redacted}`")
	return cmd
}

func newAuthStatusCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "status <provider>",
		Short:        "Show redacted provider auth status",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthStatusCommand(cmd, args[0], asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a machine-readable JSON document with the same redacted fields (suitable for credential-health monitoring)")
	return cmd
}

func newAuthLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "logout <provider>",
		Short:        "Clear provider credentials",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthLogoutCommand(cmd, args[0])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action: 'logged_out'|'absent', provider, redacted}`")
	return cmd
}

// authLifecycleReportJSON is the wire shape for `auth remove --json`,
// `auth reset --json`, and `auth logout --json`. Fleet credential
// rotation/cleanup automation parses this to confirm what changed
// per machine. Raw secrets are NEVER present — `redacted: true` is
// always emitted as a guarantee.
type authLifecycleReportJSON struct {
	Build    buildProvenanceJSON `json:"build"`
	Action   string              `json:"action"`
	Provider string              `json:"provider"`
	Count    int                 `json:"count,omitempty"`
	Removed  *authRemovedJSON    `json:"removed,omitempty"`
	Redacted bool                `json:"redacted"`
}

type authRemovedJSON struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type authAddOptions struct {
	Provider                    string
	AuthType                    string
	Label                       string
	APIKey                      string
	InferenceURL                string
	PortalURL                   string
	ClientID                    string
	Scope                       string
	NoBrowser                   bool
	Timeout                     string
	Insecure                    bool
	CABundle                    string
	EmergencyImportFromCodexCLI string
}

type anthropicOAuthLoginRequest struct {
	Label string
	Out   interface{ Write([]byte) (int, error) }
}

type anthropicOAuthTokens struct {
	AccountID    string
	Label        string
	AccessToken  string
	RefreshToken string
	BaseURL      string
	ExpiresAtMS  int64
	Source       string
}

var authAnthropicOAuthLogin func(context.Context, anthropicOAuthLoginRequest) (anthropicOAuthTokens, error)

func runAnthropicOAuthLoginUnavailable(context.Context, anthropicOAuthLoginRequest) (anthropicOAuthTokens, error) {
	return anthropicOAuthTokens{}, errors.New("native Anthropic Hermes PKCE login is unavailable in this slice; inject OAuth login seam or import Claude credentials first")
}

type nousOAuthLoginRequest struct {
	Label            string
	PortalBaseURL    string
	InferenceBaseURL string
	ClientID         string
	Scope            string
	NoBrowser        bool
	Timeout          string
	Insecure         bool
	CABundle         string
	Out              interface{ Write([]byte) (int, error) }
}

type nousOAuthTokens = config.NousOAuthCredentials

var authNousOAuthLogin func(context.Context, nousOAuthLoginRequest) (nousOAuthTokens, error)

func runNousOAuthLoginUnavailable(context.Context, nousOAuthLoginRequest) (nousOAuthTokens, error) {
	return nousOAuthTokens{}, errors.New("native Nous device-code OAuth login is unavailable in this slice; inject OAuth login seam before calling provider endpoints")
}

type googleGeminiOAuthLoginRequest struct {
	Label string
	Out   interface{ Write([]byte) (int, error) }
}

type googleGeminiOAuthTokens struct {
	AccountID    string
	Label        string
	Email        string
	AccessToken  string
	RefreshToken string
	BaseURL      string
	ExpiresAtMS  int64
	Source       string
}

var authGoogleGeminiOAuthLogin func(context.Context, googleGeminiOAuthLoginRequest) (googleGeminiOAuthTokens, error)

func runGoogleGeminiOAuthLoginUnavailable(context.Context, googleGeminiOAuthLoginRequest) (googleGeminiOAuthTokens, error) {
	return googleGeminiOAuthTokens{}, errors.New("native Google Gemini CLI PKCE login is unavailable in this slice; inject OAuth login seam before calling Google endpoints")
}

type qwenOAuthImportRequest struct {
	Label string
	Out   interface{ Write([]byte) (int, error) }
}

type qwenOAuthTokens struct {
	AccountID    string
	Label        string
	AccessToken  string
	RefreshToken string
	BaseURL      string
	ExpiresAtMS  int64
	Source       string
}

var authQwenOAuthImport func(context.Context, qwenOAuthImportRequest) (qwenOAuthTokens, error)

var (
	errQwenCLIAuthMissing   = errors.New("qwen_cli_auth_missing")
	errQwenCLIRefreshFailed = errors.New("qwen_cli_refresh_failed")
)

func runQwenOAuthImportUnavailable(context.Context, qwenOAuthImportRequest) (qwenOAuthTokens, error) {
	return qwenOAuthTokens{}, errQwenCLIAuthMissing
}

func runAuthBareCommand(cmd *cobra.Command) error {
	providers, err := config.ListCredentialPoolProviders(config.CredentialPoolOptions{})
	if err != nil {
		return fmt.Errorf("gormes auth: credential_pool_corrupt")
	}
	printed := false
	for _, provider := range providers {
		pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
		if err != nil {
			continue
		}
		status := pool.RedactedStatus()
		if status.Count == 0 {
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s (%d credentials): credential_pool provider=%s credentials=%d redacted=%t\n", provider, status.Count, provider, status.Count, status.Redacted)
		fmt.Fprintln(cmd.OutOrStdout(), "label\tauth_type\tsource\tstatus\tcurrent")
		currentMarked := false
		for _, entry := range status.Entries {
			current := ""
			if !currentMarked && entry.LastStatus != config.CredentialStatusExhausted {
				current = "←"
				currentMarked = true
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n", entry.Label, entry.AuthType, entry.Source, bareCredentialStatus(entry), current)
		}
		printed = true
	}
	if !printed {
		fmt.Fprintln(cmd.OutOrStdout(), "(no credentials configured) credential_pool_empty provider=all redacted=true")
	}
	fmt.Fprintln(cmd.OutOrStdout(), bareAWSIdentityLine())
	return nil
}

func bareAWSIdentityLine() string {
	if authBareAWSIdentityProbe == nil {
		return "bedrock_identity status=not_checked redacted=true"
	}
	identity, err := authBareAWSIdentityProbe()
	if err != nil || strings.TrimSpace(identity) == "" {
		return "bedrock identity: aws_identity_unavailable bedrock_identity status=aws_identity_unavailable redacted=true"
	}
	return fmt.Sprintf("bedrock identity: %s bedrock_identity status=ok redacted=true", strings.TrimSpace(identity))
}

func bareCredentialStatus(entry config.RedactedCredentialStatus) string {
	status := strings.TrimSpace(entry.LastStatus)
	if status == "" {
		status = config.CredentialStatusOK
	}
	if status == config.CredentialStatusExhausted && strings.TrimSpace(entry.LastErrorReason) != "" {
		return fmt.Sprintf("%s(%s)", status, entry.LastErrorReason)
	}
	return status
}

func runAuthAddCommand(cmd *cobra.Command, opts authAddOptions) error {
	provider := normalizeAuthProvider(opts.Provider)
	if provider == "" {
		return errors.New("gormes auth add: provider is required")
	}
	if provider == "bedrock" {
		return errors.New("gormes auth add bedrock: bedrock_use_aws_sdk_chain; configure AWS credentials through the AWS credential chain (env vars, shared profile, SSO, or role credentials) and use bare `gormes auth` for redacted Bedrock identity status")
	}
	authType := normalizeAuthType(opts.AuthType, provider)
	if authType == config.CredentialAuthOAuth {
		switch provider {
		case config.AnthropicProvider:
			return runAuthAddAnthropicOAuthCommand(cmd, opts)
		case config.NousOAuthProvider:
			return runAuthAddNousOAuthCommand(cmd, opts)
		case config.CodexOAuthProvider:
			return runAuthAddCodexOAuthCommand(cmd, opts)
		case "google-gemini-cli":
			return runAuthAddGoogleGeminiOAuthCommand(cmd, opts)
		case "qwen-oauth":
			return runAuthAddQwenOAuthCommand(cmd, opts)
		default:
			return fmt.Errorf("gormes auth add %s --type oauth: provider OAuth adapters are planned; use --type api-key for API-key providers", provider)
		}
	}
	if authType != config.CredentialAuthAPIKey {
		return fmt.Errorf("gormes auth add %s: unsupported auth type %q", provider, opts.AuthType)
	}
	apiKey := strings.TrimSpace(opts.APIKey)
	if apiKey == "" {
		return fmt.Errorf("gormes auth add %s: auth_api_key_missing", provider)
	}
	baseURL := providerBaseURL(provider, opts.InferenceURL)
	if baseURL == "" {
		return fmt.Errorf("gormes auth add %s: inference URL is required for this provider", provider)
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return fmt.Errorf("gormes auth add %s: credential_pool_corrupt", provider)
	}
	entries := pool.Entries()
	id := nextCredentialID(provider, entries)
	label := strings.TrimSpace(opts.Label)
	if label == "" {
		label = id
	}
	entries = append(entries, config.PooledCredential{
		ID:               id,
		Label:            label,
		AuthType:         config.CredentialAuthAPIKey,
		Source:           "manual",
		AccessToken:      apiKey,
		BaseURL:          baseURL,
		InferenceBaseURL: baseURL,
		LastStatus:       config.CredentialStatusOK,
	})
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, entries); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_api_key_saved provider=%s id=%s label=%s redacted=true\n", provider, id, label)
	return nil
}

func runAuthAddAnthropicOAuthCommand(cmd *cobra.Command, opts authAddOptions) error {
	login := authAnthropicOAuthLogin
	if login == nil {
		login = runAnthropicOAuthLoginUnavailable
	}
	tokens, err := login(context.Background(), anthropicOAuthLoginRequest{
		Label: strings.TrimSpace(opts.Label),
		Out:   cmd.OutOrStdout(),
	})
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: anthropic_oauth_failed: %s", config.AnthropicProvider, sanitizeAuthCommandError(err.Error()))
	}
	tokens.Label = firstNonEmpty(strings.TrimSpace(opts.Label), strings.TrimSpace(tokens.Label), labelFromOAuthToken(tokens.AccessToken, "anthropic-oauth"))
	tokens.AccountID = firstNonEmpty(strings.TrimSpace(tokens.AccountID), nextCredentialID(config.AnthropicProvider, nil))
	tokens.Source = firstNonEmpty(strings.TrimSpace(tokens.Source), "hermes_pkce")
	tokens.BaseURL = firstNonEmpty(strings.TrimRight(strings.TrimSpace(tokens.BaseURL), "/"), providerBaseURL(config.AnthropicProvider, ""))
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return fmt.Errorf("gormes auth add %s --type oauth: anthropic_oauth_failed: oauth_access_token_missing", config.AnthropicProvider)
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: config.AnthropicProvider})
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: credential_pool_corrupt", config.AnthropicProvider)
	}
	entries := pool.Entries()
	entry := config.PooledCredential{
		ID:               tokens.AccountID,
		Label:            tokens.Label,
		AuthType:         config.CredentialAuthOAuth,
		Source:           "manual:" + tokens.Source,
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		BaseURL:          tokens.BaseURL,
		InferenceBaseURL: tokens.BaseURL,
		ExpiresAtMS:      tokens.ExpiresAtMS,
		LastStatus:       config.CredentialStatusOK,
	}
	entries = append(entries, entry)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.AnthropicProvider}, entries); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_oauth_saved provider=%s account_id=%s label=%s source=%s redacted=true\n", config.AnthropicProvider, entry.ID, entry.Label, tokens.Source)
	return nil
}

func runAuthAddNousOAuthCommand(cmd *cobra.Command, opts authAddOptions) error {
	login := authNousOAuthLogin
	if login == nil {
		login = runNousOAuthLoginUnavailable
	}
	tokens, err := login(context.Background(), nousOAuthLoginRequest{
		Label:            strings.TrimSpace(opts.Label),
		PortalBaseURL:    strings.TrimRight(strings.TrimSpace(opts.PortalURL), "/"),
		InferenceBaseURL: strings.TrimRight(strings.TrimSpace(opts.InferenceURL), "/"),
		ClientID:         strings.TrimSpace(opts.ClientID),
		Scope:            strings.TrimSpace(opts.Scope),
		NoBrowser:        opts.NoBrowser,
		Timeout:          strings.TrimSpace(opts.Timeout),
		Insecure:         opts.Insecure,
		CABundle:         strings.TrimSpace(opts.CABundle),
		Out:              cmd.OutOrStdout(),
	})
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: nous_device_code_failed: %s", config.NousOAuthProvider, sanitizeAuthCommandError(err.Error()))
	}
	tokens.Label = firstNonEmpty(strings.TrimSpace(opts.Label), strings.TrimSpace(tokens.Label), labelFromOAuthToken(tokens.AccessToken, "Nous device code"))
	tokens.PortalBaseURL = firstNonEmpty(strings.TrimRight(strings.TrimSpace(tokens.PortalBaseURL), "/"), strings.TrimRight(strings.TrimSpace(opts.PortalURL), "/"), "https://portal.nousresearch.com")
	tokens.InferenceBaseURL = firstNonEmpty(strings.TrimRight(strings.TrimSpace(tokens.InferenceBaseURL), "/"), providerBaseURL(config.NousOAuthProvider, opts.InferenceURL))
	tokens.TokenType = firstNonEmpty(strings.TrimSpace(tokens.TokenType), "Bearer")
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return fmt.Errorf("gormes auth add %s --type oauth: nous_device_code_failed: oauth_access_token_missing", config.NousOAuthProvider)
	}
	entry, err := config.SaveNousOAuthCredentials(config.CredentialPoolOptions{Provider: config.NousOAuthProvider}, tokens)
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: credential_pool_corrupt", config.NousOAuthProvider)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_oauth_saved provider=%s account_id=%s label=%s source=%s redacted=true\n", config.NousOAuthProvider, entry.ID, entry.Label, entry.Source)
	return nil
}

func runAuthAddGoogleGeminiOAuthCommand(cmd *cobra.Command, opts authAddOptions) error {
	const provider = "google-gemini-cli"
	login := authGoogleGeminiOAuthLogin
	if login == nil {
		login = runGoogleGeminiOAuthLoginUnavailable
	}
	tokens, err := login(context.Background(), googleGeminiOAuthLoginRequest{
		Label: strings.TrimSpace(opts.Label),
		Out:   cmd.OutOrStdout(),
	})
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: google_gemini_oauth_failed: %s", provider, sanitizeAuthCommandError(err.Error()))
	}
	tokens.Label = firstNonEmpty(strings.TrimSpace(opts.Label), strings.TrimSpace(tokens.Label), strings.TrimSpace(tokens.Email), "Google Gemini CLI")
	tokens.Source = firstNonEmpty(strings.TrimSpace(tokens.Source), "google_pkce")
	tokens.BaseURL = firstNonEmpty(strings.TrimRight(strings.TrimSpace(tokens.BaseURL), "/"), providerBaseURL(provider, ""))
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return fmt.Errorf("gormes auth add %s --type oauth: google_gemini_oauth_failed: oauth_access_token_missing", provider)
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: credential_pool_corrupt", provider)
	}
	entries := pool.Entries()
	id := firstNonEmpty(strings.TrimSpace(tokens.AccountID), nextCredentialID(provider, entries))
	entry := config.PooledCredential{
		ID:               id,
		Label:            tokens.Label,
		AuthType:         config.CredentialAuthOAuth,
		Source:           "manual:" + tokens.Source,
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		BaseURL:          tokens.BaseURL,
		InferenceBaseURL: tokens.BaseURL,
		ExpiresAtMS:      tokens.ExpiresAtMS,
		LastStatus:       config.CredentialStatusOK,
	}
	entries = append(entries, entry)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, entries); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_oauth_saved provider=%s account_id=%s label=%s source=%s redacted=true\n", provider, entry.ID, entry.Label, tokens.Source)
	return nil
}

func runAuthAddQwenOAuthCommand(cmd *cobra.Command, opts authAddOptions) error {
	const provider = "qwen-oauth"
	importer := authQwenOAuthImport
	if importer == nil {
		importer = runQwenOAuthImportUnavailable
	}
	tokens, err := importer(context.Background(), qwenOAuthImportRequest{
		Label: strings.TrimSpace(opts.Label),
		Out:   cmd.OutOrStdout(),
	})
	if err != nil {
		code := "qwen_cli_auth_missing"
		if errors.Is(err, errQwenCLIRefreshFailed) {
			code = "qwen_cli_refresh_failed"
		}
		return fmt.Errorf("gormes auth add %s --type oauth: %s: %s", provider, code, sanitizeAuthCommandError(err.Error()))
	}
	tokens.Label = firstNonEmpty(strings.TrimSpace(opts.Label), strings.TrimSpace(tokens.Label), "Qwen CLI")
	tokens.Source = firstNonEmpty(strings.TrimSpace(tokens.Source), "qwen_cli")
	tokens.BaseURL = firstNonEmpty(strings.TrimRight(strings.TrimSpace(tokens.BaseURL), "/"), providerBaseURL(provider, opts.InferenceURL))
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return fmt.Errorf("gormes auth add %s --type oauth: qwen_cli_auth_missing: oauth_access_token_missing", provider)
	}
	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: credential_pool_corrupt", provider)
	}
	entries := pool.Entries()
	id := firstNonEmpty(strings.TrimSpace(tokens.AccountID), nextCredentialID(provider, entries))
	entry := config.PooledCredential{
		ID:               id,
		Label:            tokens.Label,
		AuthType:         config.CredentialAuthOAuth,
		Source:           "manual:" + tokens.Source,
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		BaseURL:          tokens.BaseURL,
		InferenceBaseURL: tokens.BaseURL,
		ExpiresAtMS:      tokens.ExpiresAtMS,
		LastStatus:       config.CredentialStatusOK,
	}
	entries = append(entries, entry)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, entries); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_oauth_saved provider=%s account_id=%s label=%s source=%s redacted=true\n", provider, entry.ID, entry.Label, tokens.Source)
	return nil
}

func runAuthAddCodexOAuthCommand(cmd *cobra.Command, opts authAddOptions) error {
	if importPath := strings.TrimSpace(opts.EmergencyImportFromCodexCLI); importPath != "" {
		return runAuthAddCodexEmergencyImportCommand(cmd, opts, importPath)
	}
	if importPath := defaultCodexCLIAuthPath(); importPath != "" {
		status, err := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{}).ImportCodexCLITokens(config.CodexCLIImportRequest{
			AuthPath:  importPath,
			Explicit:  true,
			Label:     strings.TrimSpace(opts.Label),
			BaseURL:   providerBaseURL(config.CodexOAuthProvider, opts.InferenceURL),
			AccountID: "",
		})
		if err != nil {
			return fmt.Errorf("gormes auth add %s --type oauth: credential_pool_corrupt", config.CodexOAuthProvider)
		}
		if status.Code == config.CodexOAuthStatusAuthorized {
			fmt.Fprintln(cmd.OutOrStdout(), "Found existing Codex CLI credentials.")
			fmt.Fprintf(cmd.OutOrStdout(), "auth_oauth_saved provider=%s account_id=%s label=%s source=%s redacted=true\n", config.CodexOAuthProvider, status.AccountID, status.Label, status.Source)
			fmt.Fprintln(cmd.OutOrStdout(), "Credentials imported into the Gormes credential pool; future Codex CLI or VS Code refreshes do not rotate Gormes tokens.")
			return nil
		}
	}
	login := authCodexOAuthLogin
	if login == nil {
		login = runCodexDeviceCodeLogin
	}
	tokens, err := login(context.Background(), codexOAuthLoginRequest{
		Label: strings.TrimSpace(opts.Label),
		Out:   cmd.OutOrStdout(),
	})
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: codex_device_code_failed: %s", config.CodexOAuthProvider, sanitizeAuthCommandError(err.Error()))
	}
	if strings.TrimSpace(opts.Label) != "" && strings.TrimSpace(tokens.Label) == "" {
		tokens.Label = strings.TrimSpace(opts.Label)
	}
	status, err := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{}).SaveTokens(tokens)
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: credential_pool_corrupt", config.CodexOAuthProvider)
	}
	if status.Code != config.CodexOAuthStatusAuthorized {
		return fmt.Errorf("gormes auth add %s --type oauth: %s", config.CodexOAuthProvider, status.Code)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_oauth_saved provider=%s account_id=%s label=%s source=%s redacted=true\n", config.CodexOAuthProvider, status.AccountID, status.Label, status.Source)
	fmt.Fprintln(cmd.OutOrStdout(), "Hermes will keep working independently with its own session; the Codex CLI / VS Code extension cannot rotate Gormes tokens.")
	return nil
}

func defaultCodexCLIAuthPath() string {
	codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codexHome == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return ""
		}
		codexHome = filepath.Join(home, ".codex")
	}
	return filepath.Join(codexHome, "auth.json")
}

func runAuthAddCodexEmergencyImportCommand(cmd *cobra.Command, opts authAddOptions, importPath string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "Codex CLI / VS Code refresh-token race warning: this emergency import copies a vendor CLI auth.json into Gormes' independent credential pool. Continue only if you accept that future Codex CLI or VS Code refreshes may rotate their own token state without updating Gormes. redacted=true")
	status, err := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{}).ImportCodexCLITokens(config.CodexCLIImportRequest{
		AuthPath:  importPath,
		Explicit:  true,
		Label:     strings.TrimSpace(opts.Label),
		BaseURL:   providerBaseURL(config.CodexOAuthProvider, opts.InferenceURL),
		AccountID: "",
	})
	if err != nil {
		return fmt.Errorf("gormes auth add %s --type oauth: credential_pool_corrupt", config.CodexOAuthProvider)
	}
	if status.Code != config.CodexOAuthStatusAuthorized {
		code := status.Code
		if status.Evidence == config.CodexOAuthEvidenceImportExpired {
			code = "codex_emergency_import_jwt_expired"
		} else if code == config.CodexOAuthStatusImportNotRequested || code == config.CodexOAuthStatusImportRejected {
			code = "codex_external_import_blocked"
		}
		return fmt.Errorf("gormes auth add %s --type oauth: %s", config.CodexOAuthProvider, code)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_oauth_saved provider=%s account_id=%s label=%s source=%s redacted=true\n", config.CodexOAuthProvider, status.AccountID, status.Label, status.Source)
	return nil
}

func runAuthListCommand(cmd *cobra.Command, providerInput string, asJSON bool) error {
	provider := normalizeAuthProvider(providerInput)
	if provider == "" {
		if asJSON {
			return emitAuthListJSON(cmd, "all", []config.RedactedCredentialStatus{})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "credential_pool_empty provider=all redacted=true")
		return nil
	}
	pool, evidence, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return fmt.Errorf("gormes auth list %s: %s", provider, evidence.Code)
	}
	status := pool.RedactedStatus()
	if asJSON {
		return emitAuthListJSON(cmd, provider, status.Entries)
	}
	if status.Count == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "credential_pool_empty provider=%s redacted=true\n", provider)
		return nil
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(cmd.OutOrStdout(), "%s (%d credentials) credentials=%d redacted=%t\n", provider, status.Count, status.Count, status.Redacted)
	for i, entry := range status.Entries {
		statusText := entry.LastStatus
		if statusText == "" {
			statusText = config.CredentialStatusOK
		}
		reason := entry.LastErrorReason
		if reason == "" {
			reason = "-"
		}
		fmt.Fprintf(out, "  %d. id=%s label=%s auth_type=%s source=%s status=%s reason=%s redacted=%t\n", i+1, entry.ID, entry.Label, entry.AuthType, displayCredentialSource(entry.Source), statusText, reason, entry.SecretsRedacted)
	}
	return nil
}

// authListReportJSON is the wire shape for `gormes auth list --json`.
// Build provenance leads, then provider + credentials — same convention
// as update / doctor / status / restore / auth-status / gateway-status /
// secrets. Credentials carry exactly the fields the human row already
// prints, with secrets pre-redacted upstream by RedactedStatus.
type authListReportJSON struct {
	Build       buildProvenanceJSON       `json:"build"`
	Provider    string                    `json:"provider"`
	Redacted    bool                      `json:"redacted"`
	Credentials []authListCredentialJSON  `json:"credentials"`
}

type authListCredentialJSON struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	AuthType        string `json:"auth_type"`
	Source          string `json:"source"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
	SecretsRedacted bool   `json:"secrets_redacted"`
}

func emitAuthListJSON(cmd *cobra.Command, provider string, entries []config.RedactedCredentialStatus) error {
	creds := make([]authListCredentialJSON, len(entries))
	for i, e := range entries {
		statusText := e.LastStatus
		if statusText == "" {
			statusText = config.CredentialStatusOK
		}
		reason := e.LastErrorReason
		if reason == "" {
			reason = "-"
		}
		creds[i] = authListCredentialJSON{
			ID:              e.ID,
			Label:           e.Label,
			AuthType:        string(e.AuthType),
			Source:          displayCredentialSource(e.Source),
			Status:          string(statusText),
			Reason:          reason,
			SecretsRedacted: e.SecretsRedacted,
		}
	}
	body, err := json.MarshalIndent(authListReportJSON{
		Build:       newBuildProvenance(),
		Provider:    provider,
		Redacted:    true,
		Credentials: creds,
	}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

func runAuthRemoveCommand(cmd *cobra.Command, providerInput, target string) error {
	provider := normalizeAuthProvider(providerInput)
	entries, err := loadAuthEntries(provider)
	if err != nil {
		return err
	}
	idx := findCredentialIndex(entries, target)
	if idx < 0 {
		return fmt.Errorf("gormes auth remove %s: credential_not_found", provider)
	}
	removed := entries[idx]
	entries = append(entries[:idx], entries[idx+1:]...)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, entries); err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		return writeAuthLifecycleJSON(cmd.OutOrStdout(), authLifecycleReportJSON{
			Build:    newBuildProvenance(),
			Action:   "removed",
			Provider: provider,
			Removed:  &authRemovedJSON{ID: removed.ID, Label: removed.Label},
			Redacted: true,
		})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_credential_removed provider=%s id=%s label=%s redacted=true\n", provider, removed.ID, removed.Label)
	return nil
}

func runAuthResetCommand(cmd *cobra.Command, providerInput string) error {
	provider := normalizeAuthProvider(providerInput)
	entries, err := loadAuthEntries(provider)
	if err != nil {
		return err
	}
	for i := range entries {
		entries[i].LastStatus = config.CredentialStatusOK
		entries[i].LastErrorCode = 0
		entries[i].LastErrorReason = ""
		entries[i].LastErrorMessage = ""
		entries[i].LastErrorResetAt = 0
	}
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, entries); err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		return writeAuthLifecycleJSON(cmd.OutOrStdout(), authLifecycleReportJSON{
			Build:    newBuildProvenance(),
			Action:   "reset",
			Provider: provider,
			Count:    len(entries),
			Redacted: true,
		})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_status_reset provider=%s count=%d redacted=true\n", provider, len(entries))
	return nil
}

func writeAuthLifecycleJSON(out interface{ Write(p []byte) (int, error) }, report authLifecycleReportJSON) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func runAuthStatusCommand(cmd *cobra.Command, providerInput string, asJSON bool) error {
	if asJSON {
		status, err := cli.ResolveAuthStatus(context.Background(), providerInput, cli.AuthStatusOptions{})
		if err != nil {
			return err
		}
		body, err := json.MarshalIndent(authStatusToJSON(status), "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return err
	}
	_, err := cli.RenderAuthStatus(context.Background(), cmd.OutOrStdout(), providerInput, cli.AuthStatusOptions{})
	return err
}

// authStatusReportJSON is the cmd-side JSON shape for ProviderAuthStatus.
// internal/cli's struct is intentionally tag-free; mirroring it here
// keeps presentation concerns out of the package and makes the wire
// shape explicit. Build provenance leads — same convention as
// update --json / doctor --json / status --json / restore --list --json.
type authStatusReportJSON struct {
	Build         buildProvenanceJSON               `json:"build"`
	Provider      string                            `json:"provider"`
	AuthType      string                            `json:"auth_type"`
	Status        string                            `json:"status"`
	Reason        string                            `json:"reason,omitempty"`
	Authenticated bool                              `json:"authenticated"`
	Redacted      bool                              `json:"redacted"`
	Credentials   []config.RedactedCredentialStatus `json:"credentials"`
}

func authStatusToJSON(status cli.ProviderAuthStatus) authStatusReportJSON {
	creds := status.Credentials
	if creds == nil {
		creds = []config.RedactedCredentialStatus{}
	}
	return authStatusReportJSON{
		Build:         newBuildProvenance(),
		Provider:      status.Provider,
		AuthType:      status.AuthType,
		Status:        status.Status,
		Reason:        status.Reason,
		Authenticated: status.Authenticated,
		Redacted:      status.Redacted,
		Credentials:   creds,
	}
}

func runAuthLogoutCommand(cmd *cobra.Command, providerInput string) error {
	provider := normalizeAuthProvider(providerInput)
	entries, err := loadAuthEntries(provider)
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if len(entries) == 0 {
		if asJSON {
			return writeAuthLifecycleJSON(cmd.OutOrStdout(), authLifecycleReportJSON{
				Build:    newBuildProvenance(),
				Action:   "absent",
				Provider: provider,
				Redacted: true,
			})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "auth_state_absent provider=%s redacted=true\n", provider)
		return nil
	}
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, nil); err != nil {
		return err
	}
	if asJSON {
		return writeAuthLifecycleJSON(cmd.OutOrStdout(), authLifecycleReportJSON{
			Build:    newBuildProvenance(),
			Action:   "logged_out",
			Provider: provider,
			Redacted: true,
		})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auth_logged_out provider=%s redacted=true\n", provider)
	return nil
}

func loadAuthEntries(provider string) ([]config.PooledCredential, error) {
	if provider == "" {
		return nil, errors.New("gormes auth: provider is required")
	}
	pool, evidence, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return nil, fmt.Errorf("gormes auth %s: %s", provider, evidence.Code)
	}
	return pool.Entries(), nil
}

func findCredentialIndex(entries []config.PooledCredential, target string) int {
	trimmed := strings.TrimSpace(target)
	if n, err := strconv.Atoi(trimmed); err == nil && n >= 1 && n <= len(entries) {
		return n - 1
	}
	for i, entry := range entries {
		if entry.ID == trimmed || entry.Label == trimmed {
			return i
		}
	}
	return -1
}

func normalizeAuthProvider(provider string) string {
	normalized := normalizeProviderName(provider)
	switch normalized {
	case "or", "open-router":
		return "openrouter"
	default:
		return normalized
	}
}

func normalizeAuthType(authType, provider string) string {
	switch strings.ReplaceAll(strings.ToLower(strings.TrimSpace(authType)), "_", "-") {
	case "", "oauth":
		if authProviderDefaultsToOAuth(provider) {
			return config.CredentialAuthOAuth
		}
		return config.CredentialAuthAPIKey
	case "api-key":
		return config.CredentialAuthAPIKey
	default:
		return strings.TrimSpace(authType)
	}
}

func authProviderDefaultsToOAuth(provider string) bool {
	switch provider {
	case "anthropic", "nous", config.CodexOAuthProvider, "qwen-oauth", "google-gemini-cli":
		return true
	default:
		return false
	}
}

func providerBaseURL(provider, override string) string {
	if baseURL := strings.TrimRight(strings.TrimSpace(override), "/"); baseURL != "" {
		return baseURL
	}
	switch provider {
	case "openrouter":
		return openRouterBaseURL
	case config.AnthropicProvider:
		return "https://api.anthropic.com"
	case config.CodexOAuthProvider:
		return "https://chatgpt.com/backend-api/codex"
	case config.NousOAuthProvider:
		return "https://inference-api.nousresearch.com/v1"
	case "google-gemini-cli":
		return "cloudcode-pa://google"
	case "qwen-oauth":
		return "https://portal.qwen.ai/v1"
	default:
		return ""
	}
}

func displayCredentialSource(source string) string {
	return strings.TrimPrefix(strings.TrimSpace(source), "manual:")
}

func labelFromOAuthToken(accessToken, fallback string) string {
	parts := strings.Split(strings.TrimSpace(accessToken), ".")
	if len(parts) < 2 {
		return fallback
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fallback
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fallback
	}
	for _, key := range []string{"email", "preferred_username", "name", "sub"} {
		if value := strings.TrimSpace(fmt.Sprint(claims[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return fallback
}

func nextCredentialID(provider string, entries []config.PooledCredential) string {
	prefix := provider + "-manual-"
	next := len(entries) + 1
	for {
		candidate := fmt.Sprintf("%s%d", prefix, next)
		exists := false
		for _, entry := range entries {
			if entry.ID == candidate {
				exists = true
				break
			}
		}
		if !exists {
			return candidate
		}
		next++
	}
}
