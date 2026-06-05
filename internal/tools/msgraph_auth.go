package tools

import (
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/msgraph"
)

const (
	DefaultMicrosoftGraphScope        = msgraph.DefaultMicrosoftGraphScope
	DefaultMicrosoftGraphAuthorityURL = msgraph.DefaultMicrosoftGraphAuthorityURL

	MicrosoftGraphNotConfigured      = msgraph.MicrosoftGraphNotConfigured
	MicrosoftGraphTokenUnavailable   = msgraph.MicrosoftGraphTokenUnavailable
	MicrosoftGraphRequestUnavailable = msgraph.MicrosoftGraphRequestUnavailable
)

type MicrosoftGraphHTTPClient = msgraph.MicrosoftGraphHTTPClient

type MicrosoftGraphError = msgraph.MicrosoftGraphError

func AsMicrosoftGraphError(err error, target **MicrosoftGraphError) bool {
	return errors.As(err, target)
}

type MicrosoftGraphCredentials = msgraph.MicrosoftGraphCredentials

func MicrosoftGraphCredentialsFromEnv(env map[string]string, required bool) (*MicrosoftGraphCredentials, error) {
	return msgraph.MicrosoftGraphCredentialsFromEnv(env, required)
}

type MicrosoftGraphAccessToken = msgraph.MicrosoftGraphAccessToken

type MicrosoftGraphTokenProviderOptions = msgraph.MicrosoftGraphTokenProviderOptions

type MicrosoftGraphTokenProvider = msgraph.MicrosoftGraphTokenProvider

func NewMicrosoftGraphTokenProvider(creds MicrosoftGraphCredentials, opts MicrosoftGraphTokenProviderOptions) *MicrosoftGraphTokenProvider {
	return msgraph.NewMicrosoftGraphTokenProvider(creds, opts)
}
