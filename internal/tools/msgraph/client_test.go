package msgraph

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMicrosoftGraphClientAttachesBearerTokenHeader(t *testing.T) {
	var capturedAuth string
	client := NewMicrosoftGraphClient(testMicrosoftGraphTokenProvider(t), MicrosoftGraphClientOptions{
		BaseURL: "https://graph.local/v1.0",
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedAuth = req.Header.Get("Authorization")
			return msgraphStringResponse(http.StatusOK, `{"ok":true}`), nil
		}),
	})

	payload, err := client.GetJSON(context.Background(), "/me", nil)
	if err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("payload = %#v", payload)
	}
	if capturedAuth != "Bearer cached-token" {
		t.Fatalf("Authorization = %q, want bearer token", capturedAuth)
	}
}

func TestMicrosoftGraphClientRetriesRateLimitWithRetryAfter(t *testing.T) {
	calls := 0
	var sleeps []time.Duration
	client := NewMicrosoftGraphClient(testMicrosoftGraphTokenProvider(t), MicrosoftGraphClientOptions{
		BaseURL:    "https://graph.local/v1.0",
		MaxRetries: 2,
		Sleep: func(_ context.Context, d time.Duration) error {
			sleeps = append(sleeps, d)
			return nil
		},
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				resp := msgraphStringResponse(http.StatusTooManyRequests, `{"error":{"code":"TooManyRequests","message":"slow down"}}`)
				resp.Header.Set("Retry-After", "3")
				return resp, nil
			}
			return msgraphStringResponse(http.StatusOK, `{"ok":true}`), nil
		}),
	})

	payload, err := client.GetJSON(context.Background(), "/me", nil)
	if err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("payload = %#v", payload)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if !reflect.DeepEqual(sleeps, []time.Duration{3 * time.Second}) {
		t.Fatalf("sleeps = %v, want 3s retry-after", sleeps)
	}
}

func TestMicrosoftGraphClientCollectPaginatedFollowsNextLink(t *testing.T) {
	var seen []string
	client := NewMicrosoftGraphClient(testMicrosoftGraphTokenProvider(t), MicrosoftGraphClientOptions{
		BaseURL: "https://graph.local/v1.0",
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.URL.String())
			if strings.HasSuffix(req.URL.String(), "/items") {
				return msgraphStringResponse(http.StatusOK, `{"value":[{"id":"1"}],"@odata.nextLink":"https://graph.local/v1.0/items?page=2"}`), nil
			}
			return msgraphStringResponse(http.StatusOK, `{"value":[{"id":"2"}]}`), nil
		}),
	})

	items, err := client.CollectPaginated(context.Background(), "/items", nil)
	if err != nil {
		t.Fatalf("CollectPaginated: %v", err)
	}
	if len(items) != 2 || items[0]["id"] != "1" || items[1]["id"] != "2" {
		t.Fatalf("items = %#v", items)
	}
	wantSeen := []string{"https://graph.local/v1.0/items", "https://graph.local/v1.0/items?page=2"}
	if !reflect.DeepEqual(seen, wantSeen) {
		t.Fatalf("seen URLs = %#v, want %#v", seen, wantSeen)
	}
}

func TestMicrosoftGraphClientDownloadToFileStreamsAndReplacesAtomically(t *testing.T) {
	body := &recordingReadCloser{reader: bytes.NewReader([]byte("meeting-recording"))}
	client := NewMicrosoftGraphClient(testMicrosoftGraphTokenProvider(t), MicrosoftGraphClientOptions{
		BaseURL: "https://graph.local/v1.0",
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       body,
			}
			resp.Header.Set("Content-Type", "video/mp4")
			return resp, nil
		}),
	})
	dest := filepath.Join(t.TempDir(), "recording.mp4")

	result, err := client.DownloadToFile(context.Background(), "/drive/item/content", dest, 4)
	if err != nil {
		t.Fatalf("DownloadToFile: %v", err)
	}
	if got := string(mustReadFile(t, dest)); got != "meeting-recording" {
		t.Fatalf("downloaded file = %q", got)
	}
	if result.ContentType != "video/mp4" || result.SizeBytes != int64(len("meeting-recording")) {
		t.Fatalf("result = %+v", result)
	}
	if len(body.readSizes) < 2 {
		t.Fatalf("read sizes = %v, want multiple streaming reads", body.readSizes)
	}
	if exists(dest + ".part") {
		t.Fatal("partial file still exists after successful download")
	}
}

func TestMicrosoftGraphClientDownloadToFileRemovesPartialOnFailure(t *testing.T) {
	client := NewMicrosoftGraphClient(testMicrosoftGraphTokenProvider(t), MicrosoftGraphClientOptions{
		BaseURL:    "https://graph.local/v1.0",
		MaxRetries: 1,
		Sleep:      func(context.Context, time.Duration) error { return nil },
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return msgraphStringResponse(http.StatusServiceUnavailable, `{"error":{"message":"unavailable"}}`), nil
		}),
	})
	dest := filepath.Join(t.TempDir(), "artifact.bin")

	_, err := client.DownloadToFile(context.Background(), "/drive/item/content", dest, 4)
	if err == nil {
		t.Fatal("DownloadToFile returned nil error for exhausted retries")
	}
	if exists(dest) || exists(dest+".part") {
		t.Fatalf("partial output was not cleaned up: dest=%t part=%t", exists(dest), exists(dest+".part"))
	}
}

func TestMicrosoftGraphClientInvalidJSONReturnsUnavailableError(t *testing.T) {
	client := NewMicrosoftGraphClient(testMicrosoftGraphTokenProvider(t), MicrosoftGraphClientOptions{
		BaseURL: "https://graph.local/v1.0",
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return msgraphStringResponse(http.StatusOK, `not-json`), nil
		}),
	})

	_, err := client.GetJSON(context.Background(), "/me", nil)
	if err == nil {
		t.Fatal("GetJSON returned nil error for invalid JSON")
	}
	var graphErr *MicrosoftGraphError
	if !AsMicrosoftGraphError(err, &graphErr) {
		t.Fatalf("err type = %T, want MicrosoftGraphError", err)
	}
	if graphErr.Evidence != MicrosoftGraphRequestUnavailable {
		t.Fatalf("Evidence = %q, want %q", graphErr.Evidence, MicrosoftGraphRequestUnavailable)
	}
}

func testMicrosoftGraphTokenProvider(t *testing.T) *MicrosoftGraphTokenProvider {
	t.Helper()
	return NewMicrosoftGraphTokenProvider(MicrosoftGraphCredentials{
		TenantID:     "tenant",
		ClientID:     "client",
		ClientSecret: "secret",
	}, MicrosoftGraphTokenProviderOptions{
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return msgraphStringResponse(http.StatusOK, `{"access_token":"cached-token","expires_in":3600}`), nil
		}),
		Now: func() time.Time { return time.Unix(100, 0) },
	})
}

type recordingReadCloser struct {
	reader    *bytes.Reader
	readSizes []int
}

func (r *recordingReadCloser) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.readSizes = append(r.readSizes, n)
	}
	return n, err
}

func (r *recordingReadCloser) Close() error {
	return nil
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := osReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func exists(path string) bool {
	_, err := osStat(path)
	return err == nil
}

var (
	osReadFile = func(path string) ([]byte, error) { return os.ReadFile(path) }
	osStat     = func(path string) (fs.FileInfo, error) { return os.Stat(path) }
)

var _ io.ReadCloser = (*recordingReadCloser)(nil)
