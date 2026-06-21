package llm

import (
	"errors"
	"net"
	"strings"
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

func TestClassifyProviderError_GenericTimeoutMessages(t *testing.T) {
	cases := []struct {
		name    string
		message string
	}{
		{"turn timed out", "claude CLI turn timed out"},
		{"request timed out", "request timed out after 120s"},
		{"deadline exceeded", "deadline exceeded"},
		{"operation timed out", "operation timed out waiting for provider"},
		{"upstream timed out", "upstream timed out while streaming"},
		{"timed out", "provider timed out"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyProviderError(errors.New(tc.message))
			if got.Kind != ProviderErrorTimeout {
				t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorTimeout)
			}
			if got.Class != ClassRetryable {
				t.Fatalf("Class = %q, want %q", got.Class, ClassRetryable)
			}
			if !got.Retryable {
				t.Fatal("Retryable = false, want true")
			}
			if !got.ShouldFallback {
				t.Fatal("ShouldFallback = false, want true")
			}
			if Classify(errors.New(tc.message)) != ClassRetryable {
				t.Fatal("Classify compatibility did not return retryable")
			}
		})
	}
}

func TestClassifyProviderError_ContentPolicyBlocked(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"openai flagged", `{"error":{"message":"Your request was flagged by our safety filters.","code":"content_filter"}}`},
		{"openai cybersecurity", `{"error":{"message":"flagged for possible cybersecurity risk"}}`},
		{"anthropic safety", `{"error":{"message":"prompt was flagged by our safety system"}}`},
		{"azure policy violation", `{"error":{"code":"responsibleaipolicyviolation","message":"blocked"}}`},
		{"violates policies", `{"error":{"message":"This request violates our usage policies."}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyProviderError(&HTTPError{Status: 400, Body: tc.body})
			if got.Kind != ProviderErrorContentPolicyBlocked {
				t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorContentPolicyBlocked)
			}
			if got.Class != ClassFatal {
				t.Fatalf("Class = %q, want ClassFatal", got.Class)
			}
			if got.Retryable {
				t.Fatal("Retryable = true, want false (deterministic rejection)")
			}
		})
	}
}

func TestClassifyProviderError_MultimodalToolContentUnsupported(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"xiaomi mimo text not set", `{"error":{"code":"400","message":"Param Incorrect","param":"text is not set"}}`},
		{"tool message must be string", `{"error":{"message":"tool message content must be a string"}}`},
		{"expected string got list", `{"error":{"message":"expected string, got list"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyProviderError(&HTTPError{Status: 400, Body: tc.body})
			if got.Kind != ProviderErrorMultimodalToolUnsupported {
				t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorMultimodalToolUnsupported)
			}
			if got.Class != ClassFatal {
				t.Fatalf("Class = %q, want ClassFatal", got.Class)
			}
		})
	}
}

func TestClassifyProviderError_BillingPatterns(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"insufficient credits", `{"error":{"message":"You have insufficient credits."}}`},
		{"payment required status", ``},
		{"balance depleted", `{"error":{"code":"balance_depleted","message":"Account balance depleted"}}`},
		{"out of funds", `{"error":{"message":"Your account is out of funds."}}`},
		{"free tier blocked", `{"error":{"message":"model_not_supported_on_free_tier"}}`},
	}
	errs := []error{
		&HTTPError{Status: 400, Body: cases[0].body},
		&HTTPError{Status: 402},
		&HTTPError{Status: 400, Body: cases[2].body},
		&HTTPError{Status: 400, Body: cases[3].body},
		&HTTPError{Status: 400, Body: cases[4].body},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyProviderError(errs[i])
			if got.Kind != ProviderErrorBilling {
				t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorBilling)
			}
			if got.Class != ClassFatal {
				t.Fatalf("Class = %q, want ClassFatal", got.Class)
			}
			if !got.ShouldRotateCredential {
				t.Fatal("ShouldRotateCredential = false, want true")
			}
			if !got.ShouldFallback {
				t.Fatal("ShouldFallback = false, want true")
			}
		})
	}
}

