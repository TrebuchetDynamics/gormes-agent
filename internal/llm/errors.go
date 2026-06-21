package llm

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorClass int

const (
	ClassUnknown ErrorClass = iota
	ClassRetryable
	ClassFatal
)

func (c ErrorClass) String() string {
	switch c {
	case ClassRetryable:
		return "retryable"
	case ClassFatal:
		return "fatal"
	}
	return "unknown"
}

type HTTPError struct {
	Status     int
	Body       string
	RetryAfter time.Duration
	Headers    map[string]string
}

func (e *HTTPError) Error() string {
	status := strings.TrimSpace(http.StatusText(e.Status))
	body := sanitizeProviderErrorBody(e.Body)
	if body == "" {
		return status
	}
	if status == "" || strings.EqualFold(status, body) {
		return body
	}
	return status + ": " + body
}

const maxRetryAfterHint = 16 * time.Second

func newHTTPError(status int, body string, header http.Header) *HTTPError {
	return &HTTPError{
		Status:     status,
		Body:       body,
		RetryAfter: parseRetryAfterHint(header.Get("Retry-After"), body, time.Now()),
		Headers:    captureStreamDiagnosticHeaders(header),
	}
}

func (e *HTTPError) StreamDiagnostics() StreamDiagnostics {
	if e == nil {
		return StreamDiagnostics{}
	}
	return sanitizeStreamDiagnostics(StreamDiagnostics{
		HTTPStatus: e.Status,
		Headers:    e.Headers,
	})
}

func sanitizeStreamDiagnostics(diag StreamDiagnostics) StreamDiagnostics {
	diag.Headers = sanitizeStreamDiagnosticHeaders(diag.Headers)
	if diag.HTTPStatus < 0 {
		diag.HTTPStatus = 0
	}
	if diag.Bytes < 0 {
		diag.Bytes = 0
	}
	if diag.Chunks < 0 {
		diag.Chunks = 0
	}
	if diag.Elapsed < 0 {
		diag.Elapsed = 0
	}
	if diag.TimeToFirstByte < 0 {
		diag.TimeToFirstByte = 0
	}
	return diag
}

