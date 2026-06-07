package command

import "strings"

type Request struct {
	Name      string
	SessionID string
}

type Result struct {
	Name string
}

type ConfigFunc func(Request) (Result, error)

type SlashResult struct {
	Name          string
	AcceptedName  string
	Apply         bool
	Body          string
	StatusMessage string
	Err           error
}

func HandleSlash(input string, sessionID string, configure ConfigFunc) SlashResult {
	name := SlashName(input)
	if configure == nil {
		return SlashResult{Name: name, StatusMessage: "skin: configuration unavailable"}
	}
	result, err := configure(Request{Name: name, SessionID: sessionID})
	if err != nil {
		return SlashResult{Name: name, Err: err, StatusMessage: "skin: " + err.Error()}
	}
	if name == "" {
		line := "skin: " + DisplayName(result.Name)
		return SlashResult{Name: name, Body: line, StatusMessage: line}
	}
	accepted := strings.TrimSpace(result.Name)
	if accepted == "" {
		accepted = name
	}
	return SlashResult{Name: name, AcceptedName: accepted, Apply: true}
}

func SlashName(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	_, rest, ok := strings.Cut(trimmed, " ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(rest)
}

func DisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}
