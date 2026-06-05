package navivox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// navivoxHTTPContract centralizes authenticated Navivox HTTP test mechanics
// only. Tests still spell endpoint paths and scenario assertions directly so
// ADR-level distinctions, especially /v1/navivox/capabilities as the feature
// gate rather than /status, remain visible in each contract test.
type navivoxHTTPContract struct {
	t       *testing.T
	baseURL string
	token   string
}

func newNavivoxHTTPContract(t *testing.T, baseURL string) navivoxHTTPContract {
	t.Helper()
	return newNavivoxHTTPContractWithToken(t, baseURL, "nvbx_test_token")
}

func newNavivoxHTTPContractWithToken(t *testing.T, baseURL, token string) navivoxHTTPContract {
	t.Helper()
	return navivoxHTTPContract{t: t, baseURL: strings.TrimRight(baseURL, "/"), token: token}
}

func (c navivoxHTTPContract) JSON(method, path, body string, wantStatus int, out any) {
	c.t.Helper()
	data := c.Raw(method, path, body, wantStatus)
	if err := json.Unmarshal(data, out); err != nil {
		c.t.Fatalf("%s %s decode JSON: %v body=%s", method, path, err, data)
	}
}

func (c navivoxHTTPContract) Raw(method, path, body string, wantStatus int) []byte {
	c.t.Helper()
	resp := c.response(method, path, body)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatalf("%s %s read response body: %v", method, path, err)
	}
	if resp.StatusCode != wantStatus {
		c.t.Fatalf("%s %s status = %d, want %d body=%s", method, path, resp.StatusCode, wantStatus, bytes.TrimSpace(data))
	}
	return data
}

func (c navivoxHTTPContract) response(method, path, body string) *http.Response {
	c.t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	url := c.baseURL + path
	if !strings.HasPrefix(path, "/") {
		url = fmt.Sprintf("%s/%s", c.baseURL, path)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		c.t.Fatalf("%s %s build request: %v", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s execute request: %v", method, path, err)
	}
	return resp
}
