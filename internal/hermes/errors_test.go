package hermes

import (
	"errors"
	"net"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want ErrorClass
	}{
		{"nil", nil, ClassUnknown},
		{"429", &HTTPError{Status: 429}, ClassRetryable},
		{"500", &HTTPError{Status: 500}, ClassRetryable},
		{"502", &HTTPError{Status: 502}, ClassRetryable},
		{"503", &HTTPError{Status: 503}, ClassRetryable},
		{"504", &HTTPError{Status: 504}, ClassRetryable},
		{"401", &HTTPError{Status: 401}, ClassFatal},
		{"403", &HTTPError{Status: 403}, ClassFatal},
		{"404", &HTTPError{Status: 404}, ClassFatal},
		{"context-length", &HTTPError{Status: 400, Body: "context length exceeded"}, ClassFatal},
		{"plain", errors.New("boom"), ClassUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Classify(c.err); got != c.want {
				t.Errorf("Classify = %v, want %v", got, c.want)
			}
		})
	}
}

func TestClassifyProviderErrorTaxonomy(t *testing.T) {
	cases := []struct {
		name                 string
		err                  error
		wantKind             ProviderErrorKind
		wantClass            ErrorClass
		wantRetryable        bool
		wantCompress         bool
		wantRotateCredential bool
	}{
		{
			name:                 "rate limit status",
			err:                  &HTTPError{Status: 429, Body: `{"error":{"message":"Too many requests","code":"rate_limit_exceeded"}}`},
			wantKind:             ProviderErrorRateLimit,
			wantClass:            ClassRetryable,
			wantRetryable:        true,
			wantRotateCredential: true,
		},
		{
			name:                 "rate limit provider body hint",
			err:                  &HTTPError{Status: 400, Body: `{"error":{"message":"Request rate increased too quickly; please retry after the window resets"}}`},
			wantKind:             ProviderErrorRateLimit,
			wantClass:            ClassRetryable,
			wantRetryable:        true,
			wantRotateCredential: true,
		},
		{
			name:                 "auth status",
			err:                  &HTTPError{Status: 401, Body: `{"error":{"message":"invalid api key"}}`},
			wantKind:             ProviderErrorAuth,
			wantClass:            ClassFatal,
			wantRetryable:        false,
			wantRotateCredential: true,
		},
		{
			name:          "context overflow body hint",
			err:           &HTTPError{Status: 400, Body: `{"error":{"message":"maximum context length exceeded"}}`},
			wantKind:      ProviderErrorContext,
			wantClass:     ClassFatal,
			wantRetryable: false,
			wantCompress:  true,
		},
		{
			name:          "server error retryable",
			err:           &HTTPError{Status: 500, Body: `{"error":{"message":"internal server error"}}`},
			wantKind:      ProviderErrorRetryable,
			wantClass:     ClassRetryable,
			wantRetryable: true,
		},
		{
			name:          "non retryable request failure",
			err:           &HTTPError{Status: 422, Body: `{"error":{"message":"invalid parameter"}}`},
			wantKind:      ProviderErrorNonRetryable,
			wantClass:     ClassFatal,
			wantRetryable: false,
		},
		{
			name:          "transport timeout retryable",
			err:           timeoutError{},
			wantKind:      ProviderErrorRetryable,
			wantClass:     ClassRetryable,
			wantRetryable: true,
		},
		{
			name:          "unknown stays unknown",
			err:           errors.New("boom"),
			wantKind:      ProviderErrorUnknown,
			wantClass:     ClassUnknown,
			wantRetryable: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyProviderError(tc.err)
			if got.Kind != tc.wantKind {
				t.Fatalf("Kind = %q, want %q", got.Kind, tc.wantKind)
			}
			if got.Class != tc.wantClass {
				t.Fatalf("Class = %q, want %q", got.Class, tc.wantClass)
			}
			if got.Retryable != tc.wantRetryable {
				t.Fatalf("Retryable = %v, want %v", got.Retryable, tc.wantRetryable)
			}
			if got.ShouldCompress != tc.wantCompress {
				t.Fatalf("ShouldCompress = %v, want %v", got.ShouldCompress, tc.wantCompress)
			}
			if got.ShouldRotateCredential != tc.wantRotateCredential {
				t.Fatalf("ShouldRotateCredential = %v, want %v", got.ShouldRotateCredential, tc.wantRotateCredential)
			}
			if got.Status != statusOf(tc.err) {
				t.Fatalf("Status = %d, want %d", got.Status, statusOf(tc.err))
			}
			if Classify(tc.err) != tc.wantClass {
				t.Fatalf("Classify compatibility = %q, want %q", Classify(tc.err), tc.wantClass)
			}
		})
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "deadline exceeded" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

var _ net.Error = timeoutError{}

func statusOf(err error) int {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status
	}
	return 0
}

func TestClassifyProviderErrorExtractsBodySignals(t *testing.T) {
	err := &HTTPError{
		Status: 400,
		Body: `{
			"error": {
				"message": "Provider returned error",
				"metadata": {
					"raw": "{\"error\":{\"message\":\"context size has been exceeded\"}}"
				}
			}
		}`,
	}

	got := ClassifyProviderError(err)
	if got.Kind != ProviderErrorContext {
		t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorContext)
	}
	if !got.ShouldCompress {
		t.Fatal("ShouldCompress = false, want true")
	}
	if got.Message != "Provider returned error" {
		t.Fatalf("Message = %q, want structured body message", got.Message)
	}
}

func TestClassifyProviderErrorKindStrings(t *testing.T) {
	for _, kind := range []ProviderErrorKind{
		ProviderErrorUnknown,
		ProviderErrorRateLimit,
		ProviderErrorAuth,
		ProviderErrorContext,
		ProviderErrorRetryable,
		ProviderErrorNonRetryable,
	} {
		if kind.String() == "" {
			t.Fatalf("%#v String() is empty", kind)
		}
	}
}
