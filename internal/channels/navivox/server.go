package navivox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"
)

const ErrorUnsupportedEvent = "unsupported_event"

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ServerOptions struct {
	Codec  Codec
	Status StatusProvider
	Now    func() time.Time
	NewID  func() string
	Log    *slog.Logger
}

type Server struct {
	codec  Codec
	status StatusProvider
	now    func() time.Time
	newID  func() string
	log    *slog.Logger
}

func NewServer(opts ServerOptions) *Server {
	codec := opts.Codec
	if codec.MaxHeaderBytes == 0 && codec.MaxPayloadBytes == 0 {
		codec = NewCodec()
	}
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	newID := opts.NewID
	if newID == nil {
		newID = func() string { return fmt.Sprintf("navivox-%d", now().UnixNano()) }
	}
	status := opts.Status
	if status == nil {
		status = StaticStatusProvider{StatusValue: ServerStatus{
			Protocol: ProtocolVersion,
			Features: DefaultFeatures(),
		}}
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	return &Server{codec: codec, status: status, now: now, newID: newID, log: log}
}

func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		frame, err := s.codec.ReadFrame(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := s.handle(ctx, w, frame); err != nil {
			return err
		}
	}
}

func (s *Server) handle(ctx context.Context, w io.Writer, frame Frame) error {
	switch frame.Header.Type {
	case EventHello:
		status, err := s.status.Status(ctx)
		if err != nil {
			return s.writeError(w, frame.Header.MessageID, "status_unavailable", "server status unavailable")
		}
		if status.Protocol == 0 {
			status.Protocol = ProtocolVersion
		}
		if len(status.Features) == 0 {
			status.Features = DefaultFeatures()
		}
		body, err := json.Marshal(status)
		if err != nil {
			return err
		}
		return s.write(w, Header{
			Type:          EventServerStatus,
			MessageID:     s.newID(),
			CorrelationID: frame.Header.MessageID,
			Timestamp:     s.now().UTC().Format(time.RFC3339Nano),
			ContentType:   ContentTypeJSON,
		}, body)
	case EventPing:
		return s.write(w, Header{
			Type:          EventPong,
			MessageID:     s.newID(),
			CorrelationID: frame.Header.MessageID,
			Timestamp:     s.now().UTC().Format(time.RFC3339Nano),
		}, nil)
	default:
		return s.writeError(w, frame.Header.MessageID, ErrorUnsupportedEvent, "unsupported Navivox event type")
	}
}

func (s *Server) writeError(w io.Writer, correlationID, code, message string) error {
	body, err := json.Marshal(ErrorBody{Code: code, Message: message})
	if err != nil {
		return err
	}
	return s.write(w, Header{
		Type:          EventError,
		MessageID:     s.newID(),
		CorrelationID: correlationID,
		Timestamp:     s.now().UTC().Format(time.RFC3339Nano),
		ContentType:   ContentTypeJSON,
	}, body)
}

func (s *Server) write(w io.Writer, header Header, payload []byte) error {
	return s.codec.WriteFrame(w, Frame{Header: header, Payload: payload})
}
