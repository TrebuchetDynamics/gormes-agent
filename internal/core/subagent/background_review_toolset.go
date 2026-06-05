package subagent

import "github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/review"

// BackgroundReviewToolsetStatus is the terminal status of a background-review
// toolset check or telemetry record. It is distinct from foreground subagent
// status so dashboards can attribute restrictions to background review only.
type BackgroundReviewToolsetStatus = review.BackgroundReviewToolsetStatus

const (
	// BackgroundReviewToolsetAllowed indicates the requested toolset is on the
	// background-review allowlist.
	BackgroundReviewToolsetAllowed = review.BackgroundReviewToolsetAllowed

	// BackgroundReviewToolsetRestricted indicates the requested toolset is not
	// on the background-review allowlist and the request was denied.
	BackgroundReviewToolsetRestricted = review.BackgroundReviewToolsetRestricted

	// BackgroundReviewToolsetUnavailable indicates the request carried no
	// resolvable toolset name (empty/whitespace) and cannot be evaluated.
	BackgroundReviewToolsetUnavailable = review.BackgroundReviewToolsetUnavailable
)

// BackgroundReviewToolsetConfig is a pure, immutable policy object describing
// which toolsets the background review worker may use. It carries no I/O
// handles and never executes a model prompt; constructing it cannot trigger a
// live provider call.
type BackgroundReviewToolsetConfig = review.BackgroundReviewToolsetConfig

// DefaultBackgroundReviewToolsetConfig returns the built-in background-review
// allowlist (memory + skills) used by Hermes parity for review workers.
func DefaultBackgroundReviewToolsetConfig() BackgroundReviewToolsetConfig {
	return review.DefaultBackgroundReviewToolsetConfig()
}

// BackgroundReviewToolsetEvidence is the per-request evidence shape returned
// from CheckToolset. It contains the allowlist plus the denied toolset name
// (when applicable) and is safe to surface in telemetry; it never carries
// model-generated prompt content.
type BackgroundReviewToolsetEvidence = review.BackgroundReviewToolsetEvidence

// BackgroundReviewToolsetTelemetry is the cumulative evidence record published
// alongside background-review status. It mirrors Hermes parity by surfacing the
// allowlist and any denied toolset names without ever embedding the review
// prompt body or any model-generated content.
type BackgroundReviewToolsetTelemetry = review.BackgroundReviewToolsetTelemetry
