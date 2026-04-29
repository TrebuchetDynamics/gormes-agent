package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const openRouterBaseURL = "https://openrouter.ai/api/v1"

func newLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "login",
		Short:        "Removed compatibility shim; use auth/model/setup",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "The 'gormes login' command has been removed.")
			fmt.Fprintln(cmd.OutOrStdout(), "Use 'gormes auth' to manage credentials,")
			fmt.Fprintln(cmd.OutOrStdout(), "'gormes model' to select a provider, or 'gormes setup' for full setup.")
			return nil
		},
	}
	cmd.Flags().String("provider", "", "ignored compatibility flag")
	cmd.Flags().String("portal-url", "", "ignored compatibility flag")
	cmd.Flags().String("inference-url", "", "ignored compatibility flag")
	cmd.Flags().String("client-id", "", "ignored compatibility flag")
	cmd.Flags().String("scope", "", "ignored compatibility flag")
	cmd.Flags().Bool("no-browser", false, "ignored compatibility flag")
	cmd.Flags().String("timeout", "", "ignored compatibility flag")
	cmd.Flags().Bool("insecure", false, "ignored compatibility flag")
	cmd.Flags().String("ca-bundle", "", "ignored compatibility flag")
	return cmd
}

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

	cmd := &cobra.Command{
		Use:   "add <provider>",
		Short: "Add a provider credential to the Hermes-compatible credential pool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthAddCommand(cmd, authAddOptions{
				Provider:     args[0],
				AuthType:     authType,
				Label:        label,
				APIKey:       apiKey,
				InferenceURL: inferenceURL,
				PortalURL:    portalURL,
				ClientID:     clientID,
				Scope:        scope,
				NoBrowser:    noBrowser,
				Timeout:      timeout,
				Insecure:     insecure,
				CABundle:     caBundle,
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
	return cmd
}

func newAuthListCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "list [provider]",
		Short:        "List provider credentials with secrets redacted",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := ""
			if len(args) > 0 {
				provider = args[0]
			}
			return runAuthListCommand(cmd, provider)
		},
	}
}

func newAuthRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "remove <provider> <target>",
		Short:        "Remove a provider credential by index, id, or label",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthRemoveCommand(cmd, args[0], args[1])
		},
	}
}

func newAuthResetCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "reset <provider>",
		Short:        "Reset provider credential cooldown/exhaustion state",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthResetCommand(cmd, args[0])
		},
	}
}

func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "status <provider>",
		Short:        "Show redacted provider auth status",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthStatusCommand(cmd, args[0])
		},
	}
}

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "logout <provider>",
		Short:        "Clear provider credentials",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthLogoutCommand(cmd, args[0])
		},
	}
}

type authAddOptions struct {
	Provider     string
	AuthType     string
	Label        string
	APIKey       string
	InferenceURL string
	PortalURL    string
	ClientID     string
	Scope        string
	NoBrowser    bool
	Timeout      string
	Insecure     bool
	CABundle     string
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
		fmt.Fprintf(cmd.OutOrStdout(), "%s (%d credentials) credential_pool provider=%s credentials=%d redacted=%t\n", provider, status.Count, provider, status.Count, status.Redacted)
		printed = true
	}
	if !printed {
		fmt.Fprintln(cmd.OutOrStdout(), "credential_pool_empty provider=all redacted=true")
	}
	fmt.Fprintln(cmd.OutOrStdout(), "bedrock_identity status=not_checked redacted=true")
	return nil
}

func runAuthAddCommand(cmd *cobra.Command, opts authAddOptions) error {
	provider := normalizeAuthProvider(opts.Provider)
	if provider == "" {
		return errors.New("gormes auth add: provider is required")
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

func runAuthAddCodexOAuthCommand(cmd *cobra.Command, opts authAddOptions) error {
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
	return nil
}

func runAuthListCommand(cmd *cobra.Command, providerInput string) error {
	provider := normalizeAuthProvider(providerInput)
	if provider == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "credential_pool_empty provider=all redacted=true")
		return nil
	}
	pool, evidence, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: provider})
	if err != nil {
		return fmt.Errorf("gormes auth list %s: %s", provider, evidence.Code)
	}
	status := pool.RedactedStatus()
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
	fmt.Fprintf(cmd.OutOrStdout(), "auth_status_reset provider=%s count=%d redacted=true\n", provider, len(entries))
	return nil
}

func runAuthStatusCommand(cmd *cobra.Command, providerInput string) error {
	_, err := cli.RenderAuthStatus(context.Background(), cmd.OutOrStdout(), providerInput, cli.AuthStatusOptions{})
	return err
}

func runAuthLogoutCommand(cmd *cobra.Command, providerInput string) error {
	provider := normalizeAuthProvider(providerInput)
	if _, err := loadAuthEntries(provider); err != nil {
		return err
	}
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: provider}, nil); err != nil {
		return err
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
