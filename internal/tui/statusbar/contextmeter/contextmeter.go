package contextmeter

import "math"

// BarWidth is the default width of the status-bar context meter.
const BarWidth = 10

// Severity classifies context usage into Hermes-compatible display buckets.
type Severity int

const (
	SeverityDim Severity = iota
	SeverityGood
	SeverityWarn
	SeverityBad
	SeverityCritical
)

// ClampPercent normalizes context percentages before rendering bars or labels
// so every status-bar surface reports the same bounded value.
func ClampPercent(pct float64) float64 {
	if math.IsNaN(pct) || pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// PercentFromTokens converts token usage to a rounded, bounded percentage.
func PercentFromTokens(tokens, length int) *int {
	if length <= 0 {
		return nil
	}
	pct := int((float64(tokens) / float64(length) * 100) + 0.5)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return &pct
}

// FilledCells returns the rounded number of filled cells for BarWidth.
func FilledCells(pct float64) int {
	pct = ClampPercent(pct)
	filled := int((pct/100)*float64(BarWidth) + 0.5)
	if filled < 0 {
		return 0
	}
	if filled > BarWidth {
		return BarWidth
	}
	return filled
}

// SeverityForPercent mirrors Hermes' _status_bar_context_style: nil → dim,
// <50 good, 50–80 warn, 81–94 bad, ≥95 critical.
func SeverityForPercent(percent *int) Severity {
	if percent == nil {
		return SeverityDim
	}
	return SeverityForFloat(float64(*percent))
}

// SeverityForFloat returns severity for already-computed percentage values.
func SeverityForFloat(pct float64) Severity {
	pct = ClampPercent(pct)
	switch {
	case pct >= 95:
		return SeverityCritical
	case pct > 80:
		return SeverityBad
	case pct >= 50:
		return SeverityWarn
	default:
		return SeverityGood
	}
}
