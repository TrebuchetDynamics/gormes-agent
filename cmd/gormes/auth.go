package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

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
			})
		},
	}
	cmd.Flags().StringVar(&authType, "type", "", "credential type: api-key, api_key, or oauth")
	cmd.Flags().StringVar(&label, "label", "", "credential label")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key to store; omitted values are not echoed")
	cmd.Flags().StringVar(&inferenceURL, "inference-url", "", "provider inference base URL override")
	cmd.Flags().String("portal-url", "", "OAuth portal URL; reserved for provider OAuth parity")
	cmd.Flags().String("client-id", "", "OAuth client ID; reserved for provider OAuth parity")
	cmd.Flags().String("scope", "", "OAuth scope; reserved for provider OAuth parity")
	cmd.Flags().Bool("no-browser", false, "reserved for provider OAuth parity")
	cmd.Flags().String("timeout", "", "reserved for provider OAuth parity")
	cmd.Flags().Bool("insecure", false, "reserved for provider OAuth parity")
	cmd.Flags().String("ca-bundle", "", "reserved for provider OAuth parity")
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
		if provider == config.CodexOAuthProvider {
			return runAuthAddCodexOAuthCommand(cmd, opts)
		}
		return fmt.Errorf("gormes auth add %s --type oauth: provider OAuth adapters are planned; use --type api-key for API-key providers", provider)
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
	provider := normalizeAuthProvider(providerInput)
	entries, err := loadAuthEntries(provider)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s: logged out redacted=true\n", provider)
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: logged in credentials=%d redacted=true\n", provider, len(entries))
	return nil
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
	case config.CodexOAuthProvider:
		return "https://chatgpt.com/backend-api/codex"
	default:
		return ""
	}
}

func displayCredentialSource(source string) string {
	return strings.TrimPrefix(strings.TrimSpace(source), "manual:")
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