func captureStreamDiagnosticHeaders(header http.Header) map[string]string {
	if len(header) == 0 {
		return nil
	}
	values := make(map[string]string, len(streamDiagnosticHeaderAllowlist))
	for name := range streamDiagnosticHeaderAllowlist {
		value := strings.TrimSpace(header.Get(name))
		if value == "" {
			continue
		}
		if len(value) > 120 {
			value = value[:120]
		}
		values[name] = value
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func sanitizeStreamDiagnosticHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	values := make(map[string]string, len(headers))
	for rawName, rawValue := range headers {
		name := strings.ToLower(strings.TrimSpace(rawName))
		if _, ok := streamDiagnosticHeaderAllowlist[name]; !ok {
			continue
		}
		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}
		if len(value) > 120 {
			value = value[:120]
		}
		values[name] = value
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

var streamDiagnosticHeaderAllowlist = map[string]struct{}{
	"cf-ray":                {},
	"cf-cache-status":       {},
	"x-openrouter-provider": {},
	"x-openrouter-model":    {},
	"x-openrouter-id":       {},
	"x-request-id":          {},
	"x-vercel-id":           {},
	"via":                   {},
	"server":                {},
	"x-forwarded-for":       {},
}

type ProviderErrorKind string

const (
	ProviderErrorUnknown           ProviderErrorKind = "unknown"
	ProviderErrorRateLimit         ProviderErrorKind = "rate_limit"
	ProviderErrorAuth              ProviderErrorKind = "auth"
	ProviderErrorAuthPermanent     ProviderErrorKind = "auth_permanent"
	ProviderErrorBilling           ProviderErrorKind = "billing"
	ProviderErrorContext           ProviderErrorKind = "context"
	ProviderErrorContextOverflow   ProviderErrorKind = "context_overflow"
	ProviderErrorImageTooLarge     ProviderErrorKind = "image_too_large"
	ProviderErrorRetryable         ProviderErrorKind = "retryable"
	ProviderErrorNonRetryable      ProviderErrorKind = "non_retryable"
	ProviderErrorOverloaded        ProviderErrorKind = "overloaded"
	ProviderErrorServerError       ProviderErrorKind = "server_error"
	ProviderErrorTimeout           ProviderErrorKind = "timeout"
	ProviderErrorPayloadTooLarge   ProviderErrorKind = "payload_too_large"
	ProviderErrorModelNotFound     ProviderErrorKind = "model_not_found"
	ProviderErrorPolicyBlocked     ProviderErrorKind = "provider_policy_blocked"
	ProviderErrorFormatError       ProviderErrorKind = "format_error"
	ProviderErrorThinkingSignature            ProviderErrorKind = "thinking_signature"
	ProviderErrorLongContextTier              ProviderErrorKind = "long_context_tier"
	ProviderErrorContentPolicyBlocked         ProviderErrorKind = "content_policy_blocked"
	ProviderErrorMultimodalToolUnsupported    ProviderErrorKind = "multimodal_tool_content_unsupported"
	ProviderErrorInvalidEncryptedContent      ProviderErrorKind = "invalid_encrypted_content"
	ProviderErrorLlamaCppGrammarPattern       ProviderErrorKind = "llama_cpp_grammar_pattern"
	ProviderErrorOAuthLongContextForbidden    ProviderErrorKind = "oauth_long_context_beta_forbidden"
)

func (k ProviderErrorKind) String() string {
	if k == "" {
		return string(ProviderErrorUnknown)
	}
	return string(k)
}

type ProviderErrorClassification struct {
	Kind                   ProviderErrorKind
	Class                  ErrorClass
	Status                 int
	Message                string
	Retryable              bool
	ShouldCompress         bool
	ShouldRotateCredential bool
	ShouldFallback         bool
}

// Classify inspects an error produced anywhere in the hermes pipeline and
// categorises it so the kernel can decide whether to retry or abort.
func Classify(err error) ErrorClass {
	return ClassifyProviderError(err).Class
}

// ClassifyProviderError returns a structured provider-error envelope for
// status reporting and future recovery decisions. Retry-After hint parsing is
// owned by a separate 4.H slice, so this function only classifies failures.
func ClassifyProviderError(err error) ProviderErrorClassification {
	if err == nil {
		return providerError(ProviderErrorUnknown, ClassUnknown, 0, "", false)
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		message, combined, code := providerHTTPErrorText(httpErr)

		if httpErr.Status == http.StatusTooManyRequests ||
			containsAny(combined, rateLimitPatterns) ||
			isRateLimitCode(code) {
			out := providerError(ProviderErrorRateLimit, ClassRetryable, httpErr.Status, message, true)
			out.ShouldRotateCredential = true
			out.ShouldFallback = true
			return out
		}
		if httpErr.Status == http.StatusUnauthorized ||
			httpErr.Status == http.StatusForbidden ||
			containsAny(combined, authPatterns) {
			out := providerError(ProviderErrorAuth, ClassFatal, httpErr.Status, message, false)
			out.ShouldRotateCredential = true
			out.ShouldFallback = true
			return out
		}
		// Anthropic thinking block signature mismatch or frozen-block mutation (400).
		// Classified before generic 400 handlers so it gets retryable recovery.
		if httpErr.Status == http.StatusBadRequest &&
			strings.Contains(combined, "thinking") &&
			(strings.Contains(combined, "signature") ||
				strings.Contains(combined, "cannot be modified") ||
				strings.Contains(combined, "must remain as they were")) {
			return providerError(ProviderErrorThinkingSignature, ClassRetryable, httpErr.Status, message, true)
		}
		// Anthropic long-context tier gate (429 "extra usage" + "long context").
		if httpErr.Status == http.StatusTooManyRequests &&
			strings.Contains(combined, "extra usage") &&
			strings.Contains(combined, "long context") {
			out := providerError(ProviderErrorLongContextTier, ClassRetryable, httpErr.Status, message, true)
			out.ShouldCompress = true
			return out
		}
		// Anthropic OAuth 1M-context beta not available for this subscription.
		if httpErr.Status == http.StatusBadRequest &&
			strings.Contains(combined, "long context beta") &&
			strings.Contains(combined, "not yet available") {
			return providerError(ProviderErrorOAuthLongContextForbidden, ClassFatal, httpErr.Status, message, false)
		}
		if containsAny(combined, imageTooLargePatterns) {
			return providerError(ProviderErrorImageTooLarge, ClassFatal, httpErr.Status, message, false)
		}
		if containsAny(combined, multimodalToolContentPatterns) {
			return providerError(ProviderErrorMultimodalToolUnsupported, ClassFatal, httpErr.Status, message, false)
		}
		if containsAny(combined, contentPolicyPatterns) {
			return providerError(ProviderErrorContentPolicyBlocked, ClassFatal, httpErr.Status, message, false)
		}
		if containsAny(combined, providerPolicyBlockedPatterns) {
			out := providerError(ProviderErrorPolicyBlocked, ClassFatal, httpErr.Status, message, false)
			out.ShouldFallback = true
			return out
		}
		if containsAny(combined, modelNotFoundPatterns) {
			out := providerError(ProviderErrorModelNotFound, ClassFatal, httpErr.Status, message, false)
			out.ShouldFallback = true
			return out
		}
		if containsAny(combined, billingPatterns) {
			out := providerError(ProviderErrorBilling, ClassFatal, httpErr.Status, message, false)
			out.ShouldRotateCredential = true
			out.ShouldFallback = true
			return out
		}
		if httpErr.Status == http.StatusPaymentRequired {
			out := providerError(ProviderErrorBilling, ClassFatal, httpErr.Status, message, false)
			out.ShouldRotateCredential = true
			out.ShouldFallback = true
			return out
		}
		if containsAny(combined, invalidEncryptedContentPatterns) {
			return providerError(ProviderErrorInvalidEncryptedContent, ClassFatal, httpErr.Status, message, false)
		}
		if containsAny(combined, requestValidationPatterns) || requestValidationCodes[strings.ToLower(code)] {
			return providerError(ProviderErrorFormatError, ClassFatal, httpErr.Status, message, false)
		}
		if httpErr.Status == http.StatusRequestEntityTooLarge ||
			containsAny(combined, payloadTooLargePatterns) {
			out := providerError(ProviderErrorPayloadTooLarge, ClassFatal, httpErr.Status, message, false)
			out.ShouldCompress = true
			return out
		}
		if containsAny(combined, contextPatterns) || isContextCode(code) {
			out := providerError(ProviderErrorContext, ClassFatal, httpErr.Status, message, false)
			out.ShouldCompress = true
			return out
		}
		switch httpErr.Status {
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return providerError(ProviderErrorRetryable, ClassRetryable, httpErr.Status, message, true)
		}
		if httpErr.Status == 529 {
			return providerError(ProviderErrorOverloaded, ClassRetryable, httpErr.Status, message, true)
		}
		if httpErr.Status >= 400 && httpErr.Status < 500 {
			return providerError(ProviderErrorNonRetryable, ClassFatal, httpErr.Status, message, false)
		}
		return providerError(ProviderErrorUnknown, ClassUnknown, httpErr.Status, message, false)
	}
	message := err.Error()
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		out := providerError(ProviderErrorRetryable, ClassRetryable, 0, message, true)
		out.ShouldFallback = true
		return out
	}
	combined := strings.ToLower(message)
	if containsAny(combined, rateLimitPatterns) {
		out := providerError(ProviderErrorRateLimit, ClassRetryable, 0, message, true)
		out.ShouldRotateCredential = true
		out.ShouldFallback = true
		return out
	}
	if containsAny(combined, billingPatterns) {
		out := providerError(ProviderErrorBilling, ClassFatal, 0, message, false)
		out.ShouldRotateCredential = true
		out.ShouldFallback = true
		return out
	}
	if containsAny(combined, authPatterns) {
		out := providerError(ProviderErrorAuth, ClassFatal, 0, message, false)
		out.ShouldRotateCredential = true
		out.ShouldFallback = true
		return out
	}
	if containsAny(combined, contentPolicyPatterns) {
		return providerError(ProviderErrorContentPolicyBlocked, ClassFatal, 0, message, false)
	}
	if containsAny(combined, imageTooLargePatterns) {
		return providerError(ProviderErrorImageTooLarge, ClassFatal, 0, message, false)
	}
	if containsAny(combined, modelNotFoundPatterns) {
		out := providerError(ProviderErrorModelNotFound, ClassFatal, 0, message, false)
		out.ShouldFallback = true
		return out
	}
	if containsAny(combined, contextPatterns) {
		out := providerError(ProviderErrorContext, ClassFatal, 0, message, false)
		out.ShouldCompress = true
		return out
	}
	if containsAny(combined, llamaCppGrammarPatterns) {
		return providerError(ProviderErrorLlamaCppGrammarPattern, ClassFatal, 0, message, false)
	}
	if containsAny(combined, sslTransientPatterns) {
		out := providerError(ProviderErrorTimeout, ClassRetryable, 0, message, true)
		out.ShouldFallback = true
		return out
	}
	if containsAny(combined, serverDisconnectPatterns) {
		out := providerError(ProviderErrorRetryable, ClassRetryable, 0, message, true)
		out.ShouldFallback = true
		return out
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return providerError(ProviderErrorRetryable, ClassRetryable, 0, message, true)
	}
	if containsAny(combined, timeoutMessagePatterns) {
		out := providerError(ProviderErrorTimeout, ClassRetryable, 0, message, true)
		out.ShouldFallback = true
		return out
	}
	return providerError(ProviderErrorUnknown, ClassUnknown, 0, message, false)
}

func providerError(kind ProviderErrorKind, class ErrorClass, status int, message string, retryable bool) ProviderErrorClassification {
	return ProviderErrorClassification{
		Kind:      kind,
		Class:     class,
		Status:    status,
		Message:   message,
		Retryable: retryable,
	}
}

var rateLimitPatterns = []string{
	"rate limit",
	"rate_limit",
	"too many requests",
	"throttled",
	"throttlingexception",
	"servicequotaexceededexception",
	"resource_exhausted",
	"requests per minute",
	"tokens per minute",
	"requests per day",
	"try again in",
	"please retry after",
	"retry after",
	"rate increased too quickly",
	"too many concurrent requests",
	// Gemini weekly / account usage limits. Both trigger credential rotation so
	// the pool can try the next pooled credential. Mirrors Hermes
	// fix(credential-pool): correct pool rotation when weekly usage limit is
	// reached (4117fc364).
	"gousagelimit",
	"usage limit reached",
	"usage limit has been reached",
}

var authPatterns = []string{
	"invalid api key",
	"invalid_api_key",
	"authentication",
	"unauthorized",
	"forbidden",
	"invalid token",
	"token expired",
	"token revoked",
	"access denied",
}

var imageTooLargePatterns = []string{
	"image too large",
	"image payload too large",
	"image is too large",
	"max image size",
	"maximum image size",
	"image size exceeds",
	"unsupported image dimensions",
	"unsupported image dimension",
	"image dimensions are too large",
	"image exceeds",
	"image_too_large",
	"image dimensions exceed",
	"dimensions exceed max allowed size",
	"max allowed size: 8000",
}

var contextPatterns = []string{
	"context length",
	"context size",
	"maximum context",
	"token limit",
	"too many tokens",
	"reduce the length",
	"exceeds the limit",
	"context window",
	"prompt is too long",
	"prompt exceeds max length",
	"maximum number of tokens",
	"exceeds the max_model_len",
	"max_model_len",
	"prompt length",
	"input is too long",
	"maximum model length",
	"context length exceeded",
	"slot context",
	"n_ctx_slot",
	"超过最大长度",
	"上下文长度",
	"max input token",
	"input token",
	"exceeds the maximum number of input tokens",
	"truncating input",
	"max_tokens",
}

// contentPolicyPatterns matches provider safety-filter rejections that are
// deterministic per-request — retrying the same prompt will fail again.
var contentPolicyPatterns = []string{
	"flagged for possible cybersecurity risk",
	"trusted access for cyber",
	"violates our usage policies",
	"violates openai's usage policies",
	"your request was flagged by",
	"prompt was flagged by our safety",
	"responses cannot be generated due to safety",
	"content_filter",
	"responsibleaipolicyviolation",
}

// multimodalToolContentPatterns matches 400-class errors where the provider
// rejected list-type content in a tool message (expects string only).
// Recovery: strip image parts from tool messages, retry.
var multimodalToolContentPatterns = []string{
	"text is not set",
	"tool message content must be a string",
	"tool content must be a string",
	"tool message must be a string",
	"expected string, got list",
	"expected string, got array",
	"tool_call.content must be string",
}

// providerPolicyBlockedPatterns matches OpenRouter aggregator-level policy
// rejections (account data/privacy setting blocks every endpoint).
var providerPolicyBlockedPatterns = []string{
	"no endpoints available matching your guardrail",
	"no endpoints available matching your data policy",
	"no endpoints found matching your data policy",
}

// requestValidationPatterns matches request format errors that will fail on
// every retry unchanged — allows fast format_error abort instead of looping.
//
// NOTE: "invalid_request_error" is deliberately omitted from this slice.
// OpenAI stamps that same type on genuine context-overflow 400s
// ("context_length_exceeded" code), so including it here would route real
// overflows into FormatError instead of Context, preventing compression.
// The unambiguous signals are the explicit "unsupported/unknown parameter"
// substrings and the provider-level error codes in requestValidationCodes
// (Hermes error_classifier.py parity, fix from 2ce3ae3d1).
var requestValidationPatterns = []string{
	"unknown parameter",
	"unsupported parameter",
	"unrecognized request argument",
	"unknown_parameter",
	"unsupported_parameter",
}

// requestValidationCodes are provider-level error codes that unambiguously
// signal an unsupported/unknown parameter — distinct from the generic
// "invalid_request_error" code that providers also use for overflow.
var requestValidationCodes = map[string]bool{
	"unknown_parameter":    true,
	"unsupported_parameter": true,
}

// modelNotFoundPatterns matches model availability errors.
var modelNotFoundPatterns = []string{
	"is not a valid model",
	"invalid model",
	"model not found",
	"model_not_found",
	"does not exist",
	"no such model",
	"unknown model",
	"unsupported model",
}

// billingPatterns matches confirmed credit/quota exhaustion.
var billingPatterns = []string{
	"insufficient credits",
	"insufficient_quota",
	"insufficient balance",
	"credit balance",
	"credits exhausted",
	"credits have been exhausted",
	"no usable credits",
	"top up your credits",
	"payment required",
	"billing hard limit",
	"exceeded your current quota",
	"account is deactivated",
	"plan does not include",
	"out of funds",
	"run out of funds",
	"balance_depleted",
	"model_not_supported_on_free_tier",
	"not available on the free tier",
}

// payloadTooLargePatterns matches 413-class errors embedded in message text.
var payloadTooLargePatterns = []string{
	"request entity too large",
	"payload too large",
	"error code: 413",
}

// serverDisconnectPatterns are ambiguous transport-level closes that Hermes
// treats as potential context overflow when the session is large.
var serverDisconnectPatterns = []string{
	"server disconnected",
	"peer closed connection",
	"connection reset by peer",
	"connection was closed",
	"network connection lost",
	"unexpected eof",
	"incomplete chunked read",
}

// sslTransientPatterns are SSL/TLS mid-stream failures that should retry but
// NOT trigger context compression (unlike server disconnect, these are pure
// network-layer hiccups unrelated to request size).
var sslTransientPatterns = []string{
	"bad record mac",
	"ssl alert",
	"tls alert",
	"ssl handshake failure",
	"tlsv1 alert",
	"sslv3 alert",
	"bad_record_mac",
	"ssl_alert",
	"tls_alert",
	"[ssl:",
}

// invalidEncryptedContentPatterns matches Responses API replay-blob rejection.
var invalidEncryptedContentPatterns = []string{
	"invalid_encrypted_content",
	"encrypted_content",
	"previous_response_id",
}

// llamaCppGrammarPatterns matches llama.cpp json-schema-to-grammar failures
// caused by unsupported regex escapes in pattern/format fields.
var llamaCppGrammarPatterns = []string{
	"grammar error",
	"failed to parse grammar",
	"json-schema-to-grammar",
	"unsupported json schema",
}

// oauthLongContextPatterns matches Anthropic OAuth subscription rejections
// of the 1M-context beta (disable beta header and retry).
var oauthLongContextPatterns = []string{
	"anthropic-beta",
	"output-128k",
	"interleaved-thinking",
	"long context",
}

var timeoutMessagePatterns = []string{
	"timed out",
	"turn timed out",
	"request timed out",
	"deadline exceeded",
	"operation timed out",
	"upstream timed out",
}

func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func isRateLimitCode(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "rate_limit", "rate_limit_exceeded", "resource_exhausted", "throttled", "throttlingexception":
		return true
	}
	return false
}

