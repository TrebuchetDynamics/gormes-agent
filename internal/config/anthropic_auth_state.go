package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	AnthropicProvider = "anthropic"

	AnthropicAuthStatusAuthorized      = "authorized"
	AnthropicAuthStatusMissing         = "anthropic_auth_missing"
	AnthropicAuthStatusReloginRequired = "anthropic_auth_relogin_required"
	AnthropicAuthStatusCorrupt         = "anthropic_auth_corrupt"

	AnthropicOAuthEvidenceKeychainSelected    = "anthropic_oauth_keychain_selected"
	AnthropicOAuthEvidenceJSONSelected        = "anthropic_oauth_json_selected"
	AnthropicOAuthEvidenceMissing             = "anthropic_oauth_missing"
	AnthropicOAuthEvidenceKeychainUnavailable = "anthropic_oauth_keychain_unavailable"
	AnthropicOAuthEvidenceCorruptBackup       = "anthropic_oauth_corrupt_backup"
	AnthropicOAuthEvidenceStaleOAuth          = "anthropic_oauth_stale"

	AnthropicOAuthSourceMacOSKeychain = "macos_keychain"
	AnthropicOAuthSourceJSONFile      = "claude_code_credentials_file"
)

type AnthropicKeychainReader func(context.Context) (AnthropicClaudeCredentials, error)

type AnthropicAuthStateStoreOptions struct {
	CredentialsPath string
	Keychain        AnthropicKeychainReader
	Now             func() time.Time
}

type AnthropicAuthStateStore struct {
	credentialsPath string
	keychain        AnthropicKeychainReader
	now             func() time.Time
}

type AnthropicClaudeCredentials struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAtMS  int64  `json:"expiresAt,omitempty"`
	Source       string `json:"source,omitempty"`
}

type AnthropicAuthStatus struct {
	Code            string                     `json:"code"`
	Provider        string                     `json:"provider"`
	Authenticated   bool                       `json:"authenticated"`
	ReloginRequired bool                       `json:"relogin_required"`
	Source          string                     `json:"source,omitempty"`
	Evidence        string                     `json:"evidence,omitempty"`
	BackupPath      string                     `json:"backup_path,omitempty"`
	ExpiresAtMS     int64                      `json:"expires_at_ms,omitempty"`
	Redacted        bool                       `json:"redacted"`
	AccessToken     string                     `json:"-"`
	RefreshToken    string                     `json:"-"`
	Credentials     AnthropicClaudeCredentials `json:"-"`
}

type anthropicClaudeCredentialsFile struct {
	ClaudeAiOauth anthropicClaudeCredentialsPayload `json:"claudeAiOauth"`
}

type anthropicClaudeCredentialsPayload struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAtMS  int64  `json:"expiresAt"`
}

func NewAnthropicAuthStateStore(opts AnthropicAuthStateStoreOptions) *AnthropicAuthStateStore {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	keychain := opts.Keychain
	if keychain == nil {
		keychain = readAnthropicClaudeCredentialsFromKeychain
	}
	return &AnthropicAuthStateStore{
		credentialsPath: strings.TrimSpace(opts.CredentialsPath),
		keychain:        keychain,
		now:             now,
	}
}

func (s *AnthropicAuthStateStore) CheckAuth(ctx context.Context) (AnthropicAuthStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	keychainUnavailable := false
	if s.keychain != nil {
		creds, err := s.keychain(ctx)
		if err != nil {
			keychainUnavailable = true
		} else if creds.AccessToken = strings.TrimSpace(creds.AccessToken); creds.AccessToken != "" {
			creds = normalizeAnthropicClaudeCredentials(creds, AnthropicOAuthSourceMacOSKeychain)
			return s.statusForCredentials(creds, AnthropicOAuthEvidenceKeychainSelected), nil
		}
	}
	path, err := s.credentialsPathOrDefault()
	if err != nil {
		return AnthropicAuthStatus{}, err
	}
	creds, status, err := s.readCredentialsFile(path)
	if err != nil {
		return status, err
	}
	if status.Code != "" {
		if keychainUnavailable && status.Code == AnthropicAuthStatusMissing {
			status.Evidence = AnthropicOAuthEvidenceKeychainUnavailable
		}
		return status, nil
	}
	return s.statusForCredentials(creds, AnthropicOAuthEvidenceJSONSelected), nil
}

