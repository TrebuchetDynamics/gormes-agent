package tuigateway

import (
	"fmt"
	"sort"
	"strings"
)

const (
	TUIConfigHealthWarningCode = "tui_config_health_warning"
	TUIPersonalitySkippedCode  = "tui_personality_skipped"
)

type ConfigHealthWarning struct {
	Code    string
	Paths   []string
	Message string
}

type ConfigHealthReport struct {
	Warnings []ConfigHealthWarning
}

func (r ConfigHealthReport) HasWarnings() bool {
	return len(r.Warnings) > 0
}

func (r ConfigHealthReport) HasCode(code string) bool {
	for _, warning := range r.Warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

func (r ConfigHealthReport) Message() string {
	parts := make([]string, 0, len(r.Warnings))
	for _, warning := range r.Warnings {
		if warning.Message != "" {
			parts = append(parts, warning.Message)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

type SessionOptions struct {
	HasEphemeralSystemPrompt bool
	EphemeralSystemPrompt    string
	DisplayPersonality       string
	PersonalitySkipped       bool
	Evidence                 []string
}

func ProbeConfigHealth(cfg map[string]any) ConfigHealthReport {
	if cfg == nil {
		return ConfigHealthReport{}
	}

	var report ConfigHealthReport
	nullKeys := make([]string, 0)
	for key, value := range cfg {
		if value == nil {
			nullKeys = append(nullKeys, key)
		}
	}
	if len(nullKeys) > 0 {
		sort.Strings(nullKeys)
		quoted := make([]string, len(nullKeys))
		for i, key := range nullKeys {
			quoted[i] = "`" + key + "`"
		}
		report.Warnings = append(report.Warnings, ConfigHealthWarning{
			Code:  TUIConfigHealthWarningCode,
			Paths: append([]string(nil), nullKeys...),
			Message: fmt.Sprintf(
				"config.yaml has empty section(s): %s. Remove the line(s) or set them to `{}` - empty sections silently drop nested settings.",
				strings.Join(quoted, ", "),
			),
		})
	}

	agentCfg, agentOK := sectionMap(cfg, "agent")
	displayCfg, displayOK := sectionMap(cfg, "display")
	if displayOK && agentOK {
		personality := normalizedConfigString(displayCfg["personality"])
		if personalityIsActive(personality) && personalitiesEmpty(agentCfg["personalities"]) {
			report.Warnings = append(report.Warnings, ConfigHealthWarning{
				Code:  TUIPersonalitySkippedCode,
				Paths: []string{"display.personality", "agent.personalities"},
				Message: "`display.personality` is set but `agent.personalities` is empty/null; " +
					"personality overlay will be skipped.",
			})
		}
	}

	return report
}

func ResolveSessionOptions(cfg map[string]any) SessionOptions {
	agentCfg, _ := sectionMap(cfg, "agent")
	displayCfg, _ := sectionMap(cfg, "display")

	opts := SessionOptions{}
	systemPrompt := strings.TrimSpace(configString(agentCfg["system_prompt"]))
	if systemPrompt != "" {
		opts.HasEphemeralSystemPrompt = true
		opts.EphemeralSystemPrompt = systemPrompt
	}

	personality := normalizedConfigString(displayCfg["personality"])
	if personalityIsActive(personality) {
		opts.DisplayPersonality = personality
		if !opts.HasEphemeralSystemPrompt {
			opts.PersonalitySkipped = true
			opts.Evidence = append(opts.Evidence, TUIPersonalitySkippedCode)
		}
	}

	return opts
}

func sectionMap(cfg map[string]any, key string) (map[string]any, bool) {
	if cfg == nil {
		return map[string]any{}, false
	}
	value, ok := cfg[key]
	if !ok {
		return map[string]any{}, false
	}
	section, ok := value.(map[string]any)
	if !ok || section == nil {
		return map[string]any{}, false
	}
	return section, true
}

func personalityIsActive(value string) bool {
	if value == "" {
		return false
	}
	_, blank := personalityAliases[value]
	return !blank
}

func personalitiesEmpty(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case map[string]any:
		return len(typed) == 0
	case map[string]string:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	default:
		return false
	}
}

func normalizedConfigString(value any) string {
	return strings.ToLower(strings.TrimSpace(configString(value)))
}

func configString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}
