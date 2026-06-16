package googleoauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials/jsonfile"
)

const (
	GoogleOAuthStatusAuthorized     = "authorized"
	GoogleOAuthStatusMissing        = "oauth_missing"
	GoogleOAuthStatusPendingMissing = "oauth_pending_missing"
	GoogleOAuthStatusStateMismatch  = "oauth_state_mismatch"
	GoogleOAuthStatusPartialScope   = "token_partial_scope"
	GoogleOAuthStatusCorrupt        = "token_corrupt"
)

type GoogleOAuthStateStore struct {
	dir string
}

type GoogleOAuthPendingAuth struct {
	State           string   `json:"state"`
	CodeVerifier    string   `json:"code_verifier"`
	RedirectURI     string   `json:"redirect_uri"`
	RequestedScopes []string `json:"requested_scopes"`
}

type GoogleOAuthCallback struct {
	Code   string   `json:"code"`
	State  string   `json:"state,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

type GoogleOAuthAuthStatus struct {
	Code          string   `json:"code"`
	Authenticated bool     `json:"authenticated"`
	MissingScopes []string `json:"missing_scopes,omitempty"`
	GrantedScopes []string `json:"granted_scopes,omitempty"`
	Evidence      string   `json:"evidence,omitempty"`
	ClientSecret  string   `json:"-"`
	AccessToken   string   `json:"-"`
	RefreshToken  string   `json:"-"`
}

type googleOAuthTokenFile struct {
	Type         string   `json:"type"`
	ClientID     string   `json:"client_id,omitempty"`
	ClientSecret string   `json:"client_secret,omitempty"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	AccessToken  string   `json:"access_token,omitempty"`
	TokenURI     string   `json:"token_uri,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

func NewGoogleOAuthStateStore(dir string) *GoogleOAuthStateStore {
	return &GoogleOAuthStateStore{dir: strings.TrimSpace(dir)}
}

func (s *GoogleOAuthStateStore) PendingPath() string {
	return filepath.Join(s.dir, "google-oauth-pending.json")
}

func (s *GoogleOAuthStateStore) TokenPath() string {
	return filepath.Join(s.dir, "google-token.json")
}

func (s *GoogleOAuthStateStore) SavePendingAuth(pending GoogleOAuthPendingAuth) error {
	pending.RequestedScopes = cleanScopes(pending.RequestedScopes)
	return jsonfile.Write0600(s.PendingPath(), pending)
}

func (s *GoogleOAuthStateStore) LoadPendingAuth() (GoogleOAuthPendingAuth, error) {
	var pending GoogleOAuthPendingAuth
	if err := jsonfile.Read(s.PendingPath(), &pending); err != nil {
		return GoogleOAuthPendingAuth{}, err
	}
	pending.RequestedScopes = cleanScopes(pending.RequestedScopes)
	return pending, nil
}

func ExtractGoogleOAuthCodeAndState(input string) (GoogleOAuthCallback, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return GoogleOAuthCallback{}, fmt.Errorf("google oauth callback: empty code")
	}
	if u, err := url.Parse(trimmed); err == nil && u.Scheme != "" && u.Host != "" {
		q := u.Query()
		code := strings.TrimSpace(q.Get("code"))
		if code == "" {
			return GoogleOAuthCallback{}, fmt.Errorf("google oauth callback: missing code")
		}
		return GoogleOAuthCallback{
			Code:   code,
			State:  strings.TrimSpace(q.Get("state")),
			Scopes: cleanScopes(strings.Fields(q.Get("scope"))),
		}, nil
	}
	return GoogleOAuthCallback{Code: trimmed}, nil
}

func (s *GoogleOAuthStateStore) ExchangeAuthCode(callback GoogleOAuthCallback, credentialPayload []byte) (GoogleOAuthAuthStatus, error) {
	pending, err := s.LoadPendingAuth()
	if errors.Is(err, os.ErrNotExist) {
		return GoogleOAuthAuthStatus{Code: GoogleOAuthStatusPendingMissing, Evidence: GoogleOAuthStatusPendingMissing}, nil
	}
	if err != nil {
		return GoogleOAuthAuthStatus{}, err
	}
	if strings.TrimSpace(pending.State) != "" && strings.TrimSpace(callback.State) != strings.TrimSpace(pending.State) {
		return GoogleOAuthAuthStatus{Code: GoogleOAuthStatusStateMismatch, Evidence: GoogleOAuthStatusStateMismatch}, nil
	}
	var token googleOAuthTokenFile
	if err := json.Unmarshal(credentialPayload, &token); err != nil {
		return GoogleOAuthAuthStatus{Code: GoogleOAuthStatusCorrupt, Evidence: GoogleOAuthStatusCorrupt}, nil
	}
	if token.Type == "" {
		token.Type = "authorized_user"
	}
	token.Scopes = grantedScopes(pending.RequestedScopes, callback.Scopes)
	if len(token.Scopes) == 0 {
		token.Scopes = cleanScopes(callback.Scopes)
	}
	if len(token.Scopes) == 0 {
		token.Scopes = cleanScopes(pending.RequestedScopes)
	}
	if err := jsonfile.Write0600(s.TokenPath(), token); err != nil {
		return GoogleOAuthAuthStatus{}, err
	}
	if err := os.Remove(s.PendingPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return GoogleOAuthAuthStatus{}, err
	}
	return GoogleOAuthAuthStatus{Code: GoogleOAuthStatusAuthorized, Authenticated: true, GrantedScopes: token.Scopes}, nil
}

func (s *GoogleOAuthStateStore) CheckAuth(requiredScopes []string) (GoogleOAuthAuthStatus, error) {
	var token googleOAuthTokenFile
	if err := jsonfile.Read(s.TokenPath(), &token); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return GoogleOAuthAuthStatus{Code: GoogleOAuthStatusMissing, Evidence: GoogleOAuthStatusMissing}, nil
		}
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
			return GoogleOAuthAuthStatus{Code: GoogleOAuthStatusCorrupt, Evidence: GoogleOAuthStatusCorrupt}, nil
		}
		return GoogleOAuthAuthStatus{}, err
	}
	granted := cleanScopes(token.Scopes)
	missing := missingScopes(cleanScopes(requiredScopes), granted)
	if len(missing) > 0 {
		return GoogleOAuthAuthStatus{Code: GoogleOAuthStatusPartialScope, Authenticated: true, GrantedScopes: granted, MissingScopes: missing, Evidence: GoogleOAuthStatusPartialScope}, nil
	}
	return GoogleOAuthAuthStatus{Code: GoogleOAuthStatusAuthorized, Authenticated: true, GrantedScopes: granted}, nil
}

func cleanScopes(scopes []string) []string {
	seen := make(map[string]struct{}, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func grantedScopes(requested, callback []string) []string {
	requested = cleanScopes(requested)
	callback = cleanScopes(callback)
	if len(requested) == 0 || len(callback) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(callback))
	for _, scope := range callback {
		allowed[scope] = struct{}{}
	}
	out := make([]string, 0, len(requested))
	for _, scope := range requested {
		if _, ok := allowed[scope]; ok {
			out = append(out, scope)
		}
	}
	return out
}

func missingScopes(required, granted []string) []string {
	grantedSet := make(map[string]struct{}, len(granted))
	for _, scope := range granted {
		grantedSet[scope] = struct{}{}
	}
	missing := make([]string, 0)
	for _, scope := range required {
		if _, ok := grantedSet[scope]; !ok {
			missing = append(missing, scope)
		}
	}
	sort.Strings(missing)
	return missing
}
