package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CodexOAuthProvider = "openai-codex"

	CodexOAuthStatusAuthorized         = "authorized"
	CodexOAuthStatusMissing            = "codex_auth_missing"
	CodexOAuthStatusReloginRequired    = "codex_auth_relogin_required"
	CodexOAuthStatusImportNotRequested = "codex_cli_import_not_requested"
	CodexOAuthStatusImportRejected     = "codex_cli_import_rejected"
	CodexOAuthStatusCorrupt            = "codex_auth_corrupt"

	CodexOAuthEvidenceSaved         = "codex_oauth_saved"
	CodexOAuthEvidenceMissing       = "codex_oauth_missing"
	CodexOAuthEvidenceImportExpired = "codex_cli_import_expired"
	CodexOAuthEvidenceImportSkipped = "codex_cli_import_not_requested"
	CodexOAuthEvidenceImportMissing = "codex_cli_import_missing"
	CodexOAuthEvidenceCorrupt       = "codex_oauth_corrupt"

	CodexOAuthSourceDeviceCode     = "device-code"
	CodexOAuthSourceCodexCLIImport = "codex-cli-import"
	defaultCodexOAuthAccountID     = "openai-codex-default"
	defaultCodexOAuthBaseURL       = "https://chatgpt.com/backend-api/codex"
)

type CodexOAuthStateStoreOptions struct {
	HermesHome string
	Now        func() time.Time
}

type CodexOAuthStateStore struct {
	hermesHome string
	now        func() time.Time
}

type CodexOAuthTokens struct {
	AccountID    string
	Label        string
	AccessToken  string
	RefreshToken string
	BaseURL      string
	Source       string
	LastRefresh  string
}

type CodexOAuthAuthStatus struct {
	Code          string `json:"code"`
	Provider      string `json:"provider"`
	AccountID     string `json:"account_id,omitempty"`
	Label         string `json:"label,omitempty"`
	Authenticated bool   `json:"authenticated"`
	BaseURL       string `json:"base_url,omitempty"`
	Source        string `json:"source,omitempty"`
	Evidence      string `json:"evidence,omitempty"`
	Redacted      bool   `json:"redacted"`
	AccessToken   string `json:"-"`
	RefreshToken  string `json:"-"`
}

type CodexCLIImportRequest struct {
	AuthPath  string
	Explicit  bool
	AccountID string
	Label     string
	BaseURL   string
}

