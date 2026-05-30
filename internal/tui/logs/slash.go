package logs

import (
	"strconv"
	"strings"
)

type TailFunc func(limit int) (string, error)

type SlashResult struct {
	Status string
	Title  string
	Body   string
	Open   bool
}

// HandleSlash resolves /logs status/page behavior with tail retrieval injected
// by the root TUI model.
func HandleSlash(input string, tail TailFunc) SlashResult {
	if tail == nil {
		return SlashResult{Status: "no gateway logs"}
	}
	text, err := tail(TailLimit(input))
	if err != nil {
		return SlashResult{Status: "logs: " + err.Error()}
	}
	text = strings.TrimRight(text, "\n")
	if strings.TrimSpace(text) == "" {
		return SlashResult{Status: "no gateway logs"}
	}
	return SlashResult{Status: "logs opened", Title: "Logs", Body: text, Open: true}
}

// TailLimit resolves /logs optional line-count input. It preserves the TUI
// command's historical default and clamps requests into the safe display range.
func TailLimit(input string) int {
	fields := strings.Fields(strings.TrimSpace(input))
	limit := 20
	if len(fields) > 1 {
		if n, err := strconv.Atoi(fields[1]); err == nil {
			switch {
			case n < 0:
				limit = 1
			case n > 0:
				limit = n
			}
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 80 {
		limit = 80
	}
	return limit
}
