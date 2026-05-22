package toon

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func Decode(raw []byte) (Value, error) {
	return DecodeValue(raw)
}

func DecodeValue(raw []byte) (Value, error) {
	lines, err := parseLines(string(raw))
	if err != nil {
		return Value{}, err
	}
	if len(lines) == 0 {
		return Value{Kind: KindObject}, nil
	}
	if len(lines) == 1 && strings.TrimSpace(lines[0].text) == "[]" {
		return Value{Kind: KindArray}, nil
	}
	if len(lines) == 1 && strings.TrimSpace(lines[0].text) == "{}" {
		return Value{Kind: KindObject}, nil
	}
	if header, ok, err := parseHeader(lines[0].text); err != nil {
		return Value{}, err
	} else if ok && header.key == nil {
		value, next, err := parseArray(lines, 0, header, 1)
		if err != nil {
			return Value{}, err
		}
		if next != len(lines) {
			return Value{}, fmt.Errorf("toon: unexpected trailing line %d", lines[next].number)
		}
		return value, nil
	}
	if len(lines) == 1 && firstUnquotedColon(lines[0].text) < 0 {
		return parsePrimitive(lines[0].text)
	}
	members, next, err := parseObject(lines, 0, 0)
	if err != nil {
		return Value{}, err
	}
	if next != len(lines) {
		return Value{}, fmt.Errorf("toon: unexpected trailing line %d", lines[next].number)
	}
	return Value{Kind: KindObject, Object: members}, nil
}

func DecodeJSON(raw []byte) ([]byte, error) {
	value, err := DecodeValue(raw)
	if err != nil {
		return nil, err
	}
	return value.MarshalJSON()
}

type parsedLine struct {
	number int
	depth  int
	text   string
}

func parseLines(raw string) ([]parsedLine, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimRight(raw, "\n")
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, "\n")
	lines := make([]parsedLine, 0, len(parts))
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		spaces := 0
		for spaces < len(part) {
			switch part[spaces] {
			case ' ':
				spaces++
			case '\t':
				return nil, fmt.Errorf("toon: tab indentation on line %d", i+1)
			default:
				goto done
			}
		}
	done:
		if spaces%defaultIndent != 0 {
			return nil, fmt.Errorf("toon: invalid indentation on line %d", i+1)
		}
		lines = append(lines, parsedLine{number: i + 1, depth: spaces / defaultIndent, text: part[spaces:]})
	}
	return lines, nil
}

func parseObject(lines []parsedLine, start, depth int) ([]Member, int, error) {
	var members []Member
	i := start
	for i < len(lines) {
		line := lines[i]
		if line.depth < depth {
			break
		}
		if line.depth > depth {
			return nil, i, fmt.Errorf("toon: unexpected indentation on line %d", line.number)
		}
		member, next, err := parseField(lines, i, depth)
		if err != nil {
			return nil, i, err
		}
		members = append(members, member)
		i = next
	}
	return members, i, nil
}

func parseField(lines []parsedLine, index, depth int) (Member, int, error) {
	line := lines[index]
	if header, ok, err := parseHeader(line.text); err != nil {
		return Member{}, index, err
	} else if ok && header.key != nil {
		value, next, err := parseArray(lines, index, header, depth+1)
		if err != nil {
			return Member{}, index, err
		}
		return Member{Key: *header.key, Value: value}, next, nil
	}
	colon := firstUnquotedColon(line.text)
	if colon < 0 {
		return Member{}, index, fmt.Errorf("toon: missing colon on line %d", line.number)
	}
	key, err := parseKey(strings.TrimSpace(line.text[:colon]))
	if err != nil {
		return Member{}, index, fmt.Errorf("toon: line %d: %w", line.number, err)
	}
	rawValue := strings.TrimSpace(line.text[colon+1:])
	if rawValue == "" {
		if index+1 < len(lines) && lines[index+1].depth > depth {
			members, next, err := parseObject(lines, index+1, depth+1)
			if err != nil {
				return Member{}, index, err
			}
			return Member{Key: key, Value: Value{Kind: KindObject, Object: members}}, next, nil
		}
		return Member{Key: key, Value: Value{Kind: KindObject}}, index + 1, nil
	}
	if rawValue == "[]" {
		return Member{Key: key, Value: Value{Kind: KindArray}}, index + 1, nil
	}
	if rawValue == "{}" {
		return Member{Key: key, Value: Value{Kind: KindObject}}, index + 1, nil
	}
	value, err := parsePrimitive(rawValue)
	if err != nil {
		return Member{}, index, fmt.Errorf("toon: line %d: %w", line.number, err)
	}
	return Member{Key: key, Value: value}, index + 1, nil
}

