// Package yuanbao parses Tencent Yuanbao websocket envelopes and Markdown
// fragments into gateway-neutral events using fixture data only. The package
// intentionally avoids any third-party protobuf runtime: it implements just
// enough wire-format reading to lift inbound text envelopes from a captured
// ConnMsg payload.
package yuanbao

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// SourceName is the platform identifier emitted on every parsed envelope.
const SourceName = "yuanbao"

// AuthorRoleUser is the platform-neutral author role for end users. Yuanbao
// inbound pushes only carry user-authored text in this slice; bot/system
// echoes are not part of the protocol envelope contract.
const AuthorRoleUser = "user"

// Degraded codes mirror the row's degraded_mode contract.
const (
	DegradedProtocolUnavailable = "protocol_unavailable"
	DegradedMarkdownParseFailed = "markdown_parse_failed"
)

// DegradedError is the typed evidence returned when parsing cannot produce a
// gateway-neutral envelope. Callers route on Code; Detail is human-readable.
type DegradedError struct {
	Code   string
	Detail string
}

// Error implements the error interface.
func (e *DegradedError) Error() string {
	return e.Code + ": " + e.Detail
}

func newDegraded(code, detail string) *DegradedError {
	return &DegradedError{Code: code, Detail: detail}
}

// Envelope is the channel-neutral shape lifted from an inbound Yuanbao push.
type Envelope struct {
	Source         string
	ConversationID string
	MessageID      string
	AuthorRole     string
	Text           string
}

// DecodeInboundEnvelope reads a captured ConnMsg payload (Yuanbao websocket
// frame body) and returns the normalized Envelope. Malformed or unknown
// payloads return a *DegradedError without panicking.
func DecodeInboundEnvelope(raw []byte) (Envelope, error) {
	if len(raw) == 0 {
		return Envelope{}, newDegraded(DegradedProtocolUnavailable, "empty payload")
	}

	conn, err := parseConnMsg(raw)
	if err != nil {
		return Envelope{}, newDegraded(DegradedProtocolUnavailable, err.Error())
	}

	push, err := parseInboundPush(conn.data)
	if err != nil {
		return Envelope{}, newDegraded(DegradedProtocolUnavailable, err.Error())
	}

	convID := push.groupCode
	if convID == "" {
		convID = push.fromAccount
	}
	if convID == "" || push.msgID == "" {
		return Envelope{}, newDegraded(
			DegradedProtocolUnavailable,
			"missing conversation id or message id",
		)
	}

	text := strings.Join(push.texts, "")
	if text == "" {
		return Envelope{}, newDegraded(
			DegradedProtocolUnavailable,
			"no text body element",
		)
	}

	return Envelope{
		Source:         SourceName,
		ConversationID: convID,
		MessageID:      push.msgID,
		AuthorRole:     AuthorRoleUser,
		Text:           text,
	}, nil
}

// connMsg is the minimum ConnMsg shape the Yuanbao slice consumes.
type connMsg struct {
	data []byte
}

func parseConnMsg(raw []byte) (connMsg, error) {
	var out connMsg
	for pos := 0; pos < len(raw); {
		fn, wt, val, next, err := readField(raw, pos)
		if err != nil {
			return connMsg{}, err
		}
		pos = next
		// Field 2 of ConnMsg holds the inner business payload. Field 1 is the
		// Head; everything else is ignored for fixture decoding.
		if fn == 2 && wt == wireLen {
			out.data = val.bytes
		}
	}
	return out, nil
}

// inboundPush is the minimum InboundMessagePush shape the slice consumes.
type inboundPush struct {
	fromAccount string
	groupCode   string
	msgID       string
	texts       []string
}

func parseInboundPush(raw []byte) (inboundPush, error) {
	if len(raw) == 0 {
		return inboundPush{}, errors.New("empty inbound push payload")
	}
	var out inboundPush
	sawAny := false
	for pos := 0; pos < len(raw); {
		fn, wt, val, next, err := readField(raw, pos)
		if err != nil {
			return inboundPush{}, err
		}
		pos = next
		if fn >= 1 && fn <= 20 {
			sawAny = true
		}
		switch {
		case fn == 2 && wt == wireLen:
			out.fromAccount = val.utf8()
		case fn == 6 && wt == wireLen:
			out.groupCode = val.utf8()
		case fn == 12 && wt == wireLen:
			out.msgID = val.utf8()
		case fn == 13 && wt == wireLen:
			text, ok := extractTextElem(val.bytes)
			if ok {
				out.texts = append(out.texts, text)
			}
		}
	}
	if !sawAny {
		return inboundPush{}, errors.New("no recognized inbound push fields")
	}
	return out, nil
}