func isContextCode(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "context_length_exceeded", "context_overflow", "max_tokens_exceeded":
		return true
	}
	return false
}

func isUnsupportedTemperatureError(err error) bool {
	return isUnsupportedParameterError(err, "temperature")
}

func isUnsupportedParameterError(err error, param string) bool {
	param = strings.ToLower(strings.TrimSpace(param))
	if err == nil || param == "" {
		return false
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		_, combined, code := providerHTTPErrorText(httpErr)
		return isUnsupportedParameterSignal(combined, code, param)
	}
	return isUnsupportedParameterSignal(strings.ToLower(err.Error()), "", param)
}

var unsupportedParameterPatterns = []string{
	"unsupported parameter",
	"unsupported_parameter",
	"not supported",
	"does not support",
	"unknown parameter",
	"unrecognized request argument",
	"unrecognized parameter",
	"invalid parameter",
}

func isUnsupportedParameterSignal(combined, code, param string) bool {
	combined = strings.ToLower(combined)
	param = strings.ToLower(strings.TrimSpace(param))
	if param == "" || !strings.Contains(combined, param) {
		return false
	}
	if isUnsupportedParameterCode(code) {
		return true
	}
	return containsAny(combined, unsupportedParameterPatterns)
}

func isUnsupportedParameterCode(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "unsupported_parameter", "unknown_parameter":
		return true
	}
	return false
}

