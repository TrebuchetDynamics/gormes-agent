package navivox

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCodecRoundTripBinaryFrame(t *testing.T) {
	codec := NewCodec()
	in := Frame{
		Header: Header{
			Type:        EventVoiceAudio,
			MessageID:   "msg-1",
			Timestamp:   time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
			ContentType: "audio/ogg",
			Metadata: map[string]any{
				"codec": "opus",
				"chunk": float64(2),
			},
		},
		Payload: []byte{0x01, 0x02, 0x00, 0xff},
	}

	var buf bytes.Buffer
	if err := codec.WriteFrame(&buf, in); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	got, err := codec.ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}

	if got.Header.Type != EventVoiceAudio || got.Header.MessageID != "msg-1" || got.Header.ContentType != "audio/ogg" {
		t.Fatalf("header = %+v, want voice.audio msg-1 audio/ogg", got.Header)
	}
	if got.Header.PayloadLength != uint32(len(in.Payload)) {
		t.Fatalf("payload_length = %d, want %d", got.Header.PayloadLength, len(in.Payload))
	}
	if !bytes.Equal(got.Payload, in.Payload) {
		t.Fatalf("payload = %v, want %v", got.Payload, in.Payload)
	}
}

func TestCodecRejectsBadMagicUnsupportedVersionAndOversize(t *testing.T) {
	codec := NewCodec()

	badMagic := []byte{'B', 'A', 'D', '!', 0, 0, 0, 1, 0, 0, 0, 2, '{', '}'}
	if _, err := codec.ReadFrame(bytes.NewReader(badMagic)); !errors.Is(err, ErrInvalidMagic) {
		t.Fatalf("bad magic err = %v, want ErrInvalidMagic", err)
	}

	unsupported := []byte{'N', 'V', 'O', 'X', 0, 0, 0, 99, 0, 0, 0, 2, '{', '}'}
	if _, err := codec.ReadFrame(bytes.NewReader(unsupported)); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("unsupported version err = %v, want ErrUnsupportedVersion", err)
	}

	var oversized bytes.Buffer
	oversized.Write([]byte{'N', 'V', 'O', 'X', 0, 0, 0, 1, 0, 1, 0, 1})
	if _, err := codec.ReadFrame(&oversized); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("oversized header err = %v, want ErrFrameTooLarge", err)
	}
}

func TestCodecRejectsPayloadLengthMismatch(t *testing.T) {
	codec := NewCodec()
	header := `{"type":"chat.submit","message_id":"msg-1","timestamp":"2026-05-05T12:00:00Z","payload_length":4}`
	var raw bytes.Buffer
	raw.Write([]byte{'N', 'V', 'O', 'X', 0, 0, 0, 1})
	raw.Write([]byte{0, 0, 0, byte(len(header))})
	raw.WriteString(header)
	raw.Write([]byte{0x01, 0x02, 0x03})

	_, err := codec.ReadFrame(&raw)
	if !errors.Is(err, ErrPayloadLengthMismatch) {
		t.Fatalf("err = %v, want ErrPayloadLengthMismatch", err)
	}
}

func TestCodecRejectsInvalidJSONHeader(t *testing.T) {
	codec := NewCodec()
	raw := []byte{'N', 'V', 'O', 'X', 0, 0, 0, 1, 0, 0, 0, 8, '{', 'n', 'o', 't', '-', 'j', 's', '}'}

	_, err := codec.ReadFrame(bytes.NewReader(raw))
	if !errors.Is(err, ErrInvalidHeader) {
		t.Fatalf("err = %v, want ErrInvalidHeader", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("invalid-header error should be redacted, got %v", err)
	}
}
