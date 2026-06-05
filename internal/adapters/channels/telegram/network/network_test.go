package network

import (
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestTelegramFallbackIPEnvParsing(t *testing.T) {
	t.Run("filters invalid ipv6 whitespace and leading zeros", func(t *testing.T) {
		got := ParseTelegramFallbackIPEnv("149.154.167.220, bad, 2001:67c:4e8:f004::9,  , 149.154.167.010")
		want := []string{"149.154.167.220"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ParseTelegramFallbackIPEnv() = %#v, want %#v", got, want)
		}
	})

	t.Run("keeps valid csv order", func(t *testing.T) {
		got := ParseTelegramFallbackIPEnv(" 149.154.167.220 , 149.154.167.221 ")
		want := []string{"149.154.167.220", "149.154.167.221"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ParseTelegramFallbackIPEnv() = %#v, want %#v", got, want)
		}
	})

	t.Run("blank returns empty", func(t *testing.T) {
		if got := ParseTelegramFallbackIPEnv(" , "); len(got) != 0 {
			t.Fatalf("ParseTelegramFallbackIPEnv(blank) = %#v, want empty", got)
		}
	})
}

func TestTelegramFallbackTransportPrimaryFallbackAndSticky(t *testing.T) {
	var calls []string
	behavior := map[string]error{
		"primary":         telegramNetworkConnectError("primary timeout"),
		"149.154.167.220": nil,
	}
	transport := newTelegramFallbackTransportWithFactory(
		[]string{"149.154.167.220", "149.154.167.220"},
		recordingTelegramRoundTripper("primary", &calls, behavior),
		func(ip string) http.RoundTripper {
			return recordingTelegramRoundTripper(ip, &calls, behavior)
		},
	)

	resp, err := transport.RoundTrip(telegramNetworkRequest(t, "https://api.telegram.org/botTOKEN/getMe"))
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	_ = resp.Body.Close()
	if want := []string{"primary", "149.154.167.220"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}

	calls = nil
	resp, err = transport.RoundTrip(telegramNetworkRequest(t, "https://api.telegram.org/botTOKEN/sendMessage"))
	if err != nil {
		t.Fatalf("RoundTrip() sticky error = %v", err)
	}
	_ = resp.Body.Close()
	if want := []string{"149.154.167.220"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("sticky calls = %#v, want %#v", calls, want)
	}
}

func TestTelegramFallbackTransportFallbackBoundaries(t *testing.T) {
	t.Run("preserves telegram host on fallback", func(t *testing.T) {
		var seenHost string
		transport := newTelegramFallbackTransportWithFactory(
			[]string{"149.154.167.220"},
			telegramRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, telegramNetworkConnectError("connect refused")
			}),
			func(string) http.RoundTripper {
				return telegramRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					seenHost = req.URL.Hostname()
					if req.Host != "" && req.Host != telegramAPIHost {
						t.Fatalf("req.Host = %q, want %q", req.Host, telegramAPIHost)
					}
					return telegramNetworkOK(req), nil
				})
			},
		)

		resp, err := transport.RoundTrip(telegramNetworkRequest(t, "https://api.telegram.org/botTOKEN/getMe"))
		if err != nil {
			t.Fatalf("RoundTrip() error = %v", err)
		}
		_ = resp.Body.Close()
		if seenHost != telegramAPIHost {
			t.Fatalf("fallback request host = %q, want %q", seenHost, telegramAPIHost)
		}
	})

	t.Run("non telegram host bypasses fallback", func(t *testing.T) {
		var calls []string
		transport := newTelegramFallbackTransportWithFactory(
			[]string{"149.154.167.220"},
			recordingTelegramRoundTripper("primary", &calls, nil),
			func(ip string) http.RoundTripper {
				return recordingTelegramRoundTripper(ip, &calls, nil)
			},
		)

		resp, err := transport.RoundTrip(telegramNetworkRequest(t, "https://example.test/path"))
		if err != nil {
			t.Fatalf("RoundTrip() error = %v", err)
		}
		_ = resp.Body.Close()
		if want := []string{"primary"}; !reflect.DeepEqual(calls, want) {
			t.Fatalf("calls = %#v, want %#v", calls, want)
		}
	})

	t.Run("does not fallback on read timeout", func(t *testing.T) {
		var calls []string
		transport := newTelegramFallbackTransportWithFactory(
			[]string{"149.154.167.220"},
			recordingTelegramRoundTripper("primary", &calls, map[string]error{"primary": &net.OpError{Op: "read", Err: errors.New("read timeout")}}),
			func(ip string) http.RoundTripper {
				return recordingTelegramRoundTripper(ip, &calls, nil)
			},
		)

		_, err := transport.RoundTrip(telegramNetworkRequest(t, "https://api.telegram.org/botTOKEN/getMe"))
		if err == nil {
			t.Fatal("RoundTrip() error = nil, want read timeout")
		}
		if want := []string{"primary"}; !reflect.DeepEqual(calls, want) {
			t.Fatalf("calls = %#v, want %#v", calls, want)
		}
	})

	t.Run("stale sticky falls through to next fallback and all fail returns last", func(t *testing.T) {
		var calls []string
		behavior := map[string]error{
			"primary":         telegramNetworkConnectError("primary timeout"),
			"149.154.167.220": nil,
			"149.154.167.221": nil,
		}
		transport := newTelegramFallbackTransportWithFactory(
			[]string{"149.154.167.220", "149.154.167.221"},
			recordingTelegramRoundTripper("primary", &calls, behavior),
			func(ip string) http.RoundTripper {
				return recordingTelegramRoundTripper(ip, &calls, behavior)
			},
		)
		resp, err := transport.RoundTrip(telegramNetworkRequest(t, "https://api.telegram.org/botTOKEN/getMe"))
		if err != nil {
			t.Fatalf("RoundTrip() initial error = %v", err)
		}
		_ = resp.Body.Close()

		calls = nil
		behavior["149.154.167.220"] = telegramNetworkConnectError("stale fallback")
		resp, err = transport.RoundTrip(telegramNetworkRequest(t, "https://api.telegram.org/botTOKEN/getMe"))
		if err != nil {
			t.Fatalf("RoundTrip() stale sticky error = %v", err)
		}
		_ = resp.Body.Close()
		if want := []string{"149.154.167.220", "149.154.167.221"}; !reflect.DeepEqual(calls, want) {
			t.Fatalf("stale sticky calls = %#v, want %#v", calls, want)
		}

		calls = nil
		behavior["149.154.167.221"] = telegramNetworkConnectError("last fallback")
		_, err = transport.RoundTrip(telegramNetworkRequest(t, "https://api.telegram.org/botTOKEN/getMe"))
		if err == nil || !strings.Contains(err.Error(), "fallback") {
			t.Fatalf("RoundTrip() all fail error = %v, want fallback connect error", err)
		}
		if want := []string{"149.154.167.221", "149.154.167.220"}; !reflect.DeepEqual(calls, want) {
			t.Fatalf("all fail calls = %#v, want %#v", calls, want)
		}
	})
}

type telegramRoundTripFunc func(*http.Request) (*http.Response, error)

func (f telegramRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func recordingTelegramRoundTripper(label string, calls *[]string, behavior map[string]error) http.RoundTripper {
	return telegramRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, label)
		if req.URL.Hostname() != telegramAPIHost && strings.Contains(req.URL.Host, "telegram") {
			return nil, errors.New("telegram host was rewritten unexpectedly")
		}
		if err := behavior[label]; err != nil {
			return nil, err
		}
		return telegramNetworkOK(req), nil
	})
}

func telegramNetworkRequest(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	return req
}

func telegramNetworkOK(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
		Request:    req,
	}
}

func telegramNetworkConnectError(text string) error {
	return &net.OpError{Op: "dial", Err: errors.New(text)}
}
