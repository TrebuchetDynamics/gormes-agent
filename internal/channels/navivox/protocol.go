package navivox

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	PlatformName           = "navivox"
	ProtocolVersion uint32 = 1

	EventHello        = "hello"
	EventServerStatus = "server.status"
	EventPing         = "ping"
	EventPong         = "pong"
	EventError        = "error"
	EventVoiceAudio   = "voice.audio"

	ContentTypeJSON = "application/json"

	DefaultMaxHeaderBytes  uint32 = 64 * 1024
	DefaultMaxPayloadBytes uint32 = 8 * 1024 * 1024
)

var (
	ErrInvalidMagic          = errors.New("navivox: invalid frame magic")
	ErrUnsupportedVersion    = errors.New("navivox: unsupported protocol version")
	ErrFrameTooLarge         = errors.New("navivox: frame too large")
	ErrInvalidHeader         = errors.New("navivox: invalid frame header")
	ErrPayloadLengthMismatch = errors.New("navivox: payload length mismatch")
)

var frameMagic = [4]byte{'N', 'V', 'O', 'X'}

type Header struct {
	Type          string         `json:"type"`
	MessageID     string         `json:"message_id"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Timestamp     string         `json:"timestamp"`
	PayloadLength uint32         `json:"payload_length"`
	ContentType   string         `json:"content_type,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type Frame struct {
	Header  Header
	Payload []byte
}

type Codec struct {
	MaxHeaderBytes  uint32
	MaxPayloadBytes uint32
}

func NewCodec() Codec {
	return Codec{
		MaxHeaderBytes:  DefaultMaxHeaderBytes,
		MaxPayloadBytes: DefaultMaxPayloadBytes,
	}
}

func (c Codec) ReadFrame(r io.Reader) (Frame, error) {
	c = c.withDefaults()
	var prelude [12]byte
	if _, err := io.ReadFull(r, prelude[:]); err != nil {
		return Frame{}, err
	}
	if [4]byte(prelude[0:4]) != frameMagic {
		return Frame{}, ErrInvalidMagic
	}
	version := binary.BigEndian.Uint32(prelude[4:8])
	if version != ProtocolVersion {
		return Frame{}, fmt.Errorf("%w: %d", ErrUnsupportedVersion, version)
	}
	headerLen := binary.BigEndian.Uint32(prelude[8:12])
	if headerLen == 0 || headerLen > c.MaxHeaderBytes {
		return Frame{}, fmt.Errorf("%w: header length %d", ErrFrameTooLarge, headerLen)
	}
	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(r, headerBytes); err != nil {
		return Frame{}, fmt.Errorf("%w: read header", ErrInvalidHeader)
	}
	var header Header
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Frame{}, fmt.Errorf("%w: malformed JSON", ErrInvalidHeader)
	}
	if header.PayloadLength > c.MaxPayloadBytes {
		return Frame{}, fmt.Errorf("%w: payload length %d", ErrFrameTooLarge, header.PayloadLength)
	}
	payload := make([]byte, header.PayloadLength)
	if header.PayloadLength > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return Frame{}, fmt.Errorf("%w: wanted %d bytes", ErrPayloadLengthMismatch, header.PayloadLength)
		}
	}
	return Frame{Header: header, Payload: payload}, nil
}

func (c Codec) WriteFrame(w io.Writer, f Frame) error {
	c = c.withDefaults()
	header := f.Header
	header.PayloadLength = uint32(len(f.Payload))
	if header.PayloadLength > c.MaxPayloadBytes {
		return fmt.Errorf("%w: payload length %d", ErrFrameTooLarge, header.PayloadLength)
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("%w: marshal header", ErrInvalidHeader)
	}
	if len(headerBytes) == 0 || uint32(len(headerBytes)) > c.MaxHeaderBytes {
		return fmt.Errorf("%w: header length %d", ErrFrameTooLarge, len(headerBytes))
	}

	var prelude [12]byte
	copy(prelude[0:4], frameMagic[:])
	binary.BigEndian.PutUint32(prelude[4:8], ProtocolVersion)
	binary.BigEndian.PutUint32(prelude[8:12], uint32(len(headerBytes)))
	if _, err := w.Write(prelude[:]); err != nil {
		return err
	}
	if _, err := w.Write(headerBytes); err != nil {
		return err
	}
	if len(f.Payload) > 0 {
		_, err = w.Write(f.Payload)
	}
	return err
}

func (c Codec) withDefaults() Codec {
	if c.MaxHeaderBytes == 0 {
		c.MaxHeaderBytes = DefaultMaxHeaderBytes
	}
	if c.MaxPayloadBytes == 0 {
		c.MaxPayloadBytes = DefaultMaxPayloadBytes
	}
	return c
}
