package toon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const defaultIndent = 2

var unquotedKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)

// Encode converts any JSON-serializable Go value to TOON through the standard
// JSON normalization path. Prefer EncodeJSON when the caller already owns JSON
// bytes and needs exact JSON data-model semantics.
func Encode(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return EncodeJSON(raw)
}

// EncodeJSON parses JSON with ordered object keys and emits TOON. It preserves
// the JSON data model and does not mutate public API/provider payloads.
func EncodeJSON(raw []byte) ([]byte, error) {
	value, err := DecodeJSONValue(raw)
	if err != nil {
		return nil, err
	}
	return EncodeValue(value)
}

func EncodeValue(value Value) ([]byte, error) {
	var enc encoder
	var buf bytes.Buffer
	if err := enc.writeRoot(&buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type encoder struct {
	indentSize int
}

func (e *encoder) indent() int {
	if e.indentSize <= 0 {
		return defaultIndent
	}
	return e.indentSize
}

func (e *encoder) writeRoot(buf *bytes.Buffer, value Value) error {
	switch value.Kind {
	case KindObject:
		if len(value.Object) == 0 {
			buf.WriteString("{}")
			return nil
		}
		return e.writeObjectFields(buf, value.Object, 0)
	case KindArray:
		if len(value.Array) == 0 {
			buf.WriteString("[]")
			return nil
		}
		return e.writeArray(buf, "", value.Array, 0, "", false)
	default:
		buf.WriteString(e.primitive(value, ','))
		return nil
	}
}

func (e *encoder) writeObjectFields(buf *bytes.Buffer, members []Member, depth int) error {
	for i, member := range members {
		if i > 0 {
			buf.WriteByte('\n')
		}
		if err := e.writeField(buf, member.Key, member.Value, depth, "", false); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) writeField(buf *bytes.Buffer, key string, value Value, depth int, linePrefix string, listItemArray bool) error {
	e.writeIndent(buf, depth)
	buf.WriteString(linePrefix)
	encodedKey := encodeKey(key)
	switch value.Kind {
	case KindObject:
		buf.WriteString(encodedKey)
		buf.WriteByte(':')
		if len(value.Object) == 0 {
			return nil
		}
		buf.WriteByte('\n')
		return e.writeObjectFields(buf, value.Object, depth+1)
	case KindArray:
		if len(value.Array) == 0 {
			buf.WriteString(encodedKey)
			buf.WriteString(": []")
			return nil
		}
		return e.writeArray(buf, encodedKey, value.Array, depth, linePrefix, listItemArray)
	default:
		buf.WriteString(encodedKey)
		buf.WriteString(": ")
		buf.WriteString(e.primitive(value, ','))
		return nil
	}
}

func (e *encoder) writeArray(buf *bytes.Buffer, key string, values []Value, depth int, linePrefix string, listItemArray bool) error {
	headerDepth := depth
	if allPrimitive(values) {
		e.writeArrayHeader(buf, key, len(values), nil)
		buf.WriteByte(' ')
		for i, value := range values {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(e.primitive(value, ','))
		}
		return nil
	}
	if fields, ok := tabularFields(values); ok {
		e.writeArrayHeader(buf, key, len(values), fields)
		rowDepth := headerDepth + 1
		if linePrefix != "" && listItemArray {
			rowDepth = headerDepth + 2
		}
		for _, row := range values {
			buf.WriteByte('\n')
			e.writeIndent(buf, rowDepth)
			for i, field := range fields {
				if i > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(e.primitive(memberValue(row.Object, field), ','))
			}
		}
		return nil
	}

	e.writeArrayHeader(buf, key, len(values), nil)
	for _, value := range values {
		buf.WriteByte('\n')
		if err := e.writeListItem(buf, value, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) writeArrayHeader(buf *bytes.Buffer, key string, length int, fields []string) {
	if key != "" {
		buf.WriteString(key)
	}
	buf.WriteByte('[')
	buf.WriteString(fmt.Sprint(length))
	buf.WriteByte(']')
	if len(fields) > 0 {
		buf.WriteByte('{')
		for i, field := range fields {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(encodeKey(field))
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(':')
}

func (e *encoder) writeListItem(buf *bytes.Buffer, value Value, depth int) error {
	switch value.Kind {
	case KindObject:
		if len(value.Object) == 0 {
			e.writeIndent(buf, depth)
			buf.WriteByte('-')
			return nil
		}
		first := value.Object[0]
		if err := e.writeField(buf, first.Key, first.Value, depth, "- ", true); err != nil {
			return err
		}
		for _, member := range value.Object[1:] {
			buf.WriteByte('\n')
			if err := e.writeField(buf, member.Key, member.Value, depth+1, "", false); err != nil {
				return err
			}
		}
	case KindArray:
		e.writeIndent(buf, depth)
		buf.WriteString("- ")
		if len(value.Array) == 0 {
			buf.WriteString("[]")
			return nil
		}
		return e.writeArray(buf, "", value.Array, depth, "- ", false)
	default:
		e.writeIndent(buf, depth)
		buf.WriteString("- ")
		buf.WriteString(e.primitive(value, ','))
	}
	return nil
}

func (e *encoder) primitive(value Value, delimiter rune) string {
	switch value.Kind {
	case KindNull:
		return "null"
	case KindBool:
		if value.Bool {
			return "true"
		}
		return "false"
	case KindNumber:
		n, err := canonicalNumber(value.Number)
		if err != nil {
			return rawJSONString(value.Number)
		}
		return n
	case KindString:
		return encodeString(value.String, delimiter)
	default:
		return rawJSONString("")
	}
}

func (e *encoder) writeIndent(buf *bytes.Buffer, depth int) {
	if depth > 0 {
		buf.WriteString(strings.Repeat(" ", depth*e.indent()))
	}
}

func encodeKey(key string) string {
	if unquotedKeyPattern.MatchString(key) {
		return key
	}
	return quoteTOONString(key)
}

func encodeString(s string, delimiter rune) string {
	if mustQuoteString(s, delimiter) {
		return quoteTOONString(s)
	}
	return s
}

func quoteTOONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				b.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func mustQuoteString(s string, delimiter rune) bool {
	if s == "" || strings.TrimSpace(s) != s {
		return true
	}
	switch s {
	case "true", "false", "null", "-":
		return true
	}
	if strings.HasPrefix(s, "-") || numericLikePattern.MatchString(s) || forbiddenLeadingZeroPattern.MatchString(s) {
		return true
	}
	if strings.ContainsAny(s, ":\"\\[]{}") || strings.ContainsRune(s, delimiter) {
		return true
	}
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			return true
		}
		if unicode.IsControl(r) {
			return true
		}
		s = s[size:]
	}
	return false
}

func allPrimitive(values []Value) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if !isPrimitive(value) {
			return false
		}
	}
	return true
}

func isPrimitive(value Value) bool {
	return value.Kind == KindNull || value.Kind == KindBool || value.Kind == KindNumber || value.Kind == KindString
}

func tabularFields(values []Value) ([]string, bool) {
	if len(values) == 0 || values[0].Kind != KindObject || len(values[0].Object) == 0 {
		return nil, false
	}
	fields := make([]string, 0, len(values[0].Object))
	for _, member := range values[0].Object {
		if !isPrimitive(member.Value) {
			return nil, false
		}
		fields = append(fields, member.Key)
	}
	for _, value := range values[1:] {
		if value.Kind != KindObject || len(value.Object) != len(fields) {
			return nil, false
		}
		seen := make(map[string]bool, len(fields))
		for _, member := range value.Object {
			if !isPrimitive(member.Value) {
				return nil, false
			}
			seen[member.Key] = true
		}
		for _, field := range fields {
			if !seen[field] {
				return nil, false
			}
		}
	}
	return fields, true
}

func memberValue(members []Member, key string) Value {
	for _, member := range members {
		if member.Key == key {
			return member.Value
		}
	}
	return Value{Kind: KindNull}
}
