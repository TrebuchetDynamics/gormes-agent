package network

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	telegramAPIHost        = "api.telegram.org"
	telegramFallbackIPEnv  = "TELEGRAM_FALLBACK_IPS"
	telegramFallbackOffEnv = "HERMES_TELEGRAM_DISABLE_FALLBACK_IPS"
	gormesFallbackOffEnv   = "GORMES_TELEGRAM_DISABLE_FALLBACK_IPS"
)

// ParseTelegramFallbackIPEnv validates TELEGRAM_FALLBACK_IPS-style CSV values.
// It mirrors Hermes' Bot API fallback contract: public IPv4 addresses only.
func ParseTelegramFallbackIPEnv(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	return normalizeTelegramFallbackIPs(parts)
}

func normalizeTelegramFallbackIPs(values []string) []string {
	var out []string
	for _, value := range values {
		raw := strings.TrimSpace(value)
		if raw == "" {
			continue
		}
		addr, err := netip.ParseAddr(raw)
		if err != nil || !addr.Is4() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() {
			continue
		}
		out = append(out, addr.String())
	}
	return out
}

type TelegramFallbackTransport struct {
	primary     http.RoundTripper
	fallbackIPs []string
	fallbacks   map[string]http.RoundTripper

	mu       sync.Mutex
	stickyIP string
}

func NewTelegramFallbackTransport(fallbackIPs []string, primary http.RoundTripper) *TelegramFallbackTransport {
	return newTelegramFallbackTransportWithFactory(fallbackIPs, primary, func(ip string) http.RoundTripper {
		return newTelegramFallbackDialTransport(ip, primary)
	})
}

func newTelegramFallbackTransportWithFactory(fallbackIPs []string, primary http.RoundTripper, factory func(string) http.RoundTripper) *TelegramFallbackTransport {
	if primary == nil {
		primary = http.DefaultTransport
	}
	if factory == nil {
		factory = func(string) http.RoundTripper { return primary }
	}
	ips := dedupeTelegramFallbackIPs(normalizeTelegramFallbackIPs(fallbackIPs))
	fallbacks := make(map[string]http.RoundTripper, len(ips))
	for _, ip := range ips {
		if rt := factory(ip); rt != nil {
			fallbacks[ip] = rt
		}
	}
	return &TelegramFallbackTransport{primary: primary, fallbackIPs: ips, fallbacks: fallbacks}
}

func dedupeTelegramFallbackIPs(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func (t *TelegramFallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	if req == nil || req.URL == nil || req.URL.Hostname() != telegramAPIHost || len(t.fallbackIPs) == 0 {
		return t.primary.RoundTrip(req)
	}

	attempts := t.attemptOrder()
	var lastErr error
	for _, ip := range attempts {
		rt := t.primary
		if ip != "" {
			rt = t.fallbacks[ip]
			if rt == nil {
				rt = t.primary
			}
		}
		resp, err := rt.RoundTrip(req)
		if err == nil {
			if ip != "" {
				t.setStickyIP(ip)
			}
			return resp, nil
		}
		lastErr = err
		if !telegramIsRetryableConnectError(err) {
			return nil, err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("telegram fallback transport exhausted without response")
}

func (t *TelegramFallbackTransport) attemptOrder() []string {
	t.mu.Lock()
	sticky := t.stickyIP
	t.mu.Unlock()
	if sticky == "" {
		out := make([]string, 0, len(t.fallbackIPs)+1)
		out = append(out, "")
		out = append(out, t.fallbackIPs...)
		return out
	}
	out := make([]string, 0, len(t.fallbackIPs))
	out = append(out, sticky)
	for _, ip := range t.fallbackIPs {
		if ip != sticky {
			out = append(out, ip)
		}
	}
	return out
}

func (t *TelegramFallbackTransport) setStickyIP(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stickyIP = ip
}

func newTelegramFallbackDialTransport(ip string, primary http.RoundTripper) http.RoundTripper {
	base, ok := primary.(*http.Transport)
	if !ok || base == nil {
		base = http.DefaultTransport.(*http.Transport)
	}
	tr := base.Clone()
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err == nil && host == telegramAPIHost {
			addr = net.JoinHostPort(ip, port)
		}
		return dialer.DialContext(ctx, network, addr)
	}
	return tr
}

func telegramIsRetryableConnectError(err error) bool {
	if err == nil {
		return false
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		op := strings.ToLower(opErr.Op)
		return op == "dial" || op == "connect"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "connect timeout") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "connection reset") ||
		strings.Contains(text, "network is unreachable") ||
		strings.Contains(text, "no such host")
}

func HTTPClientFromEnv() *http.Client {
	if telegramFallbackDisabled() {
		return &http.Client{}
	}
	ips := ParseTelegramFallbackIPEnv(os.Getenv(telegramFallbackIPEnv))
	if len(ips) == 0 {
		return &http.Client{}
	}
	return &http.Client{Transport: NewTelegramFallbackTransport(ips, http.DefaultTransport)}
}

func telegramFallbackDisabled() bool {
	return envBool(telegramFallbackOffEnv) || envBool(gormesFallbackOffEnv)
}

func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