type header struct {
	key       *string
	length    int
	delimiter rune
	fields    []string
	inline    string
}

func parseHeader(text string) (header, bool, error) {
	bracket := firstUnquotedRune(text, '[')
	if bracket < 0 {
		return header{}, false, nil
	}
	if colon := firstUnquotedColon(text); colon >= 0 && colon < bracket {
		return header{}, false, nil
	}
	if bracket > 0 && strings.ContainsAny(strings.TrimSpace(text[:bracket]), " \t") && !strings.HasPrefix(strings.TrimSpace(text[:bracket]), `"`) {
		return header{}, false, nil
	}
	var key *string
	if bracket > 0 {
		parsedKey, err := parseKey(strings.TrimSpace(text[:bracket]))
		if err != nil {
			return header{}, false, err
		}
		key = &parsedKey
	}
	closeBracket := strings.IndexByte(text[bracket:], ']')
	if closeBracket < 0 {
		return header{}, false, nil
	}
	closeBracket += bracket
	inside := text[bracket+1 : closeBracket]
	delimiter := ','
	if strings.HasSuffix(inside, "|") {
		delimiter = '|'
		inside = strings.TrimSuffix(inside, "|")
	} else if strings.HasSuffix(inside, "\t") {
		delimiter = '\t'
		inside = strings.TrimSuffix(inside, "\t")
	}
	if inside == "" {
		return header{}, false, nil
	}
	length, err := strconv.Atoi(inside)
	if err != nil || length < 0 {
		return header{}, false, nil
	}
	pos := closeBracket + 1
	for pos < len(text) && text[pos] == ' ' {
		pos++
	}
	var fields []string
	if pos < len(text) && text[pos] == '{' {
		end := findUnquotedClosing(text, pos, '}')
		if end < 0 {
			return header{}, false, fmt.Errorf("toon: unterminated fields segment")
		}
		fieldTokens, err := splitDelimited(text[pos+1:end], delimiter)
		if err != nil {
			return header{}, false, err
		}
		for _, token := range fieldTokens {
			field, err := parseKey(token)
			if err != nil {
				return header{}, false, err
			}
			fields = append(fields, field)
		}
		pos = end + 1
		for pos < len(text) && text[pos] == ' ' {
			pos++
		}
	}
	if pos >= len(text) || text[pos] != ':' {
		return header{}, false, nil
	}
	inline := ""
	if pos+1 < len(text) {
		inline = strings.TrimSpace(text[pos+1:])
	}
	return header{key: key, length: length, delimiter: delimiter, fields: fields, inline: inline}, true, nil
}