func providerHTTPErrorText(err *HTTPError) (message, combined, code string) {
	body := strings.TrimSpace(err.Body)
	message = sanitizeProviderErrorBody(body)
	parts := []string{body, message}
	if body != "" {
		var decoded any
		if json.Unmarshal([]byte(body), &decoded) == nil {
			extractedMessage, extractedCode, extractedRaw := providerBodySignals(decoded)
			if extractedMessage != "" {
				message = extractedMessage
				parts = append(parts, extractedMessage)
			}
			if extractedCode != "" {
				code = extractedCode
				parts = append(parts, extractedCode)
			}
			if extractedRaw != "" {
				parts = append(parts, extractedRaw)
			}
		}
	}
	combined = strings.ToLower(strings.Join(parts, " "))
	return message, combined, code
}

func sanitizeProviderErrorBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	lower := strings.ToLower(body)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") || strings.Contains(lower, "<svg") {
		return "provider returned HTML error body"
	}
	var decoded any
	if json.Unmarshal([]byte(body), &decoded) == nil {
		message, code, _ := providerBodySignals(decoded)
		switch {
		case message != "":
			body = message
		case code != "":
			body = code
		default:
			return "provider returned JSON error body"
		}
	}
	body = strings.ReplaceAll(body, "\r", " ")
	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 480 {
		return body[:467] + "...[truncated]"
	}
	return body
}

