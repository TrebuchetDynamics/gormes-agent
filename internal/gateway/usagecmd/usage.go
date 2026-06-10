package usagecmd

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

type FrameSource string

const (
	FrameSourceNone    FrameSource = "unavailable"
	FrameSourceRunning FrameSource = "running turn"
	FrameSourceCached  FrameSource = "cached turn"
)

type FrameSnapshot struct {
	Frame  kernel.RenderFrame
	Source FrameSource
}

func renderUsageField(value string) string {
	value = normalizeUsageFormatting(value)
	value = collapseRedactedUsageAssignments(redaction.RedactSecrets(value))
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"#", "＃",
	)
	value = normalizeUsageFormatting(replacer.Replace(strings.TrimSpace(value)))
	return strings.Join(strings.Fields(value), " ")
}

func normalizeUsageFormatting(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return ' '
		}
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, value)
}

func collapseRedactedUsageAssignments(value string) string {
	replacer := strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"authorization=[redacted]", "[redacted]",
		"bearer=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
	)
	fields := strings.Fields(replacer.Replace(value))
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		lower := strings.ToLower(field)
		nextRedacted := i+1 < len(fields) && strings.Contains(strings.ToLower(fields[i+1]), "[redacted]")
		if usageSecretField(lower) && (strings.Contains(lower, "[redacted]") || nextRedacted || strings.ContainsAny(lower, "=:")) {
			out = append(out, "[redacted]")
			if nextRedacted {
				i++
			}
			continue
		}
		out = append(out, field)
	}
	return strings.Join(out, " ")
}

func usageSecretField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "authorization") || strings.Contains(value, "bearer") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
}

func RenderFrameLines(snapshot FrameSnapshot) []string {
	source := renderUsageField(string(snapshot.Source))
	if source == "" {
		source = string(FrameSourceNone)
	}
	lines := []string{"Usage source: " + source}
	if snapshot.Source == FrameSourceNone {
		return append(lines, "Runtime usage unavailable: no running or cached turn telemetry")
	}
	frame := snapshot.Frame
	model := renderUsageField(frame.Model)
	if model == "" {
		model = "unknown"
	}
	sessionID := renderUsageField(frame.SessionID)
	if sessionID == "" {
		sessionID = "unknown"
	}
	lines = append(lines,
		"Model: "+model,
		"Session: "+sessionID,
		fmt.Sprintf("Tokens: %d in / %d out", nonNegative(frame.Telemetry.TokensInTotal), nonNegative(frame.Telemetry.TokensOutTotal)),
	)
	if frame.Telemetry.LatencyMsLast > 0 {
		lines = append(lines, fmt.Sprintf("Last latency: %d ms", frame.Telemetry.LatencyMsLast))
	}
	if finitePositive(frame.Telemetry.TokensPerSec) {
		lines = append(lines, fmt.Sprintf("Speed: %.2f tokens/sec", frame.Telemetry.TokensPerSec))
	}
	return lines
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsInf(value, 0) && !math.IsNaN(value)
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