func TestClassifyProviderError_ModelNotFound(t *testing.T) {
	got := ClassifyProviderError(&HTTPError{Status: 404, Body: `{"error":{"message":"The model gpt-5 does not exist","code":"model_not_found"}}`})
	if got.Kind != ProviderErrorModelNotFound {
		t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorModelNotFound)
	}
	if !got.ShouldFallback {
		t.Fatal("ShouldFallback = false, want true")
	}
}

func TestClassifyProviderError_OpenRouterPolicyBlocked(t *testing.T) {
	body := `{"error":{"message":"No endpoints available matching your guardrail restrictions and data policy. Configure: https://openrouter.ai/settings/privacy"}}`
	got := ClassifyProviderError(&HTTPError{Status: 404, Body: body})
	if got.Kind != ProviderErrorPolicyBlocked {
		t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorPolicyBlocked)
	}
	if !got.ShouldFallback {
		t.Fatal("ShouldFallback = false, want true")
	}
}

func TestClassifyProviderError_ImageTooLargeAnthropicVariants(t *testing.T) {
	cases := []string{
		`image exceeds 5 MB maximum`,
		`image dimensions exceed max allowed size: 8000 pixels`,
		`dimensions exceed max allowed size`,
	}
	for _, msg := range cases {
		body := `{"error":{"message":"` + msg + `"}}`
		got := ClassifyProviderError(&HTTPError{Status: 400, Body: body})
		if got.Kind != ProviderErrorImageTooLarge {
			t.Fatalf("msg %q: Kind = %q, want %q", msg, got.Kind, ProviderErrorImageTooLarge)
		}
	}
}

func TestClassifyProviderError_Overloaded529(t *testing.T) {
	got := ClassifyProviderError(&HTTPError{Status: 529, Body: `{"error":{"message":"Overloaded"}}`})
	if got.Kind != ProviderErrorOverloaded {
		t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorOverloaded)
	}
	if !got.Retryable {
		t.Fatal("Retryable = false, want true")
	}
}

func TestClassifyProviderError_BillingMessageOnly(t *testing.T) {
	got := ClassifyProviderError(errors.New("insufficient credits to continue"))
	if got.Kind != ProviderErrorBilling {
		t.Fatalf("Kind = %q, want %q", got.Kind, ProviderErrorBilling)
	}
	if !got.ShouldRotateCredential {
		t.Fatal("ShouldRotateCredential = false, want true")
	}
}

func TestDefaultChainErrorClassifierTimeoutRetriesThenFallback(t *testing.T) {
	classifier := &DefaultChainErrorClassifier{MaxRetriesPerProvider: 2}
	classification := ProviderErrorClassification{Kind: ProviderErrorTimeout, Class: ClassRetryable, Retryable: true}

	if got := classifier.Decide(classification, 0); got != ChainDecisionRetry {
		t.Fatalf("attempt 0 decision = %q, want %q", got, ChainDecisionRetry)
	}
	if got := classifier.Decide(classification, 1); got != ChainDecisionRetry {
		t.Fatalf("attempt 1 decision = %q, want %q", got, ChainDecisionRetry)
	}
	if got := classifier.Decide(classification, 2); got != ChainDecisionFallback {
		t.Fatalf("attempt 2 decision = %q, want %q", got, ChainDecisionFallback)
	}
}

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
		ProviderErrorImageTooLarge,
		ProviderErrorRetryable,
		ProviderErrorNonRetryable,
		ProviderErrorTimeout,
	} {
		if kind.String() == "" {
			t.Fatalf("%#v String() is empty", kind)
		}
	}
}

func TestHTTPErrorErrorSanitizesHTMLProviderBodies(t *testing.T) {
	err := &HTTPError{Status: 403, Body: `<html><body><svg><path d="secret"></path></svg>Forbidden</body></html>`}

	if got := err.Error(); got != "Forbidden: provider returned HTML error body" {
		t.Fatalf("Error() = %q, want sanitized HTML body", got)
	}
	classification := ClassifyProviderError(err)
	if classification.Kind != ProviderErrorAuth {
		t.Fatalf("Kind = %q, want %q", classification.Kind, ProviderErrorAuth)
	}
	if classification.Message != "provider returned HTML error body" {
		t.Fatalf("Message = %q, want sanitized classification message", classification.Message)
	}
}