func providerBodySignals(v any) (message, code, raw string) {
	obj, ok := v.(map[string]any)
	if !ok {
		return "", "", ""
	}
	if errObj, ok := obj["error"].(map[string]any); ok {
		message = stringField(errObj["message"])
		code = firstStringField(errObj, "code", "type")
		if metadata, ok := errObj["metadata"].(map[string]any); ok {
			raw = stringField(metadata["raw"])
			if raw != "" {
				if rawMessage, rawCode := providerRawSignals(raw); rawMessage != "" || rawCode != "" {
					raw = strings.TrimSpace(raw + " " + rawMessage + " " + rawCode)
				}
			}
		}
		return message, code, raw
	}
	if errText := stringField(obj["error"]); errText != "" {
		code = errText
		message = firstStringField(obj, "error_description", "message", "detail")
		if message == "" {
			message = errText
		}
		return message, code, ""
	}
	message = firstStringField(obj, "message", "detail", "error_description")
	code = firstStringField(obj, "code", "error_code", "type")
	return message, code, ""
}

func providerRawSignals(raw string) (message, code string) {
	var decoded any
	if json.Unmarshal([]byte(raw), &decoded) != nil {
		return "", ""
	}
	message, code, _ = providerBodySignals(decoded)
	return message, code
}

func firstStringField(obj map[string]any, names ...string) string {
	for _, name := range names {
		if s := stringField(obj[name]); s != "" {
			return s
		}
	}
	return ""
}

