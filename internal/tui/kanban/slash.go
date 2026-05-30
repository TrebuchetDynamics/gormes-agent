package kanban

import (
	"strings"
	"unicode/utf8"
)

const MaxStatusRunes = 600

type Runner func(input string) (string, error)

type Result struct {
	StatusMessage string
}

func HandleSlash(input string, run Runner) Result {
	if run == nil {
		return Result{StatusMessage: "kanban: command runner unavailable"}
	}
	output, err := run(input)
	status := strings.TrimSpace(output)
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			msg = "command failed"
		}
		if status != "" {
			msg = msg + ": " + status
		}
		return Result{StatusMessage: BoundStatus("kanban: " + msg)}
	}
	if status == "" {
		status = "kanban: no output"
	}
	return Result{StatusMessage: BoundStatus(status)}
}

func BoundStatus(status string) string {
	status = strings.Join(strings.Fields(strings.TrimSpace(status)), " ")
	if utf8.RuneCountInString(status) <= MaxStatusRunes {
		return status
	}
	runes := []rune(status)
	return string(runes[:MaxStatusRunes]) + "..."
}
