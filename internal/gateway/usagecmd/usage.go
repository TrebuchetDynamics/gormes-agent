package usagecmd

import (
	"fmt"
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

func RenderFrameLines(snapshot FrameSnapshot) []string {
	lines := []string{"Usage source: " + string(snapshot.Source)}
	if snapshot.Source == FrameSourceNone {
		return append(lines, "Runtime usage unavailable: no running or cached turn telemetry")
	}
	frame := snapshot.Frame
	model := strings.TrimSpace(frame.Model)
	if model == "" {
		model = "unknown"
	}
	sessionID := strings.TrimSpace(frame.SessionID)
	if sessionID == "" {
		sessionID = "unknown"
	}
	lines = append(lines,
		"Model: "+model,
		"Session: "+sessionID,
		fmt.Sprintf("Tokens: %d in / %d out", frame.Telemetry.TokensInTotal, frame.Telemetry.TokensOutTotal),
	)
	if frame.Telemetry.LatencyMsLast > 0 {
		lines = append(lines, fmt.Sprintf("Last latency: %d ms", frame.Telemetry.LatencyMsLast))
	}
	if frame.Telemetry.TokensPerSec > 0 {
		lines = append(lines, fmt.Sprintf("Speed: %.2f tokens/sec", frame.Telemetry.TokensPerSec))
	}
	return lines
}
