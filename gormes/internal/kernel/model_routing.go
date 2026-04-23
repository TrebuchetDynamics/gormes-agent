package kernel

import (
	"regexp"
	"strings"
)

type SmartModelRouting struct {
	Enabled        bool
	SimpleModel    string
	MaxSimpleChars int
	MaxSimpleWords int
}

var (
	complexTurnKeywords = map[string]struct{}{
		"debug":          {},
		"debugging":      {},
		"implement":      {},
		"implementation": {},
		"refactor":       {},
		"patch":          {},
		"traceback":      {},
		"stacktrace":     {},
		"exception":      {},
		"error":          {},
		"analyze":        {},
		"analysis":       {},
		"investigate":    {},
		"architecture":   {},
		"design":         {},
		"compare":        {},
		"benchmark":      {},
		"optimize":       {},
		"optimise":       {},
		"review":         {},
		"terminal":       {},
		"shell":          {},
		"tool":           {},
		"tools":          {},
		"test":           {},
		"tests":          {},
		"plan":           {},
		"planning":       {},
		"delegate":       {},
		"subagent":       {},
		"cron":           {},
		"docker":         {},
		"kubernetes":     {},
	}
	urlPattern = regexp.MustCompile(`https?://|www\.`)
)

func (r SmartModelRouting) withDefaults() SmartModelRouting {
	if r.MaxSimpleChars <= 0 {
		r.MaxSimpleChars = 160
	}
	if r.MaxSimpleWords <= 0 {
		r.MaxSimpleWords = 28
	}
	return r
}

func selectTurnModel(primaryModel, userMessage string, routing SmartModelRouting) string {
	primaryModel = strings.TrimSpace(primaryModel)
	routing = routing.withDefaults()
	if !routing.Enabled {
		return primaryModel
	}

	simpleModel := strings.TrimSpace(routing.SimpleModel)
	if simpleModel == "" {
		return primaryModel
	}

	text := strings.TrimSpace(userMessage)
	if text == "" {
		return primaryModel
	}
	if len(text) > routing.MaxSimpleChars {
		return primaryModel
	}
	if len(strings.Fields(text)) > routing.MaxSimpleWords {
		return primaryModel
	}
	if strings.Count(text, "\n") > 1 {
		return primaryModel
	}
	if strings.Contains(text, "`") || urlPattern.MatchString(text) {
		return primaryModel
	}

	for _, field := range strings.Fields(strings.ToLower(text)) {
		token := strings.Trim(field, ".,:;!?()[]{}\"'`")
		if _, ok := complexTurnKeywords[token]; ok {
			return primaryModel
		}
	}
	return simpleModel
}
