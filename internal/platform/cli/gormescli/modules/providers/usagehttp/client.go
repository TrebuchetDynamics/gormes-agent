package usagehttp

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// DefaultClient bounds provider account-usage requests so an unresponsive
// provider cannot hang the operator's terminal. http.DefaultClient has no
// timeout.
var DefaultClient = &http.Client{Timeout: 30 * time.Second}

// Client adapts net/http to llm.AccountUsageHTTPClient.
type Client struct{ Client *http.Client }

func (c Client) DoAccountUsageRequest(ctx context.Context, req llm.AccountUsageHTTPRequest) (llm.AccountUsageHTTPResponse, error) {
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return llm.AccountUsageHTTPResponse{}, err
	}
	for key, value := range req.Headers {
		if textvalue.IsNonBlank(value) {
			httpReq.Header.Set(key, value)
		}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return llm.AccountUsageHTTPResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return llm.AccountUsageHTTPResponse{}, err
	}
	return llm.AccountUsageHTTPResponse{StatusCode: resp.StatusCode, Body: body}, nil
}
