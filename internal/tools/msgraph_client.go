package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/msgraph"

const DefaultMicrosoftGraphBaseURL = msgraph.DefaultMicrosoftGraphBaseURL

type MicrosoftGraphClientOptions = msgraph.MicrosoftGraphClientOptions

type MicrosoftGraphClient = msgraph.MicrosoftGraphClient

type MicrosoftGraphDownloadResult = msgraph.MicrosoftGraphDownloadResult

func NewMicrosoftGraphClient(provider *MicrosoftGraphTokenProvider, opts MicrosoftGraphClientOptions) *MicrosoftGraphClient {
	return msgraph.NewMicrosoftGraphClient(provider, opts)
}
