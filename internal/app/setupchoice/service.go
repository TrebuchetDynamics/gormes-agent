package setupchoice

import (
	"strconv"
	"strings"
)

// Choice is one normalized setup prompt option.
type Choice struct {
	ID      string
	Label   string
	Aliases []string
}

// NormalizeAnswer resolves prompt input to an option ID, preserving the old
// setup behavior for blank defaults, numeric selections, labels, and aliases.
func NormalizeAnswer(answer string, options []Choice, defaultID string) string {
	answer = strings.TrimSpace(StripInputNoise(answer))
	if answer == "" {
		return strings.TrimSpace(defaultID)
	}
	if idx, err := strconv.Atoi(answer); err == nil && idx >= 1 && idx <= len(options) {
		return options[idx-1].ID
	}
	normalized := NormalizeValue(answer)
	for _, option := range options {
		if normalized == NormalizeValue(option.ID) || normalized == NormalizeValue(option.Label) {
			return option.ID
		}
		for _, alias := range option.Aliases {
			if normalized == NormalizeValue(alias) {
				return option.ID
			}
		}
	}
	return normalized
}

// YesNo parses setup yes/no answers and reports whether the input was valid.
func YesNo(value string, defaultValue bool) (bool, bool) {
	value = NormalizeValue(value)
	if value == "" {
		return defaultValue, true
	}
	switch value {
	case "y", "yes", "true", "1", "on":
		return true, true
	case "n", "no", "false", "0", "off":
		return false, true
	default:
		return false, false
	}
}

// NormalizeValue canonicalizes free-form setup choice text.
func NormalizeValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	if value == "apptainer" {
		return "singularity"
	}
	return value
}

// StripInputNoise removes ANSI escape/control bytes from line-mode prompt input.
func StripInputNoise(answer string) string {
	var b strings.Builder
	for i := 0; i < len(answer); {
		ch := answer[i]
		if ch == 0x1b {
			i++
			if i < len(answer) && answer[i] == '[' {
				i++
				for i < len(answer) {
					final := answer[i]
					i++
					if final >= 0x40 && final <= 0x7e {
						break
					}
				}
				continue
			}
			if i < len(answer) {
				i++
			}
			continue
		}
		if ch < 0x20 || ch == 0x7f {
			i++
			continue
		}
		b.WriteByte(ch)
		i++
	}
	return b.String()
}
