package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"

const (
	GoogleOAuthStatusAuthorized     = credentials.GoogleOAuthStatusAuthorized
	GoogleOAuthStatusMissing        = credentials.GoogleOAuthStatusMissing
	GoogleOAuthStatusPendingMissing = credentials.GoogleOAuthStatusPendingMissing
	GoogleOAuthStatusStateMismatch  = credentials.GoogleOAuthStatusStateMismatch
	GoogleOAuthStatusPartialScope   = credentials.GoogleOAuthStatusPartialScope
	GoogleOAuthStatusCorrupt        = credentials.GoogleOAuthStatusCorrupt
)

type GoogleOAuthStateStore = credentials.GoogleOAuthStateStore
type GoogleOAuthPendingAuth = credentials.GoogleOAuthPendingAuth
type GoogleOAuthCallback = credentials.GoogleOAuthCallback
type GoogleOAuthAuthStatus = credentials.GoogleOAuthAuthStatus

func NewGoogleOAuthStateStore(dir string) *GoogleOAuthStateStore {
	return credentials.NewGoogleOAuthStateStore(dir)
}

func ExtractGoogleOAuthCodeAndState(input string) (GoogleOAuthCallback, error) {
	return credentials.ExtractGoogleOAuthCodeAndState(input)
}
