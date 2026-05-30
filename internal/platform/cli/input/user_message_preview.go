package input

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

const (
	defaultUserMessagePreviewFirstLines = 2
	defaultUserMessagePreviewLastLines  = 2
)

type UserMessagePreviewConfig struct {
	FirstLines any
	LastLines  any
}

func FormatUserMessagePreview(userInput string, cfg UserMessagePreviewConfig) string {
	lines := strings.Split(userInput, "\n")
	if len(lines) <= 1 {
		return "● " + sanitizeUserMessagePreviewLine(userInput)
	}

	firstLines := normalizeUserMessagePreviewLines(cfg.FirstLines, defaultUserMessagePreviewFirstLines, 1)
	lastLines := normalizeUserMessagePreviewLines(cfg.LastLines, defaultUserMessagePreviewLastLines, 0)

	head := lines[:minInt(firstLines, len(lines))]
	remainingAfterHead := len(lines) - len(head)
	tailCount := minInt(lastLines, remainingAfterHead)
	var tail []string
	if tailCount > 0 {
		tail = lines[len(lines)-tailCount:]
	}

	hiddenMiddleCount := len(lines) - len(head) - len(tail)
	if hiddenMiddleCount < 0 {
		hiddenMiddleCount = 0
		tail = nil
	}

	previewLines := []string{"● " + sanitizeUserMessagePreviewLine(head[0])}
	for _, line := range head[1:] {
		previewLines = append(previewLines, sanitizeUserMessagePreviewLine(line))
	}
	if hiddenMiddleCount > 0 {
		noun := "lines"
		if hiddenMiddleCount == 1 {
			noun = "line"
		}
		previewLines = append(previewLines, fmt.Sprintf("... (+%d more %s)", hiddenMiddleCount, noun))
	}
	for _, line := range tail {
		previewLines = append(previewLines, sanitizeUserMessagePreviewLine(line))
	}
	return strings.Join(previewLines, "\n")
}

func normalizeUserMessagePreviewLines(raw any, fallback int, minimum int) int {
	value, ok := parseUserMessagePreviewInt(raw)
	if !ok {
		value = fallback
	}
	if value < minimum {
		return minimum
	}
	return value
}

func parseUserMessagePreviewInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case nil:
		return 0, false
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func sanitizeUserMessagePreviewLine(line string) string {
	line = redaction.StripANSI(line)
	line = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' {
			return -1
		}
		if r == 0x7f {
			return -1
		}
		return r
	}, line)
	line = strings.ReplaceAll(line, `[`, `\[`)
	line = strings.ReplaceAll(line, `]`, `\]`)
	return line
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
