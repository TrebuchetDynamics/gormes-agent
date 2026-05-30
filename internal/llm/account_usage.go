package llm

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/accountusage"
)

type AccountUsageReason = accountusage.AccountUsageReason

const (
	AccountUsageReasonUnsupportedProvider AccountUsageReason = accountusage.AccountUsageReasonUnsupportedProvider
	AccountUsageReasonCredentialMissing   AccountUsageReason = accountusage.AccountUsageReasonCredentialMissing
	AccountUsageReasonOAuthRequired       AccountUsageReason = accountusage.AccountUsageReasonOAuthRequired
	AccountUsageReasonHTTPStatus          AccountUsageReason = accountusage.AccountUsageReasonHTTPStatus
	AccountUsageReasonMalformedPayload    AccountUsageReason = accountusage.AccountUsageReasonMalformedPayload
	AccountUsageReasonRequestFailed       AccountUsageReason = accountusage.AccountUsageReasonRequestFailed
)

type AccountUsageFetchRequest = accountusage.AccountUsageFetchRequest
type AccountUsageHTTPRequest = accountusage.AccountUsageHTTPRequest
type AccountUsageHTTPResponse = accountusage.AccountUsageHTTPResponse
type AccountUsageHTTPClient = accountusage.AccountUsageHTTPClient
type AccountUsageFetcher = accountusage.AccountUsageFetcher
type AccountUsageSnapshot = accountusage.AccountUsageSnapshot
type AccountUsageWindow = accountusage.AccountUsageWindow
type AccountUsageUnavailable = accountusage.AccountUsageUnavailable
type AccountUsageRenderOptions = accountusage.AccountUsageRenderOptions

func NewAccountUsageFetcher(client AccountUsageHTTPClient, now func() time.Time) AccountUsageFetcher {
	return accountusage.NewAccountUsageFetcher(client, now)
}

func RenderAccountUsageLines(snapshot AccountUsageSnapshot, opts AccountUsageRenderOptions) []string {
	return accountusage.RenderAccountUsageLines(snapshot, opts)
}

func RenderAccountUsageJSON(snapshot AccountUsageSnapshot) ([]byte, error) {
	return accountusage.RenderAccountUsageJSON(snapshot)
}
