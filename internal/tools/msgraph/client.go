package msgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultMicrosoftGraphBaseURL   = "https://graph.microsoft.com/v1.0"
	defaultMicrosoftGraphUserAgent = "Gormes-Agent/graph-client"
)

type MicrosoftGraphClientOptions struct {
	BaseURL     string
	HTTPClient  MicrosoftGraphHTTPClient
	MaxRetries  int
	Sleep       func(context.Context, time.Duration) error
	UserAgent   string
	RedactExtra []string
}

type MicrosoftGraphClient struct {
	tokenProvider *MicrosoftGraphTokenProvider
	client        MicrosoftGraphHTTPClient
	baseURL       string
	maxRetries    int
	sleep         func(context.Context, time.Duration) error
	userAgent     string
	redact        []string
}

type MicrosoftGraphDownloadResult struct {
	Path        string
	SizeBytes   int64
	ContentType string
}

func NewMicrosoftGraphClient(provider *MicrosoftGraphTokenProvider, opts MicrosoftGraphClientOptions) *MicrosoftGraphClient {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	baseURL := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultMicrosoftGraphBaseURL
	}
	maxRetries := opts.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		}
	}
	userAgent := strings.TrimSpace(opts.UserAgent)
	if userAgent == "" {
		userAgent = defaultMicrosoftGraphUserAgent
	}
	return &MicrosoftGraphClient{
		tokenProvider: provider,
		client:        client,
		baseURL:       baseURL,
		maxRetries:    maxRetries,
		sleep:         sleep,
		userAgent:     userAgent,
		redact:        append([]string(nil), opts.RedactExtra...),
	}
}

func (c *MicrosoftGraphClient) GetJSON(ctx context.Context, path string, params map[string]string) (map[string]interface{}, error) {
	body, err := c.request(ctx, http.MethodGet, path, params)
	if err != nil {
		return nil, err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, c.errorf(MicrosoftGraphRequestUnavailable, "Microsoft Graph response was not valid JSON for GET %s", path)
	}
	return payload, nil
}

func (c *MicrosoftGraphClient) CollectPaginated(ctx context.Context, path string, params map[string]string) ([]map[string]interface{}, error) {
	next := path
	nextParams := params
	var out []map[string]interface{}
	for strings.TrimSpace(next) != "" {
		page, err := c.GetJSON(ctx, next, nextParams)
		if err != nil {
			return nil, err
		}
		value, _ := page["value"].([]interface{})
		for _, item := range value {
			if row, ok := item.(map[string]interface{}); ok {
				out = append(out, row)
			}
		}
		nextLink, _ := page["@odata.nextLink"].(string)
		next = strings.TrimSpace(nextLink)
		nextParams = nil
	}
	return out, nil
}

func (c *MicrosoftGraphClient) DownloadToFile(ctx context.Context, path string, destination string, chunkSize int) (MicrosoftGraphDownloadResult, error) {
	if chunkSize <= 0 {
		chunkSize = 65536
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return MicrosoftGraphDownloadResult{}, c.errorf(MicrosoftGraphRequestUnavailable, "prepare download destination: %s", err)
	}
	tmp := destination + ".part"
	_ = os.Remove(tmp)

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, token, err := c.do(ctx, http.MethodGet, path, nil, attempt > 0 && isMicrosoftGraphUnauthorized(lastErr))
		if err != nil {
			lastErr = err
			if attempt >= c.maxRetries {
				_ = os.Remove(tmp)
				return MicrosoftGraphDownloadResult{}, err
			}
			if sleepErr := c.sleep(ctx, c.retryDelay(nil, attempt)); sleepErr != nil {
				_ = os.Remove(tmp)
				return MicrosoftGraphDownloadResult{}, sleepErr
			}
			continue
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			err := c.statusError(resp, body, token, http.MethodGet, path)
			lastErr = err
			if c.shouldRetry(resp.StatusCode) && attempt < c.maxRetries {
				if resp.StatusCode == http.StatusUnauthorized && c.tokenProvider != nil {
					c.tokenProvider.ClearCache()
				}
				if sleepErr := c.sleep(ctx, c.retryDelay(resp, attempt)); sleepErr != nil {
					_ = os.Remove(tmp)
					return MicrosoftGraphDownloadResult{}, sleepErr
				}
				continue
			}
			_ = os.Remove(tmp)
			return MicrosoftGraphDownloadResult{}, err
		}

		contentType := resp.Header.Get("Content-Type")
		file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			resp.Body.Close()
			_ = os.Remove(tmp)
			return MicrosoftGraphDownloadResult{}, c.errorf(MicrosoftGraphRequestUnavailable, "open partial download: %s", err)
		}
		copyErr := copyMicrosoftGraphStream(file, resp.Body, chunkSize)
		closeErr := file.Close()
		resp.Body.Close()
		if copyErr != nil {
			_ = os.Remove(tmp)
			return MicrosoftGraphDownloadResult{}, c.errorf(MicrosoftGraphRequestUnavailable, "stream Microsoft Graph download: %s", copyErr)
		}
		if closeErr != nil {
			_ = os.Remove(tmp)
			return MicrosoftGraphDownloadResult{}, c.errorf(MicrosoftGraphRequestUnavailable, "close partial download: %s", closeErr)
		}
		if err := os.Rename(tmp, destination); err != nil {
			_ = os.Remove(tmp)
			return MicrosoftGraphDownloadResult{}, c.errorf(MicrosoftGraphRequestUnavailable, "replace Microsoft Graph download: %s", err)
		}
		info, err := os.Stat(destination)
		if err != nil {
			return MicrosoftGraphDownloadResult{}, c.errorf(MicrosoftGraphRequestUnavailable, "stat Microsoft Graph download: %s", err)
		}
		return MicrosoftGraphDownloadResult{Path: destination, SizeBytes: info.Size(), ContentType: contentType}, nil
	}
	_ = os.Remove(tmp)
	if lastErr != nil {
		return MicrosoftGraphDownloadResult{}, lastErr
	}
	return MicrosoftGraphDownloadResult{}, c.errorf(MicrosoftGraphRequestUnavailable, "Microsoft Graph download exhausted retries for GET %s", path)
}

