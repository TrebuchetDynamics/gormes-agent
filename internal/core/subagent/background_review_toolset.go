package subagent

import (
	"fmt"
	"sort"
	"strings"
)

// BackgroundReviewToolsetStatus is the terminal status of a background-review
// toolset check or telemetry record. It is distinct from foreground subagent
// status so dashboards can attribute restrictions to background review only.
type BackgroundReviewToolsetStatus string

const (
	// BackgroundReviewToolsetAllowed indicates the requested toolset is on the
	// background-review allowlist.
	BackgroundReviewToolsetAllowed BackgroundReviewToolsetStatus = "background_review_toolset_allowed"

	// BackgroundReviewToolsetRestricted indicates the requested toolset is not
	// on the background-review allowlist and the request was denied.
	BackgroundReviewToolsetRestricted BackgroundReviewToolsetStatus = "background_review_toolset_restricted"

	// BackgroundReviewToolsetUnavailable indicates the request carried no
	// resolvable toolset name (empty/whitespace) and cannot be evaluated.
	BackgroundReviewToolsetUnavailable BackgroundReviewToolsetStatus = "background_review_toolset_unavailable"
)

// defaultBackgroundReviewToolsets is the positive allowlist mirroring Hermes
// run_agent.py background-review agent toolsets ("memory", "skills"). The
// background reviewer never gets terminal, send_message, delegate_task,
// execute_code, browser, or provider-management tools.
var defaultBackgroundReviewToolsets = []string{"memory", "skills"}

// BackgroundReviewToolsetConfig is a pure, immutable policy object describing
// which toolsets the background review worker may use. It carries no I/O
// handles and never executes a model prompt; constructing it cannot trigger a
// live provider call.
type BackgroundReviewToolsetConfig struct {
	allowed map[string]struct{}
}

// DefaultBackgroundReviewToolsetConfig returns the built-in background-review
// allowlist (memory + skills) used by Hermes parity for review workers.
func DefaultBackgroundReviewToolsetConfig() BackgroundReviewToolsetConfig {
	allowed := make(map[string]struct{}, len(defaultBackgroundReviewToolsets))
	for _, name := range defaultBackgroundReviewToolsets {
		allowed[name] = struct{}{}
	}
	return BackgroundReviewToolsetConfig{allowed: allowed}
}

// AllowedToolsets returns a sorted copy of the allowlist. Mutating the
// returned slice does not affect the config.
func (c BackgroundReviewToolsetConfig) AllowedToolsets() []string {
	out := make([]string, 0, len(c.allowed))
	for name := range c.allowed {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// AllowsToolset reports whether the named toolset is on the allowlist.
// Names are normalised by trimming whitespace and lowercasing.
func (c BackgroundReviewToolsetConfig) AllowsToolset(name string) bool {
	normalised := normaliseBackgroundReviewToolsetName(name)
	if normalised == "" {
		return false
	}
	_, ok := c.allowed[normalised]
	return ok
}

// BackgroundReviewToolsetEvidence is the per-request evidence shape returned
// from CheckToolset. It contains the allowlist plus the denied toolset name
// (when applicable) and is safe to surface in telemetry; it never carries
// model-generated prompt content.
type BackgroundReviewToolsetEvidence struct {
	Status          BackgroundReviewToolsetStatus
	AllowedToolsets []string
	DeniedToolset   string
	Reason          string
}

// CheckToolset evaluates a single requested toolset against the allowlist.
// The boolean is true only when the toolset is allowed; the evidence value is
// always populated.
func (c BackgroundReviewToolsetConfig) CheckToolset(name string) (BackgroundReviewToolsetEvidence, bool) {
	allowed := c.AllowedToolsets()
	normalised := normaliseBackgroundReviewToolsetName(name)
	if normalised == "" {
		return BackgroundReviewToolsetEvidence{
			Status:          BackgroundReviewToolsetUnavailable,
			AllowedToolsets: allowed,
			Reason:          "background review toolset request carried no resolvable name",
		}, false
	}
	if _, ok := c.allowed[normalised]; ok {
		return BackgroundReviewToolsetEvidence{
			Status:          BackgroundReviewToolsetAllowed,
			AllowedToolsets: allowed,
		}, true
	}
	return BackgroundReviewToolsetEvidence{
		Status:          BackgroundReviewToolsetRestricted,
		AllowedToolsets: allowed,
		DeniedToolset:   normalised,
		Reason: fmt.Sprintf(
			"background review workers only allow toolsets %s",
			strings.Join(allowed, ","),
		),
	}, false
}

// BackgroundReviewToolsetTelemetry is the cumulative evidence record published
// alongside background-review status. It mirrors Hermes parity by surfacing the
// allowlist and any denied toolset names without ever embedding the review
// prompt body or any model-generated content.
type BackgroundReviewToolsetTelemetry struct {
	Status          BackgroundReviewToolsetStatus
	AllowedToolsets []string
	DeniedToolsets  []string
}

// Telemetry returns a telemetry record for a background-review run. The prompt
// argument is intentionally accepted and discarded so callers cannot wire
// model prompt content into the evidence surface by mistake.
func (c BackgroundReviewToolsetConfig) Telemetry(_ string, denied []string) BackgroundReviewToolsetTelemetry {
	telemetry := BackgroundReviewToolsetTelemetry{
		AllowedToolsets: c.AllowedToolsets(),
	}
	seen := make(map[string]struct{}, len(denied))
	for _, name := range denied {
		normalised := normaliseBackgroundReviewToolsetName(name)
		if normalised == "" {
			continue
		}
		if _, dup := seen[normalised]; dup {
			continue
		}
		seen[normalised] = struct{}{}
		telemetry.DeniedToolsets = append(telemetry.DeniedToolsets, normalised)
	}
	sort.Strings(telemetry.DeniedToolsets)
	if len(telemetry.DeniedToolsets) == 0 {
		telemetry.Status = BackgroundReviewToolsetAllowed
	} else {
		telemetry.Status = BackgroundReviewToolsetRestricted
	}
	return telemetry
}

// String returns a stable, prompt-free rendering safe for log/audit surfaces.
func (t BackgroundReviewToolsetTelemetry) String() string {
	return fmt.Sprintf(
		"background_review_toolset status=%s allowed=[%s] denied=[%s]",
		t.Status,
		strings.Join(t.AllowedToolsets, ","),
		strings.Join(t.DeniedToolsets, ","),
	)
}

// PromptContent is intentionally always empty. Background-review telemetry
// must not carry model-generated review prompt content; the method exists so
// future surfaces cannot accidentally introduce a prompt-bearing field.
func (BackgroundReviewToolsetTelemetry) PromptContent() string { return "" }

func normaliseBackgroundReviewToolsetName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
