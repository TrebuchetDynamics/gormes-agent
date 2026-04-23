package hermes

import (
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
	Status int
	Body   string

	retryAfter time.Duration
}

// RetryAfterer is implemented by retryable provider errors that carry an
// explicit Retry-After backpressure hint.
type RetryAfterer interface {
	RetryAfter() (time.Duration, bool)
}

func (e *HTTPError) Error() string {
	return http.StatusText(e.Status) + ": " + e.Body
}

// RetryAfter reports provider backpressure duration when the upstream server
// sent an HTTP Retry-After header on a retryable response.
func (e *HTTPError) RetryAfter() (time.Duration, bool) {
	if e == nil || e.retryAfter <= 0 {
		return 0, false
	}
	return e.retryAfter, true
}

func newHTTPError(resp *http.Response, body []byte) *HTTPError {
	return &HTTPError{
		Status:     resp.StatusCode,
		Body:       string(body),
		retryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()),
	}
}

func parseRetryAfter(raw string, now time.Time) time.Duration {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
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

// Classify inspects an error produced anywhere in the hermes pipeline and
// categorises it so the kernel can decide whether to retry or abort.
func Classify(err error) ErrorClass {
	if err == nil {
		return ClassUnknown
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Status {
		case 429, 500, 502, 503, 504:
			return ClassRetryable
		case 401, 403, 404:
			return ClassFatal
		}
		if strings.Contains(strings.ToLower(httpErr.Body), "context length") {
			return ClassFatal
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ClassRetryable
	}
	return ClassUnknown
}
