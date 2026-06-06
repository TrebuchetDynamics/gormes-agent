package command

import (
	"fmt"
	"strings"
)

type SlashResult struct {
	Text    string
	Status  string
	Enqueue bool
}

func HandleSlash(input string, currentLen int) SlashResult {
	text := strings.TrimSpace(invocationArgs(input))
	if text == "" {
		return SlashResult{Status: fmt.Sprintf("%d queued message(s)", currentLen)}
	}
	return SlashResult{Text: text, Enqueue: true}
}

func invocationArgs(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(input, fields[0]))
}
