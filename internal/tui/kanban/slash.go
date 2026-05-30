package kanban

import (
	"strings"
	"unicode/utf8"
)

const MaxStatusRunes = 600

func BoundStatus(status string) string {
	status = strings.Join(strings.Fields(strings.TrimSpace(status)), " ")
	if utf8.RuneCountInString(status) <= MaxStatusRunes {
		return status
	}
	runes := []rune(status)
	return string(runes[:MaxStatusRunes]) + "..."
}