func stringField(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		return x.String()
	case map[string]any:
		// Nested dict: providers like HF router embed a structured object in
		// error.message (e.g. {"message":{"type":"Bad Request","message":"..."}}}).
		// Walk the priority key list to extract the most descriptive string —
		// mirrors Hermes fix(agent): summarize structured provider error messages.
		for _, key := range []string{"message", "detail", "error", "code", "type"} {
			if s := stringField(x[key]); s != "" {
				return s
			}
		}
		return ""
	case []any:
		// List: join non-empty string representations of each element.
		var parts []string
		for _, item := range x {
			if s := stringField(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "; ")
	default:
		return ""
	}
}

func parseRetryAfterHint(headerValue, body string, now time.Time) time.Duration {
	if d := parseRetryAfterHeader(headerValue, now); d > 0 {
		return capRetryAfterHint(d)
	}
	if d := parseRetryAfterBody(body); d > 0 {
		return capRetryAfterHint(d)
	}
	return 0
}

func parseRetryAfterHeader(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0
	}
	return when.Sub(now)
}

func parseRetryAfterBody(body string) time.Duration {
	body = strings.TrimSpace(body)
	if body == "" {
		return 0
	}
	var decoded any
	if json.Unmarshal([]byte(body), &decoded) != nil {
		return 0
	}
	return retryAfterFromValue(decoded)
}

func retryAfterFromValue(v any) time.Duration {
	switch x := v.(type) {
	case map[string]any:
		if d := retryAfterDuration(x["retry_after"]); d > 0 {
			return d
		}
		if d := retryAfterDuration(x["retryAfter"]); d > 0 {
			return d
		}
		for _, child := range x {
			if d := retryAfterFromValue(child); d > 0 {
				return d
			}
		}
	case []any:
		for _, child := range x {
			if d := retryAfterFromValue(child); d > 0 {
				return d
			}
		}
	}
	return 0
}

func retryAfterDuration(v any) time.Duration {
	switch x := v.(type) {
	case float64:
		if x <= 0 {
			return 0
		}
		return time.Duration(x * float64(time.Second))
	case json.Number:
		seconds, err := strconv.ParseFloat(x.String(), 64)
		if err != nil || seconds <= 0 {
			return 0
		}
		return time.Duration(seconds * float64(time.Second))
	case string:
		seconds, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil || seconds <= 0 {
			return 0
		}
		return time.Duration(seconds * float64(time.Second))
	default:
		return 0
	}
}

func capRetryAfterHint(d time.Duration) time.Duration {
	if d > maxRetryAfterHint {
		return maxRetryAfterHint
	}
	return d
}
