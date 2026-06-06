package ssrfguard

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/browser/navigation"
)

const (
	EvidenceConfigInvalid     = "ssrf_guard_config_invalid"
	EvidencePrivateURLBlocked = "private_url_blocked"
)

// Bool is a normalized bool-like browser safety config value.
type Bool struct {
	Value    bool
	Evidence string
}

// Options are the pure inputs needed before a browser provider
// receives a navigation URL.
type Options struct {
	CloudConfigured         bool
	AllowPrivateURLs        any
	AutoLocalForPrivateURLs any
	CDPOverride             bool
	CamofoxMode             bool
}

// Decision is the pure pre-navigation cloud safety decision.
type Decision struct {
	Allowed  bool
	Evidence string
	Route    navigation.Route
}

// CoerceBool normalizes bool-like config values without using
// language truthiness for strings such as "false".
func CoerceBool(raw any, fallback bool) Bool {
	switch value := raw.(type) {
	case nil:
		return Bool{Value: fallback}
	case bool:
		return Bool{Value: value}
	case string:
		switch normalizeBoolString(value) {
		case "1", "true", "yes", "on":
			return Bool{Value: true}
		case "0", "false", "no", "off":
			return Bool{Value: false}
		}
	case int:
		if value == 0 {
			return Bool{Value: false}
		}
		if value == 1 {
			return Bool{Value: true}
		}
	}
	return Bool{Value: fallback, Evidence: EvidenceConfigInvalid}
}

// Check determines whether rawURL may proceed to its selected
// browser route without starting a browser or resolving DNS.
func Check(taskID, rawURL string, opts Options) Decision {
	allowPrivate := CoerceBool(opts.AllowPrivateURLs, false)
	autoLocal := CoerceBool(opts.AutoLocalForPrivateURLs, true)

	route := navigation.RouteNavigation(
		taskID,
		rawURL,
		opts.CloudConfigured,
		autoLocal.Value,
		opts.CDPOverride,
		opts.CamofoxMode,
	)
	decision := Decision{Allowed: true, Route: route}

	if allowPrivate.Evidence != "" {
		decision.Allowed = false
		decision.Evidence = allowPrivate.Evidence
		return decision
	}
	if autoLocal.Evidence != "" {
		decision.Allowed = false
		decision.Evidence = autoLocal.Evidence
		return decision
	}

	if !opts.CloudConfigured || opts.CDPOverride || opts.CamofoxMode || allowPrivate.Value || route.ForceLocal {
		return decision
	}

	if navigation.IsPrivateNavigationTarget(rawURL) {
		decision.Allowed = false
		decision.Evidence = EvidencePrivateURLBlocked
	}
	return decision
}

func normalizeBoolString(value string) string {
	trimmed := strings.TrimSpace(value)
	for len(trimmed) >= 2 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first != '"' && first != '\'') || first != last {
			break
		}
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	return strings.ToLower(trimmed)
}
