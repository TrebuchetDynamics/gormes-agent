package codingagents

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// KnownBackends names the coding-agent binaries Gormes will dispatch to.
// "claude" and "claude-code" are both probed because the Anthropic CLI has
// shipped under both names.
var KnownBackends = []string{"codex", "claude", "claude-code", "opencode"}

// Availability describes whether a coding-agent binary is callable from the
// current PATH and, when available, its self-reported version string.
type Availability struct {
	Name      string
	Available bool
	Version   string
	Error     string
}

// detectTimeout bounds the per-binary --version probe. Coding-agent binaries
// occasionally hang on first-run wizards; a short timeout keeps doctor checks
// snappy.
const detectTimeout = 2 * time.Second

// DetectAvailability runs `<binaryName> --version` with a short timeout and
// returns the Availability result. A missing binary is reported as
// Available=false without an error category leaking through.
func DetectAvailability(ctx context.Context, binaryName string) Availability {
	av := Availability{Name: binaryName}
	if strings.TrimSpace(binaryName) == "" {
		av.Error = "empty binary name"
		return av
	}
	if _, err := exec.LookPath(binaryName); err != nil {
		// Treat "not on PATH" as a clean unavailable, not a hard error.
		return av
	}
	probeCtx, cancel := context.WithTimeout(ctx, detectTimeout)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, binaryName, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			av.Error = "version probe timed out"
			return av
		}
		av.Error = strings.TrimSpace(err.Error())
		return av
	}
	av.Available = true
	av.Version = strings.TrimSpace(string(out))
	return av
}

// DetectAll probes every backend in KnownBackends and returns the results
// keyed by binary name. Callers can distinguish unavailable backends by
// inspecting Available.
func DetectAll(ctx context.Context) map[string]Availability {
	results := make(map[string]Availability, len(KnownBackends))
	for _, name := range KnownBackends {
		results[name] = DetectAvailability(ctx, name)
	}
	return results
}
