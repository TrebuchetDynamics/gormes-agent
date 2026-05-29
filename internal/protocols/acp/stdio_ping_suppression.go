package acp

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"
)

var benignProbeMethods = map[string]struct{}{
	"ping":        {},
	"health":      {},
	"healthcheck": {},
}

// StdioDiagnostics writes ACP stdio diagnostics to the caller-provided stderr
// stream. It never receives or writes protocol stdout.
type StdioDiagnostics struct {
	mu             sync.Mutex
	w              io.Writer
	suppressedByID map[string]uint64
}

func NewStdioDiagnostics(w io.Writer) *StdioDiagnostics {
	return &StdioDiagnostics{
		w:              w,
		suppressedByID: make(map[string]uint64),
	}
}

func (d *StdioDiagnostics) Info(code, message string) {
	if d == nil || d.w == nil {
		return
	}
	code = sanitizeDiagnosticToken(code)
	message = sanitizeDiagnosticMessage(message)
	d.mu.Lock()
	defer d.mu.Unlock()
	if message == "" {
		_, _ = fmt.Fprintf(d.w, "%s\n", code)
		return
	}
	_, _ = fmt.Fprintf(d.w, "%s message=%s\n", code, message)
}

func (d *StdioDiagnostics) BenignProbeSuppressed(method string) {
	if d == nil || d.w == nil {
		return
	}
	method = normalizeProbeMethod(method)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.suppressedByID[method]++
	_, _ = fmt.Fprintf(d.w, "acp_benign_probe_suppressed method=%s count=%d\n", method, d.suppressedByID[method])
}

func (d *StdioDiagnostics) Error(reason, method string) {
	if d == nil || d.w == nil {
		return
	}
	reason = sanitizeDiagnosticToken(reason)
	method = sanitizeDiagnosticMethod(method)
	d.mu.Lock()
	defer d.mu.Unlock()
	if method == "" {
		_, _ = fmt.Fprintf(d.w, "acp_stdio_error reason=%s\n", reason)
		return
	}
	_, _ = fmt.Fprintf(d.w, "acp_stdio_error reason=%s method=%s\n", reason, method)
}

func IsBenignProbeMethod(method string) bool {
	_, ok := benignProbeMethods[normalizeProbeMethod(method)]
	return ok
}

func normalizeProbeMethod(method string) string {
	return strings.ToLower(strings.TrimSpace(method))
}

func sanitizeDiagnosticToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	if len(value) > 80 {
		value = value[:80]
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == ':') {
			return "redacted"
		}
	}
	return value
}

func sanitizeDiagnosticMethod(method string) string {
	method = strings.TrimSpace(method)
	if method == "" {
		return ""
	}
	if len(method) > 80 {
		method = method[:80]
	}
	for _, r := range method {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == ':' || r == '/') {
			return "redacted"
		}
	}
	return method
}

func sanitizeDiagnosticMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if len(message) > 120 {
		message = message[:120]
	}
	for _, r := range message {
		if r == '\n' || r == '\r' || r == '\t' {
			return "redacted"
		}
	}
	return message
}
