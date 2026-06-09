package usagecmd

import (
	"fmt"
	"math"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"#", "＃",
	)
	return strings.Join(strings.Fields(replacer.Replace(strings.TrimSpace(value))), " ")
}

func RenderFrameLines(snapshot FrameSnapshot) []string {
	lines := []string{"Usage source: " + string(snapshot.Source)}
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