func TestHTTPErrorErrorExtractsDetailJSONWithoutLeakingRawBody(t *testing.T) {
	err := &HTTPError{Status: 401, Body: `{"detail":"Unauthorized"}`}

	got := err.Error()
	if strings.Contains(got, "{") || strings.Contains(got, "detail") {
		t.Fatalf("Error() leaked raw JSON body: %q", got)
	}
	if got != "Unauthorized" {
		t.Fatalf("Error() = %q, want deduped Unauthorized", got)
	}

	classification := ClassifyProviderError(err)
	if classification.Kind != ProviderErrorAuth {
		t.Fatalf("Kind = %q, want %q", classification.Kind, ProviderErrorAuth)
	}
	if classification.Message != "Unauthorized" {
		t.Fatalf("Message = %q, want extracted detail", classification.Message)
	}
}

// TestClassifyProviderError_UnsupportedParamNotMisclassifiedAsOverflow verifies
// Hermes error_classifier.py parity (fix 2ce3ae3d1): a 400 whose body contains
// "max_tokens" (an overflow pattern) but whose message says "unsupported
// parameter" must be classified as FormatError, not ContextOverflow — the
// validation guard must fire before the context check.
// Separately, a genuine OpenAI overflow 400 using "invalid_request_error" as
// the error type (but "context_length_exceeded" as the code) must still be
// classified as ContextOverflow with ShouldCompress=true.
func TestClassifyProviderError_UnsupportedParamNotMisclassifiedAsOverflow(t *testing.T) {
	// GPT-5-style "unsupported parameter" 400: message contains "max_tokens"
	// (overflow pattern) but ALSO contains "unsupported parameter".
	unsupportedParam := &HTTPError{
		Status: 400,
		Body: `{"error":{"code":"unknown_parameter","message":"Unsupported parameter: 'max_tokens' is not supported with this model. Use 'max_completion_tokens' instead."}}`,
	}
	got := ClassifyProviderError(unsupportedParam)
	if got.Kind != ProviderErrorFormatError {
		t.Errorf("unsupported param 400: Kind = %q, want %q (should not misclassify as context overflow)", got.Kind, ProviderErrorFormatError)
	}
	if got.ShouldCompress {
		t.Errorf("unsupported param 400: ShouldCompress = true, want false (must not trigger compression loop)")
	}

	// Genuine OpenAI overflow: "invalid_request_error" type + "context_length_exceeded" code.
	// Must reach the context overflow path, not FormatError.
	overflow := &HTTPError{
		Status: 400,
		Body: `{"error":{"message":"This model's maximum context length is 4096 tokens, however you requested 5001 tokens. Please reduce your prompt.","type":"invalid_request_error","param":"messages","code":"context_length_exceeded"}}`,
	}
	gotOvf := ClassifyProviderError(overflow)
	if gotOvf.Kind != ProviderErrorContext {
		t.Errorf("overflow 400: Kind = %q, want %q (invalid_request_error must not block overflow classification)", gotOvf.Kind, ProviderErrorContext)
	}
	if !gotOvf.ShouldCompress {
		t.Errorf("overflow 400: ShouldCompress = false, want true")
	}
}

func TestSanitizeProviderErrorBodyNestedStructuredMessage(t *testing.T) {
	// Mirrors Hermes fix(agent): summarize structured provider error messages
	// (e4452ffb8). Some providers (e.g. HF router) embed a structured object in
	// the error.message field instead of a plain string. The sanitizer must
	// recursively extract the human-readable message rather than returning a
	// generic "provider returned JSON error body" fallback.
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "nested dict in error.message",
			body: `{"error":{"message":{"type":"Bad Request","code":"context_length_exceeded","message":"This model's maximum context length is 4096 tokens"}}}`,
			want: "This model's maximum context length is 4096 tokens",
		},
		{
			name: "flat string error.message (unchanged)",
			body: `{"error":{"message":"rate limit exceeded"}}`,
			want: "rate limit exceeded",
		},
		{
			name: "list in error.message",
			body: `{"error":{"message":["token limit exceeded","please reduce your input"]}}`,
			want: "token limit exceeded; please reduce your input",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeProviderErrorBody(tc.body)
			if !strings.Contains(got, tc.want) {
				t.Errorf("sanitizeProviderErrorBody(%q) = %q, want it to contain %q", tc.body, got, tc.want)
			}
		})
	}
}
