package navivox

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestServerHelloReturnsServerStatusOverFakeStdio(t *testing.T) {
	codec := NewCodec()
	input := encodeTestFrame(t, codec, Frame{Header: Header{
		Type:        EventHello,
		MessageID:   "hello-1",
		Timestamp:   "2026-05-05T12:00:00Z",
		ContentType: ContentTypeJSON,
	}, Payload: []byte(`{"device":{"id":"phone-1"},"supported_versions":[1]}`)})
	var output bytes.Buffer
	srv := NewServer(ServerOptions{
		Status: StaticStatusProvider{StatusValue: ServerStatus{
			GormesVersion:  "v0.test",
			ConfigVersion:  "cfg-test",
			Protocol:       ProtocolVersion,
			Features:       []string{"chat", "config", "voice"},
			ActiveChannels: []string{"telegram", "navivox"},
		}},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 12, 0, 1, 0, time.UTC) },
		NewID: fixedID("srv-1"),
	})

	if err := srv.Serve(context.Background(), bytes.NewReader(input), &output); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	got, err := codec.ReadFrame(&output)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if got.Header.Type != EventServerStatus || got.Header.CorrelationID != "hello-1" || got.Header.MessageID != "srv-1" {
		t.Fatalf("response header = %+v, want server.status correlated to hello-1", got.Header)
	}
	var status ServerStatus
	if err := json.Unmarshal(got.Payload, &status); err != nil {
		t.Fatalf("status payload: %v", err)
	}
	if status.GormesVersion != "v0.test" || status.ConfigVersion != "cfg-test" || status.Protocol != ProtocolVersion {
		t.Fatalf("status = %+v, want version/config/protocol evidence", status)
	}
}

func TestServerPingReturnsPongWithCorrelationID(t *testing.T) {
	codec := NewCodec()
	input := encodeTestFrame(t, codec, Frame{Header: Header{
		Type:      EventPing,
		MessageID: "ping-1",
		Timestamp: "2026-05-05T12:00:00Z",
	}})
	var output bytes.Buffer
	srv := NewServer(ServerOptions{
		Now:   func() time.Time { return time.Date(2026, 5, 5, 12, 0, 2, 0, time.UTC) },
		NewID: fixedID("srv-2"),
	})

	if err := srv.Serve(context.Background(), bytes.NewReader(input), &output); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	got, err := codec.ReadFrame(&output)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if got.Header.Type != EventPong || got.Header.CorrelationID != "ping-1" || got.Header.MessageID != "srv-2" {
		t.Fatalf("response header = %+v, want pong correlated to ping-1", got.Header)
	}
}

func TestServerUnknownTypeReturnsErrorFrame(t *testing.T) {
	codec := NewCodec()
	input := encodeTestFrame(t, codec, Frame{Header: Header{
		Type:      "unknown.future",
		MessageID: "future-1",
		Timestamp: "2026-05-05T12:00:00Z",
	}})
	var output bytes.Buffer
	srv := NewServer(ServerOptions{
		Now:   func() time.Time { return time.Date(2026, 5, 5, 12, 0, 3, 0, time.UTC) },
		NewID: fixedID("srv-3"),
	})

	if err := srv.Serve(context.Background(), bytes.NewReader(input), &output); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	got, err := codec.ReadFrame(&output)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if got.Header.Type != EventError || got.Header.CorrelationID != "future-1" {
		t.Fatalf("response header = %+v, want correlated error", got.Header)
	}
	var body ErrorBody
	if err := json.Unmarshal(got.Payload, &body); err != nil {
		t.Fatalf("error payload: %v", err)
	}
	if body.Code != ErrorUnsupportedEvent || body.Message == "" {
		t.Fatalf("error body = %+v, want unsupported_event with message", body)
	}
}

func encodeTestFrame(t *testing.T, codec Codec, frame Frame) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := codec.WriteFrame(&buf, frame); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	return buf.Bytes()
}

func fixedID(id string) func() string {
	return func() string { return id }
}