func parseArray(lines []parsedLine, index int, h header, childDepth int) (Value, int, error) {
	if h.length == 0 && h.inline == "" {
		return Value{Kind: KindArray}, index + 1, nil
	}
	if len(h.fields) > 0 {
		rows := make([]Value, 0, h.length)
		i := index + 1
		for len(rows) < h.length && i < len(lines) && lines[i].depth == childDepth {
			tokens, err := splitDelimited(lines[i].text, h.delimiter)
			if err != nil {
				return Value{}, index, err
			}
			if len(tokens) != len(h.fields) {
				return Value{}, index, fmt.Errorf("toon: line %d has %d tabular fields, want %d", lines[i].number, len(tokens), len(h.fields))
			}
			members := make([]Member, 0, len(h.fields))
			for fieldIndex, field := range h.fields {
				value, err := parsePrimitive(tokens[fieldIndex])
				if err != nil {
					return Value{}, index, fmt.Errorf("toon: line %d: %w", lines[i].number, err)
				}
				members = append(members, Member{Key: field, Value: value})
			}
			rows = append(rows, Value{Kind: KindObject, Object: members})
			i++
		}
		if len(rows) != h.length {
			return Value{}, index, fmt.Errorf("toon: expected %d tabular rows, got %d", h.length, len(rows))
		}
		return Value{Kind: KindArray, Array: rows}, i, nil
	}
	if h.inline != "" {
		tokens, err := splitDelimited(h.inline, h.delimiter)
		if err != nil {
			return Value{}, index, err
		}
		if len(tokens) != h.length {
			return Value{}, index, fmt.Errorf("toon: expected %d inline values, got %d", h.length, len(tokens))
		}
		values := make([]Value, 0, len(tokens))
		for _, token := range tokens {
			value, err := parsePrimitive(token)
			if err != nil {
				return Value{}, index, err
			}
			values = append(values, value)
		}
		return Value{Kind: KindArray, Array: values}, index + 1, nil
	}

	values := make([]Value, 0, h.length)
	i := index + 1
	for len(values) < h.length && i < len(lines) && lines[i].depth == childDepth {
		value, next, err := parseListItem(lines, i, childDepth)
		if err != nil {
			return Value{}, index, err
		}
		values = append(values, value)
		i = next
	}
	if len(values) != h.length {
		return Value{}, index, fmt.Errorf("toon: expected %d list array items, got %d", h.length, len(values))
	}
	return Value{Kind: KindArray, Array: values}, i, nil
}

func parseListItem(lines []parsedLine, index, depth int) (Value, int, error) {
	line := lines[index]
	if line.text == "-" {
		return Value{Kind: KindObject}, index + 1, nil
	}
	if !strings.HasPrefix(line.text, "- ") {
		return Value{}, index, fmt.Errorf("toon: expected list item on line %d", line.number)
	}
	body := strings.TrimSpace(strings.TrimPrefix(line.text, "- "))
	if body == "[]" {
		return Value{Kind: KindArray}, index + 1, nil
	}
	if body == "{}" {
		return Value{Kind: KindObject}, index + 1, nil
	}
	if h, ok, err := parseHeader(body); err != nil {
		return Value{}, index, err
	} else if ok {
		if h.key == nil {
			value, next, err := parseArray(lines, index, h, depth+1)
			if err != nil {
				return Value{}, index, err
			}
			return value, next, nil
		}
		value, next, err := parseArray(lines, index, h, depth+2)
		if err != nil {
			return Value{}, index, err
		}
		members := []Member{{Key: *h.key, Value: value}}
		if next < len(lines) && lines[next].depth == depth+1 {
			rest, after, err := parseObject(lines, next, depth+1)
			if err != nil {
				return Value{}, index, err
			}
			members = append(members, rest...)
			next = after
		}
		return Value{Kind: KindObject, Object: members}, next, nil
	}
	if firstUnquotedColon(body) >= 0 {
		synthetic := parsedLine{number: line.number, depth: depth, text: body}
		copyLines := append([]parsedLine(nil), lines...)
		copyLines[index] = synthetic
		first, next, err := parseField(copyLines, index, depth)
		if err != nil {
			return Value{}, index, err
		}
		members := []Member{first}
		if next < len(lines) && lines[next].depth == depth+1 {
			rest, after, err := parseObject(lines, next, depth+1)
			if err != nil {
				return Value{}, index, err
			}
			members = append(members, rest...)
			next = after
		}
		return Value{Kind: KindObject, Object: members}, next, nil
	}
	value, err := parsePrimitive(body)
	if err != nil {
		return Value{}, index, fmt.Errorf("toon: line %d: %w", line.number, err)
	}
	return value, index + 1, nil
}

func parsePrimitive(token string) (Value, error) {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(token, `"`) {
		s, err := unquoteTOONString(token)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KindString, String: s}, nil
	}
	switch token {
	case "true":
		return Value{Kind: KindBool, Bool: true}, nil
	case "false":
		return Value{Kind: KindBool}, nil
	case "null":
		return Value{Kind: KindNull}, nil
	}
	if isNumericToken(token) {
		n, err := canonicalNumber(token)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KindNumber, Number: n}, nil
	}
	return Value{Kind: KindString, String: token}, nil
}