func (s *AnthropicAuthStateStore) readCredentialsFile(path string) (AnthropicClaudeCredentials, AnthropicAuthStatus, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return AnthropicClaudeCredentials{}, AnthropicAuthStatus{
			Code:     AnthropicAuthStatusMissing,
			Provider: AnthropicProvider,
			Evidence: AnthropicOAuthEvidenceMissing,
			Redacted: true,
		}, nil
	}
	if err != nil {
		return AnthropicClaudeCredentials{}, AnthropicAuthStatus{}, err
	}
	creds, err := parseAnthropicClaudeCredentials(data, AnthropicOAuthSourceJSONFile)
	if err != nil {
		backupPath, backupErr := s.preserveCorruptCredentials(path, data)
		status := AnthropicAuthStatus{
			Code:       AnthropicAuthStatusCorrupt,
			Provider:   AnthropicProvider,
			Evidence:   AnthropicOAuthEvidenceCorruptBackup,
			BackupPath: backupPath,
			Redacted:   true,
		}
		if backupErr != nil {
			return AnthropicClaudeCredentials{}, status, backupErr
		}
		return AnthropicClaudeCredentials{}, status, nil
	}
	if strings.TrimSpace(creds.AccessToken) == "" {
		return AnthropicClaudeCredentials{}, AnthropicAuthStatus{
			Code:     AnthropicAuthStatusMissing,
			Provider: AnthropicProvider,
			Evidence: AnthropicOAuthEvidenceMissing,
			Redacted: true,
		}, nil
	}
	return creds, AnthropicAuthStatus{}, nil
}

func (s *AnthropicAuthStateStore) statusForCredentials(creds AnthropicClaudeCredentials, evidence string) AnthropicAuthStatus {
	if anthropicCredentialsRequireRelogin(creds, s.now()) {
		return AnthropicAuthStatus{
			Code:            AnthropicAuthStatusReloginRequired,
			Provider:        AnthropicProvider,
			ReloginRequired: true,
			Source:          creds.Source,
			Evidence:        AnthropicOAuthEvidenceStaleOAuth,
			ExpiresAtMS:     creds.ExpiresAtMS,
			Redacted:        true,
		}
	}
	return AnthropicAuthStatus{
		Code:          AnthropicAuthStatusAuthorized,
		Provider:      AnthropicProvider,
		Authenticated: true,
		Source:        creds.Source,
		Evidence:      evidence,
		ExpiresAtMS:   creds.ExpiresAtMS,
		Redacted:      true,
		Credentials:   creds,
	}
}

func (s *AnthropicAuthStateStore) preserveCorruptCredentials(path string, data []byte) (string, error) {
	backupPath := path + ".corrupt." + s.now().UTC().Format("20060102T150405Z")
	if _, err := os.Stat(backupPath); err == nil {
		for i := 1; ; i++ {
			candidate := backupPath + "." + strconv.Itoa(i)
			if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
				backupPath = candidate
				break
			}
		}
	}
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}

func (s *AnthropicAuthStateStore) credentialsPathOrDefault() (string, error) {
	if s.credentialsPath != "" {
		return s.credentialsPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", ".credentials.json"), nil
}

func readAnthropicClaudeCredentialsFromKeychain(ctx context.Context) (AnthropicClaudeCredentials, error) {
	if runtime.GOOS != "darwin" {
		return AnthropicClaudeCredentials{}, nil
	}
	if _, err := exec.LookPath("security"); err != nil {
		return AnthropicClaudeCredentials{}, nil
	}
	cmd := exec.CommandContext(ctx, "security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	out, err := cmd.Output()
	if err != nil {
		return AnthropicClaudeCredentials{}, nil
	}
	return parseAnthropicClaudeCredentials(out, AnthropicOAuthSourceMacOSKeychain)
}

func parseAnthropicClaudeCredentials(data []byte, source string) (AnthropicClaudeCredentials, error) {
	var payload anthropicClaudeCredentialsFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return AnthropicClaudeCredentials{}, err
	}
	return normalizeAnthropicClaudeCredentials(AnthropicClaudeCredentials{
		AccessToken:  payload.ClaudeAiOauth.AccessToken,
		RefreshToken: payload.ClaudeAiOauth.RefreshToken,
		ExpiresAtMS:  payload.ClaudeAiOauth.ExpiresAtMS,
	}, source), nil
}

func normalizeAnthropicClaudeCredentials(creds AnthropicClaudeCredentials, source string) AnthropicClaudeCredentials {
	creds.AccessToken = strings.TrimSpace(creds.AccessToken)
	creds.RefreshToken = strings.TrimSpace(creds.RefreshToken)
	creds.Source = firstNonEmpty(strings.TrimSpace(creds.Source), source)
	return creds
}

func anthropicCredentialsRequireRelogin(creds AnthropicClaudeCredentials, now time.Time) bool {
	if strings.TrimSpace(creds.RefreshToken) != "" {
		return false
	}
	if !anthropicTokenLooksOAuth(creds.AccessToken) {
		return false
	}
	if creds.ExpiresAtMS > 0 {
		return creds.ExpiresAtMS <= now.Add(time.Minute).UnixMilli()
	}
	claims, ok := decodeJWTClaims(creds.AccessToken)
	if !ok {
		return false
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return false
	}
	return int64(exp) <= now.Add(time.Minute).Unix()
}

func anthropicTokenLooksOAuth(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	for _, prefix := range []string{"sk-ant-oat", "sk-ant-setup", "cc-"} {
		if strings.HasPrefix(token, prefix) {
			return true
		}
	}
	return false
}
