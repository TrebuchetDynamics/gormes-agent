package llm

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimitBucket is one provider rate-limit window (for example requests per
// minute or tokens per hour) parsed from x-ratelimit-* response headers.
type RateLimitBucket struct {
	Limit      int
	Remaining  int
	ResetAfter time.Duration
	CapturedAt time.Time
}

func (b RateLimitBucket) Used() int {
	if b.Limit <= 0 {
		return 0
	}
	return b.Limit - b.normalizedRemaining()
}

func (b RateLimitBucket) normalizedRemaining() int {
	if b.Remaining < 0 {
		return 0
	}
	if b.Limit > 0 && b.Remaining > b.Limit {
		return b.Limit
	}
	return b.Remaining
}

func (b RateLimitBucket) UsagePercent() float64 {
	if b.Limit <= 0 {
		return 0
	}
	return float64(b.Used()) / float64(b.Limit) * 100
}

func (b RateLimitBucket) RemainingDuration(now time.Time) time.Duration {
	if b.ResetAfter <= 0 {
		return 0
	}
	if now.IsZero() || b.CapturedAt.IsZero() {
		return b.ResetAfter
	}
	remaining := b.ResetAfter - now.Sub(b.CapturedAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RateLimitState is the latest Nous/OpenRouter/OpenAI-compatible rate-limit
// read model captured from provider response headers.
type RateLimitState struct {
	RequestsMinute RateLimitBucket
	RequestsHour   RateLimitBucket
	TokensMinute   RateLimitBucket
	TokensHour     RateLimitBucket
	CapturedAt     time.Time
	Provider       string
}

func (s RateLimitState) HasData() bool {
	return !s.CapturedAt.IsZero()
}

func (s RateLimitState) Age(now time.Time) time.Duration {
	if !s.HasData() {
		return time.Duration(math.MaxInt64)
	}
	if now.IsZero() {
		now = time.Now()
	}
	age := now.Sub(s.CapturedAt)
	if age < 0 {
		return 0
	}
	return age
}

// ParseRateLimitHeaders parses Hermes' supported x-ratelimit-* schema into a
// RateLimitState. It returns ok=false only when no x-ratelimit headers are
// present. Malformed individual bucket values degrade to zero-valued buckets.
func ParseRateLimitHeaders(headers http.Header, provider string, capturedAt time.Time) (RateLimitState, bool) {
	lowered := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		lowered[strings.ToLower(key)] = values[0]
	}
	var hasAny bool
	for key := range lowered {
		if strings.HasPrefix(key, "x-ratelimit-") {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return RateLimitState{}, false
	}
	if capturedAt.IsZero() {
		capturedAt = time.Now()
	}
	bucket := func(resource, suffix string) RateLimitBucket {
		tag := resource + suffix
		return RateLimitBucket{
			Limit:      safeRateLimitInt(lowered["x-ratelimit-limit-"+tag]),
			Remaining:  safeRateLimitInt(lowered["x-ratelimit-remaining-"+tag]),
			ResetAfter: safeRateLimitDuration(lowered["x-ratelimit-reset-"+tag]),
			CapturedAt: capturedAt,
		}
	}
	return RateLimitState{
		RequestsMinute: bucket("requests", ""),
		RequestsHour:   bucket("requests", "-1h"),
		TokensMinute:   bucket("tokens", ""),
		TokensHour:     bucket("tokens", "-1h"),
		CapturedAt:     capturedAt,
		Provider:       strings.TrimSpace(provider),
	}, true
}

func safeRateLimitInt(raw string) int {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value <= 0 {
		return 0
	}
	if value > float64(int(^uint(0)>>1)) {
		return int(^uint(0) >> 1)
	}
	return int(value)
}

func safeRateLimitDuration(raw string) time.Duration {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value <= 0 {
		return 0
	}
	return time.Duration(value * float64(time.Second))
}

func FormatRateLimitDisplay(state RateLimitState) string {
	return FormatRateLimitDisplayAt(state, time.Now())
}

func FormatRateLimitDisplayAt(state RateLimitState, now time.Time) string {
	if !state.HasData() {
		return "No rate limit data yet — make an API request first."
	}
	providerLabel := strings.TrimSpace(state.Provider)
	if providerLabel == "" {
		providerLabel = "Provider"
	} else {
		providerLabel = strings.Title(providerLabel)
	}
	lines := []string{
		fmt.Sprintf("%s Rate Limits (captured %s):", providerLabel, rateLimitFreshness(state.Age(now))),
		"",
		rateLimitBucketLine("Requests/min", state.RequestsMinute, now),
		rateLimitBucketLine("Requests/hr", state.RequestsHour, now),
		"",
		rateLimitBucketLine("Tokens/min", state.TokensMinute, now),
		rateLimitBucketLine("Tokens/hr", state.TokensHour, now),
	}
	var warnings []string
	for _, item := range []struct {
		label  string
		bucket RateLimitBucket
	}{
		{"requests/min", state.RequestsMinute},
		{"requests/hr", state.RequestsHour},
		{"tokens/min", state.TokensMinute},
		{"tokens/hr", state.TokensHour},
	} {
		if item.bucket.Limit > 0 && item.bucket.UsagePercent() >= 80 {
			warnings = append(warnings, fmt.Sprintf("  ⚠ %s at %.0f%% — resets in %s", item.label, item.bucket.UsagePercent(), formatRateLimitSeconds(item.bucket.RemainingDuration(now))))
		}
	}
	if len(warnings) > 0 {
		lines = append(lines, "")
		lines = append(lines, warnings...)
	}
	return strings.Join(lines, "\n")
}

func FormatRateLimitCompact(state RateLimitState) string {
	return FormatRateLimitCompactAt(state, time.Now())
}

func FormatRateLimitCompactAt(state RateLimitState, now time.Time) string {
	if !state.HasData() {
		return "No rate limit data."
	}
	var parts []string
	if state.RequestsMinute.Limit > 0 {
		parts = append(parts, fmt.Sprintf("RPM: %d/%d", state.RequestsMinute.normalizedRemaining(), state.RequestsMinute.Limit))
	}
	if state.RequestsHour.Limit > 0 {
		parts = append(parts, fmt.Sprintf("RPH: %s/%s (resets %s)", formatRateLimitCount(state.RequestsHour.normalizedRemaining()), formatRateLimitCount(state.RequestsHour.Limit), formatRateLimitSeconds(state.RequestsHour.RemainingDuration(now))))
	}
	if state.TokensMinute.Limit > 0 {
		parts = append(parts, fmt.Sprintf("TPM: %s/%s", formatRateLimitCount(state.TokensMinute.normalizedRemaining()), formatRateLimitCount(state.TokensMinute.Limit)))
	}
	if state.TokensHour.Limit > 0 {
		parts = append(parts, fmt.Sprintf("TPH: %s/%s (resets %s)", formatRateLimitCount(state.TokensHour.normalizedRemaining()), formatRateLimitCount(state.TokensHour.Limit), formatRateLimitSeconds(state.TokensHour.RemainingDuration(now))))
	}
	if len(parts) == 0 {
		return "No rate limit bucket data."
	}
	return strings.Join(parts, " | ")
}

func rateLimitBucketLine(label string, bucket RateLimitBucket, now time.Time) string {
	const labelWidth = 14
	if bucket.Limit <= 0 {
		return fmt.Sprintf("  %-*s  (no data)", labelWidth, label)
	}
	pct := bucket.UsagePercent()
	return fmt.Sprintf(
		"  %-*s %s %5.1f%%  %s/%s used  (%s left, resets in %s)",
		labelWidth,
		label,
		rateLimitBar(pct, 20),
		pct,
		formatRateLimitCount(bucket.Used()),
		formatRateLimitCount(bucket.Limit),
		formatRateLimitCount(bucket.normalizedRemaining()),
		formatRateLimitSeconds(bucket.RemainingDuration(now)),
	)
}

func rateLimitBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func formatRateLimitCount(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return strconv.Itoa(n)
}

func formatRateLimitSeconds(d time.Duration) string {
	seconds := int(d / time.Second)
	if seconds < 0 {
		seconds = 0
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		minutes := seconds / 60
		rem := seconds % 60
		if rem == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, rem)
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func rateLimitFreshness(age time.Duration) string {
	if age < 5*time.Second {
		return "just now"
	}
	if age < time.Minute {
		return fmt.Sprintf("%ds ago", int(age/time.Second))
	}
	return formatRateLimitSeconds(age) + " ago"
}
