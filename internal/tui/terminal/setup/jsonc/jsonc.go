package jsonc

import (
	"bytes"
	"encoding/json"
	"strings"
)

func StripJSONComments(input string) string {
	var out strings.Builder
	var quote jsonStringState
	lineComment := false
	blockComment := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		var next byte
		if i+1 < len(input) {
			next = input[i+1]
		}

		if lineComment {
			if ch == '\n' {
				lineComment = false
				out.WriteByte(ch)
			}
			continue
		}
		if blockComment {
			if ch == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote.InString() {
			out.WriteByte(ch)
			quote.Consume(ch)
			continue
		}

		if ch == '"' {
			quote.Consume(ch)
			out.WriteByte(ch)
			continue
		}
		if ch == '/' && next == '/' {
			lineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			blockComment = true
			i++
			continue
		}
		out.WriteByte(ch)
	}
	return removeTrailingJSONCommas(out.String())
}

func removeTrailingJSONCommas(input string) string {
	var out strings.Builder
	var quote jsonStringState
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if quote.InString() {
			out.WriteByte(ch)
			quote.Consume(ch)
			continue
		}
		if ch == '"' {
			quote.Consume(ch)
			out.WriteByte(ch)
			continue
		}
		if ch == ',' {
			j := i + 1
			for j < len(input) && (input[j] == ' ' || input[j] == '\t' || input[j] == '\r' || input[j] == '\n') {
				j++
			}
			if j < len(input) && (input[j] == ']' || input[j] == '}') {
				continue
			}
		}
		out.WriteByte(ch)
	}
	return out.String()
}

func ParseKeybindings(body []byte) ([]map[string]any, error) {
	body = bytes.TrimSpace([]byte(StripJSONComments(string(body))))
	if len(body) == 0 {
		body = []byte("[]")
	}
	var bindings []map[string]any
	if err := json.Unmarshal(body, &bindings); err != nil {
		return nil, err
	}
	return bindings, nil
}
