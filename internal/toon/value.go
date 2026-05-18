package toon

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type Kind int

const (
	KindNull Kind = iota
	KindBool
	KindNumber
	KindString
	KindObject
	KindArray
)

type Member struct {
	Key   string
	Value Value
}

// Value is Gormes' ordered JSON data model for TOON conversion. Object members
// preserve encounter order so JSON -> TOON prompt payloads stay readable and
// deterministic.
type Value struct {
	Kind   Kind
	Bool   bool
	Number string
	String string
	Object []Member
	Array  []Value
}

func DecodeJSONValue(raw []byte) (Value, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	value, err := decodeJSONValue(dec)
	if err != nil {
		return Value{}, err
	}
	if dec.More() {
		return Value{}, errors.New("toon: trailing JSON data")
	}
	if tok, err := dec.Token(); err != io.EOF {
		if err != nil {
			return Value{}, err
		}
		return Value{}, fmt.Errorf("toon: trailing JSON token %v", tok)
	}
	return value, nil
}

func decodeJSONValue(dec *json.Decoder) (Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return Value{}, err
	}
	switch v := tok.(type) {
	case nil:
		return Value{Kind: KindNull}, nil
	case bool:
		return Value{Kind: KindBool, Bool: v}, nil
	case string:
		return Value{Kind: KindString, String: v}, nil
	case json.Number:
		n, err := canonicalNumber(v.String())
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KindNumber, Number: n}, nil
	case json.Delim:
		switch v {
		case '{':
			var members []Member
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return Value{}, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return Value{}, fmt.Errorf("toon: expected JSON object key, got %T", keyTok)
				}
				child, err := decodeJSONValue(dec)
				if err != nil {
					return Value{}, err
				}
				members = append(members, Member{Key: key, Value: child})
			}
			if end, err := dec.Token(); err != nil || end != json.Delim('}') {
				if err != nil {
					return Value{}, err
				}
				return Value{}, fmt.Errorf("toon: expected JSON object end, got %v", end)
			}
			return Value{Kind: KindObject, Object: members}, nil
		case '[':
			var values []Value
			for dec.More() {
				child, err := decodeJSONValue(dec)
				if err != nil {
					return Value{}, err
				}
				values = append(values, child)
			}
			if end, err := dec.Token(); err != nil || end != json.Delim(']') {
				if err != nil {
					return Value{}, err
				}
				return Value{}, fmt.Errorf("toon: expected JSON array end, got %v", end)
			}
			return Value{Kind: KindArray, Array: values}, nil
		default:
			return Value{}, fmt.Errorf("toon: unexpected JSON delimiter %q", v)
		}
	default:
		return Value{}, fmt.Errorf("toon: unsupported JSON token %T", tok)
	}
}

func (v Value) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := writeJSONValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeJSONValue(w *bytes.Buffer, v Value) error {
	switch v.Kind {
	case KindNull:
		w.WriteString("null")
	case KindBool:
		if v.Bool {
			w.WriteString("true")
		} else {
			w.WriteString("false")
		}
	case KindNumber:
		n, err := canonicalNumber(v.Number)
		if err != nil {
			return err
		}
		w.WriteString(n)
	case KindString:
		raw, err := json.Marshal(v.String)
		if err != nil {
			return err
		}
		w.Write(raw)
	case KindObject:
		w.WriteByte('{')
		for i, member := range v.Object {
			if i > 0 {
				w.WriteByte(',')
			}
			rawKey, err := json.Marshal(member.Key)
			if err != nil {
				return err
			}
			w.Write(rawKey)
			w.WriteByte(':')
			if err := writeJSONValue(w, member.Value); err != nil {
				return err
			}
		}
		w.WriteByte('}')
	case KindArray:
		w.WriteByte('[')
		for i, child := range v.Array {
			if i > 0 {
				w.WriteByte(',')
			}
			if err := writeJSONValue(w, child); err != nil {
				return err
			}
		}
		w.WriteByte(']')
	default:
		return fmt.Errorf("toon: unknown value kind %d", v.Kind)
	}
	return nil
}

func rawJSONString(s string) string {
	raw, err := json.Marshal(s)
	if err != nil {
		return strconv.Quote(s)
	}
	return string(raw)
}