// extractTextElem reads a MsgBodyElement looking for TIMTextElem text.
func extractTextElem(raw []byte) (string, bool) {
	var msgType, text string
	var sawContent bool
	for pos := 0; pos < len(raw); {
		fn, wt, val, next, err := readField(raw, pos)
		if err != nil {
			return "", false
		}
		pos = next
		switch {
		case fn == 1 && wt == wireLen:
			msgType = val.utf8()
		case fn == 2 && wt == wireLen:
			sawContent = true
			text = extractMsgContentText(val.bytes)
		}
	}
	if msgType != "TIMTextElem" || !sawContent {
		return "", false
	}
	return text, true
}

// extractMsgContentText reads MsgContent.text (field 1).
func extractMsgContentText(raw []byte) string {
	for pos := 0; pos < len(raw); {
		fn, wt, val, next, err := readField(raw, pos)
		if err != nil {
			return ""
		}
		pos = next
		if fn == 1 && wt == wireLen {
			return val.utf8()
		}
	}
	return ""
}

// Wire-format primitives (no third-party protobuf runtime).

const (
	wireVarint = 0
	wire64bit  = 1
	wireLen    = 2
	wire32bit  = 5
)

type fieldValue struct {
	v     uint64
	bytes []byte
}

func (f fieldValue) utf8() string {
	if !utf8.Valid(f.bytes) {
		return ""
	}
	return string(f.bytes)
}

func readField(buf []byte, pos int) (fieldNumber int, wireType int, val fieldValue, next int, err error) {
	tag, n, err := readVarint(buf, pos)
	if err != nil {
		return 0, 0, fieldValue{}, 0, fmt.Errorf("read tag: %w", err)
	}
	pos = n
	wireType = int(tag & 0x07)
	fieldNumber = int(tag >> 3)
	switch wireType {
	case wireVarint:
		v, np, verr := readVarint(buf, pos)
		if verr != nil {
			return 0, 0, fieldValue{}, 0, fmt.Errorf("field %d varint: %w", fieldNumber, verr)
		}
		return fieldNumber, wireType, fieldValue{v: v}, np, nil
	case wire64bit:
		if pos+8 > len(buf) {
			return 0, 0, fieldValue{}, 0, fmt.Errorf("field %d 64bit truncated", fieldNumber)
		}
		v := binary.LittleEndian.Uint64(buf[pos : pos+8])
		return fieldNumber, wireType, fieldValue{v: v}, pos + 8, nil
	case wireLen:
		length, np, lerr := readVarint(buf, pos)
		if lerr != nil {
			return 0, 0, fieldValue{}, 0, fmt.Errorf("field %d length: %w", fieldNumber, lerr)
		}
		end := np + int(length)
		if length > uint64(len(buf)) || end > len(buf) || end < np {
			return 0, 0, fieldValue{}, 0, fmt.Errorf("field %d length-delimited overruns buffer", fieldNumber)
		}
		return fieldNumber, wireType, fieldValue{bytes: buf[np:end]}, end, nil
	case wire32bit:
		if pos+4 > len(buf) {
			return 0, 0, fieldValue{}, 0, fmt.Errorf("field %d 32bit truncated", fieldNumber)
		}
		v := uint64(binary.LittleEndian.Uint32(buf[pos : pos+4]))
		return fieldNumber, wireType, fieldValue{v: v}, pos + 4, nil
	default:
		return 0, 0, fieldValue{}, 0, fmt.Errorf("unsupported wire type %d", wireType)
	}
}

func readVarint(buf []byte, pos int) (uint64, int, error) {
	var v uint64
	var shift uint
	for i := 0; i < 10; i++ {
		if pos >= len(buf) {
			return 0, 0, errors.New("varint truncated")
		}
		b := buf[pos]
		pos++
		v |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return v, pos, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, errors.New("varint too long")
		}
	}
	return 0, 0, errors.New("varint too long")
}