func copyMicrosoftGraphStream(dst io.Writer, src io.Reader, chunkSize int) error {
	buf := make([]byte, chunkSize)
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (c *MicrosoftGraphClient) request(ctx context.Context, method, path string, params map[string]string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, token, err := c.do(ctx, method, path, params, attempt > 0 && isMicrosoftGraphUnauthorized(lastErr))
		if err != nil {
			lastErr = err
			if attempt >= c.maxRetries {
				return nil, err
			}
			if sleepErr := c.sleep(ctx, c.retryDelay(nil, attempt)); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, c.errorf(MicrosoftGraphRequestUnavailable, "read Microsoft Graph response: %s", readErr)
		}
		if resp.StatusCode >= 400 {
			err := c.statusError(resp, body, token, method, path)
			lastErr = err
			if c.shouldRetry(resp.StatusCode) && attempt < c.maxRetries {
				if resp.StatusCode == http.StatusUnauthorized && c.tokenProvider != nil {
					c.tokenProvider.ClearCache()
				}
				if sleepErr := c.sleep(ctx, c.retryDelay(resp, attempt)); sleepErr != nil {
					return nil, sleepErr
				}
				continue
			}
			return nil, err
		}
		return body, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, c.errorf(MicrosoftGraphRequestUnavailable, "Microsoft Graph request exhausted retries for %s %s", method, path)
}

func (c *MicrosoftGraphClient) do(ctx context.Context, method, path string, params map[string]string, forceTokenRefresh bool) (*http.Response, string, error) {
	if c == nil || c.tokenProvider == nil {
		return nil, "", &MicrosoftGraphError{Evidence: MicrosoftGraphRequestUnavailable, Message: "Microsoft Graph token provider is missing"}
	}
	token, err := c.tokenProvider.GetAccessToken(ctx, forceTokenRefresh)
	if err != nil {
		return nil, "", err
	}
	reqURL, err := c.resolveURL(path, params)
	if err != nil {
		return nil, token, c.errorf(MicrosoftGraphRequestUnavailable, "resolve Microsoft Graph URL: %s", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, token, c.errorf(MicrosoftGraphRequestUnavailable, "build Microsoft Graph request: %s", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, token, c.errorf(MicrosoftGraphRequestUnavailable, "Microsoft Graph request failed for %s %s: %s", method, reqURL, err)
	}
	return resp, token, nil
}

func (c *MicrosoftGraphClient) resolveURL(path string, params map[string]string) (string, error) {
	raw := strings.TrimSpace(path)
	if raw == "" {
		return "", fmt.Errorf("empty path")
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		return appendMicrosoftGraphParams(u, params), nil
	}
	u, err := url.Parse(c.baseURL + "/" + strings.TrimLeft(raw, "/"))
	if err != nil {
		return "", err
	}
	return appendMicrosoftGraphParams(u, params), nil
}

func appendMicrosoftGraphParams(u *url.URL, params map[string]string) string {
	if len(params) == 0 {
		return u.String()
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *MicrosoftGraphClient) statusError(resp *http.Response, body []byte, token string, method string, path string) error {
	detail := microsoftGraphErrorDetail(body)
	reqURL := path
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		reqURL = resp.Request.URL.String()
	}
	return c.errorf(MicrosoftGraphRequestUnavailable, "Microsoft Graph API error %d for %s %s: %s", resp.StatusCode, method, reqURL, redactMicrosoftGraphText(detail, append(c.redact, token)...))
}

func (c *MicrosoftGraphClient) shouldRetry(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusUnauthorized || (status >= 500 && status <= 599)
}

func (c *MicrosoftGraphClient) retryDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		raw := strings.TrimSpace(resp.Header.Get("Retry-After"))
		if raw != "" {
			if seconds, err := strconv.ParseFloat(raw, 64); err == nil && seconds >= 0 {
				return time.Duration(seconds * float64(time.Second))
			}
		}
	}
	base := 500 * time.Millisecond
	for i := 0; i < attempt; i++ {
		base *= 2
	}
	return base
}

func (c *MicrosoftGraphClient) errorf(evidence, format string, args ...interface{}) error {
	return &MicrosoftGraphError{
		Evidence: evidence,
		Message:  redactMicrosoftGraphText(fmt.Sprintf(format, args...), c.redact...),
	}
}

func isMicrosoftGraphUnauthorized(err error) bool {
	var graphErr *MicrosoftGraphError
	if !errors.As(err, &graphErr) {
		return false
	}
	return strings.Contains(graphErr.Message, "API error 401")
}
