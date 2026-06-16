package credentials

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials/googleoauth"

const (
	GoogleOAuthStatusAuthorized     = googleoauth.GoogleOAuthStatusAuthorized
	GoogleOAuthStatusMissing        = googleoauth.GoogleOAuthStatusMissing
	GoogleOAuthStatusPendingMissing = googleoauth.GoogleOAuthStatusPendingMissing
	GoogleOAuthStatusStateMismatch  = googleoauth.GoogleOAuthStatusStateMismatch
	GoogleOAuthStatusPartialScope   = googleoauth.GoogleOAuthStatusPartialScope
	GoogleOAuthStatusCorrupt        = googleoauth.GoogleOAuthStatusCorrupt
)

type GoogleOAuthStateStore = googleoauth.GoogleOAuthStateStore
type GoogleOAuthPendingAuth = googleoauth.GoogleOAuthPendingAuth
type GoogleOAuthCallback = googleoauth.GoogleOAuthCallback
type GoogleOAuthAuthStatus = googleoauth.GoogleOAuthAuthStatus

func NewGoogleOAuthStateStore(dir string) *GoogleOAuthStateStore {
	return googleoauth.NewGoogleOAuthStateStore(dir)
}

func ExtractGoogleOAuthCodeAndState(input string) (GoogleOAuthCallback, error) {
	return googleoauth.ExtractGoogleOAuthCodeAndState(input)
}