func parseKey(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("empty key")
	}
	if strings.HasPrefix(token, `"`) {
		return unquoteTOONString(token)
	}
	if !unquotedKeyPattern.MatchString(token) {
		return "", fmt.Errorf("invalid unquoted key %q", token)
	}
	return token, nil
}

func unquoteTOONString(token string) (string, error) {
	if len(token) < 2 || token[0] != '"' {
		return "", fmt.Errorf("expected quoted string")
	}
	var b strings.Builder
	for i := 1; i < len(token); {
		c := token[i]
		if c == '"' {
			if strings.TrimSpace(token[i+1:]) != "" {
				return "", fmt.Errorf("trailing characters after quoted string")
			}
			return b.String(), nil
		}
		if c == '\\' {
			if i+1 >= len(token) {
				return "", fmt.Errorf("unterminated escape")
			}
			switch token[i+1] {
			case '\\', '"', '/':
				b.WriteByte(token[i+1])
				i += 2
			case 'b':
				b.WriteByte('\b')
				i += 2
			case 'f':
				b.WriteByte('\f')
				i += 2
			case 'n':
				b.WriteByte('\n')
				i += 2
			case 'r':
				b.WriteByte('\r')
				i += 2
			case 't':
				b.WriteByte('\t')
				i += 2
			case 'u':
				if i+6 > len(token) {
					return "", fmt.Errorf("short unicode escape")
				}
				code, err := strconv.ParseUint(token[i+2:i+6], 16, 16)
				if err != nil {
					return "", fmt.Errorf("invalid unicode escape")
				}
				r := rune(code)
				if utf16.IsSurrogate(r) {
					if !isHighSurrogate(r) || i+12 > len(token) || token[i+6] != '\\' || token[i+7] != 'u' {
						return "", fmt.Errorf("invalid unicode surrogate pair")
					}
					lowCode, err := strconv.ParseUint(token[i+8:i+12], 16, 16)
					if err != nil {
						return "", fmt.Errorf("invalid unicode escape")
					}
					low := rune(lowCode)
					if !isLowSurrogate(low) {
						return "", fmt.Errorf("invalid unicode surrogate pair")
					}
					b.WriteRune(utf16.DecodeRune(r, low))
					i += 12
					continue
				}
				b.WriteRune(r)
				i += 6
			default:
				return "", fmt.Errorf("invalid escape sequence")
			}
			continue
		}
		r, size := utf8.DecodeRuneInString(token[i:])
		if r == utf8.RuneError && size == 1 {
			return "", fmt.Errorf("invalid UTF-8")
		}
		b.WriteRune(r)
		i += size
	}
	return "", fmt.Errorf("unterminated string")
}

func isHighSurrogate(r rune) bool {
	return r >= 0xD800 && r <= 0xDBFF
}

func isLowSurrogate(r rune) bool {
	return r >= 0xDC00 && r <= 0xDFFF
}

func firstUnquotedColon(s string) int {
	return firstUnquotedRune(s, ':')
}

func firstUnquotedRune(s string, target rune) int {
	inQuotes := false
	escaping := false
	for i, r := range s {
		if escaping {
			escaping = false
			continue
		}
		if inQuotes && r == '\\' {
			escaping = true
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes && r == target {
			return i
		}
	}
	return -1
}

func findUnquotedClosing(s string, start int, target rune) int {
	inQuotes := false
	escaping := false
	for i, r := range s[start+1:] {
		absolute := start + 1 + i
		if escaping {
			escaping = false
			continue
		}
		if inQuotes && r == '\\' {
			escaping = true
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes && r == target {
			return absolute
		}
	}
	return -1
}

func splitDelimited(s string, delimiter rune) ([]string, error) {
	var tokens []string
	start := 0
	inQuotes := false
	escaping := false
	for i, r := range s {
		if escaping {
			escaping = false
			continue
		}
		if inQuotes && r == '\\' {
			escaping = true
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes && r == delimiter {
			tokens = append(tokens, strings.TrimSpace(s[start:i]))
			start = i + len(string(r))
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("toon: unterminated quoted token")
	}
	tokens = append(tokens, strings.TrimSpace(s[start:]))
	return tokens, nil
}