type codexCLIAuthFile struct {
	Tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"tokens"`
}

func NewCodexOAuthStateStore(opts CodexOAuthStateStoreOptions) *CodexOAuthStateStore {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &CodexOAuthStateStore{
		hermesHome: strings.TrimSpace(opts.HermesHome),
		now:        now,
	}
}

func (s *CodexOAuthStateStore) SaveTokens(tokens CodexOAuthTokens) (CodexOAuthAuthStatus, error) {
	normalized := normalizeCodexOAuthTokens(tokens, s.now())
	if normalized.AccessToken == "" || normalized.RefreshToken == "" {
		return CodexOAuthAuthStatus{
			Code:      CodexOAuthStatusReloginRequired,
			Provider:  CodexOAuthProvider,
			AccountID: normalized.AccountID,
			Evidence:  CodexOAuthStatusReloginRequired,
			Redacted:  true,
		}, nil
	}
	entry := PooledCredential{
		ID:                 normalized.AccountID,
		Label:              normalized.Label,
		AuthType:           CredentialAuthOAuth,
		Priority:           0,
		Source:             normalized.Source,
		AccessToken:        normalized.AccessToken,
		RefreshToken:       normalized.RefreshToken,
		BaseURL:            normalized.BaseURL,
		InferenceBaseURL:   normalized.BaseURL,
		LastRefresh:        normalized.LastRefresh,
		LastStatus:         CredentialStatusOK,
		MaxConcurrentLease: 1,
	}
	home, err := credentialPoolHermesHome(s.hermesHome)
	if err != nil {
		return CodexOAuthAuthStatus{}, err
	}
	store, err := readCredentialPoolAuthStore(home)
	if err != nil {
		return CodexOAuthAuthStatus{}, err
	}
	entries := cloneCredentialEntries(store.CredentialPool[CodexOAuthProvider])
	replaced := false
	for i := range entries {
		if entries[i].ID == entry.ID {
			entry.Priority = entries[i].Priority
			entries[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		entries = append(entries, entry)
	}
	if store.CredentialPool == nil {
		store.CredentialPool = make(map[string][]PooledCredential)
	}
	store.CredentialPool[CodexOAuthProvider] = normalizeCredentialEntries(entries)
	if store.SuppressedSources != nil {
		delete(store.SuppressedSources, CodexOAuthProvider)
		if len(store.SuppressedSources) == 0 {
			store.SuppressedSources = nil
		}
	}
	if err := writeCredentialPoolAuthStore(home, store); err != nil {
		return CodexOAuthAuthStatus{}, err
	}
	return CodexOAuthAuthStatus{
		Code:          CodexOAuthStatusAuthorized,
		Provider:      CodexOAuthProvider,
		AccountID:     normalized.AccountID,
		Label:         normalized.Label,
		Authenticated: true,
		BaseURL:       normalized.BaseURL,
		Source:        normalized.Source,
		Evidence:      CodexOAuthEvidenceSaved,
		Redacted:      true,
	}, nil
}

func (s *CodexOAuthStateStore) codexCredentialEntries() ([]PooledCredential, error) {
	pool, _, err := LoadCredentialPool(CredentialPoolOptions{
		HermesHome: s.hermesHome,
		Provider:   CodexOAuthProvider,
	})
	if err != nil {
		return nil, err
	}
	return pool.Entries(), nil
}

func (s *CodexOAuthStateStore) CheckAuth() (CodexOAuthAuthStatus, error) {
	pool, evidence, err := LoadCredentialPool(CredentialPoolOptions{
		HermesHome: s.hermesHome,
		Provider:   CodexOAuthProvider,
	})
	if err != nil {
		return CodexOAuthAuthStatus{
			Code:     CodexOAuthStatusCorrupt,
			Provider: CodexOAuthProvider,
			Evidence: CodexOAuthEvidenceCorrupt,
			Redacted: true,
		}, err
	}
	entries := pool.Entries()
	if len(entries) == 0 || evidence.Code == CredentialPoolEvidenceEmpty {
		return CodexOAuthAuthStatus{
			Code:     CodexOAuthStatusMissing,
			Provider: CodexOAuthProvider,
			Evidence: CodexOAuthEvidenceMissing,
			Redacted: true,
		}, nil
	}
	entry := entries[0]
	if strings.TrimSpace(entry.AccessToken) == "" || strings.TrimSpace(entry.RefreshToken) == "" {
		return CodexOAuthAuthStatus{
			Code:      CodexOAuthStatusReloginRequired,
			Provider:  CodexOAuthProvider,
			AccountID: entry.ID,
			Label:     entry.Label,
			BaseURL:   firstNonEmpty(entry.InferenceBaseURL, entry.BaseURL),
			Source:    entry.Source,
			Evidence:  CodexOAuthStatusReloginRequired,
			Redacted:  true,
		}, nil
	}
	return CodexOAuthAuthStatus{
		Code:          CodexOAuthStatusAuthorized,
		Provider:      CodexOAuthProvider,
		AccountID:     entry.ID,
		Label:         entry.Label,
		Authenticated: true,
		BaseURL:       firstNonEmpty(entry.InferenceBaseURL, entry.BaseURL),
		Source:        entry.Source,
		Redacted:      true,
	}, nil
}

func (s *CodexOAuthStateStore) ImportCodexCLITokens(req CodexCLIImportRequest) (CodexOAuthAuthStatus, error) {
	if !req.Explicit || strings.TrimSpace(req.AuthPath) == "" {
		return CodexOAuthAuthStatus{
			Code:     CodexOAuthStatusImportNotRequested,
			Provider: CodexOAuthProvider,
			Evidence: CodexOAuthEvidenceImportSkipped,
			Redacted: true,
		}, nil
	}
	var payload codexCLIAuthFile
	if err := readJSON(req.AuthPath, &payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CodexOAuthAuthStatus{
				Code:     CodexOAuthStatusImportRejected,
				Provider: CodexOAuthProvider,
				Evidence: CodexOAuthEvidenceImportMissing,
				Redacted: true,
			}, nil
		}
		return CodexOAuthAuthStatus{
			Code:     CodexOAuthStatusCorrupt,
			Provider: CodexOAuthProvider,
			Evidence: CodexOAuthEvidenceCorrupt,
			Redacted: true,
		}, nil
	}
	accessToken := strings.TrimSpace(payload.Tokens.AccessToken)
	refreshToken := strings.TrimSpace(payload.Tokens.RefreshToken)
	if accessToken == "" || refreshToken == "" || codexAccessTokenExpired(accessToken, s.now()) {
		return CodexOAuthAuthStatus{
			Code:     CodexOAuthStatusImportRejected,
			Provider: CodexOAuthProvider,
			Evidence: CodexOAuthEvidenceImportExpired,
			Redacted: true,
		}, nil
	}
	return s.SaveTokens(CodexOAuthTokens{
		AccountID:    req.AccountID,
		Label:        firstNonEmpty(req.Label, "Imported Codex CLI"),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		BaseURL:      req.BaseURL,
		Source:       CodexOAuthSourceCodexCLIImport,
	})
}

func normalizeCodexOAuthTokens(tokens CodexOAuthTokens, now time.Time) CodexOAuthTokens {
	tokens.AccountID = firstNonEmpty(tokens.AccountID, defaultCodexOAuthAccountID)
	tokens.Label = firstNonEmpty(tokens.Label, "OpenAI Codex")
	tokens.AccessToken = strings.TrimSpace(tokens.AccessToken)
	tokens.RefreshToken = strings.TrimSpace(tokens.RefreshToken)
	tokens.BaseURL = strings.TrimRight(firstNonEmpty(tokens.BaseURL, defaultCodexOAuthBaseURL), "/")
	tokens.Source = firstNonEmpty(tokens.Source, CodexOAuthSourceDeviceCode)
	if strings.TrimSpace(tokens.LastRefresh) == "" {
		tokens.LastRefresh = now.UTC().Format(time.RFC3339)
	}
	return tokens
}

func codexAccessTokenExpired(accessToken string, now time.Time) bool {
	claims, ok := decodeJWTClaims(accessToken)
	if !ok {
		return false
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return false
	}
	return int64(exp) <= now.Unix()
}

func decodeJWTClaims(token string) (map[string]any, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, false
	}
	return claims, true
}

func (s *CodexOAuthStateStore) AuthPath() (string, error) {
	home, err := credentialPoolHermesHome(s.hermesHome)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "auth.json"), nil
}

func (s *CodexOAuthStateStore) String() string {
	path, err := s.AuthPath()
	if err != nil {
		return CodexOAuthProvider
	}
	return fmt.Sprintf("%s:%s", CodexOAuthProvider, filepath.ToSlash(path))
}
