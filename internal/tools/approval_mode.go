package tools

import "strings"

const (
	ApprovalModeManual = "manual"
	ApprovalModeSmart  = "smart"
	ApprovalModeOff    = "off"
)

// ApprovalMode describes a normalized dangerous-command approval mode plus
// redacted evidence about whether the value had to fall back to the safe
// manual default.
type ApprovalMode struct {
	Mode      string
	Defaulted bool
	Evidence  map[string]string
}

// NormalizeApprovalMode ports Hermes' approval mode parsing semantics for
// config-loaded values. YAML 1.1 parses bare `off` as boolean false, so false
// must mean the operator intended approval mode "off" rather than defaulting
// to manual.
func NormalizeApprovalMode(value any) ApprovalMode {
	mode := ApprovalMode{Mode: ApprovalModeManual, Evidence: map[string]string{"approval_mode": ApprovalModeManual}}
	switch v := value.(type) {
	case bool:
		if !v {
			mode.Mode = ApprovalModeOff
			mode.Evidence["approval_mode"] = ApprovalModeOff
		}
		return mode
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		switch normalized {
		case ApprovalModeManual, ApprovalModeSmart, ApprovalModeOff:
			mode.Mode = normalized
			mode.Evidence["approval_mode"] = normalized
			return mode
		}
	}
	mode.Defaulted = true
	mode.Evidence["approval_mode_defaulted"] = "true"
	return mode
}
