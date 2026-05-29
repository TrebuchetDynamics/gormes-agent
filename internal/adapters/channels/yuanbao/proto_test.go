package yuanbao

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestYuanbaoProto_DecodesInboundTextFixture(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		fixture    string
		wantSource string
		wantConvID string
		wantMsgID  string
		wantRole   string
		wantText   string
	}{
		{
			name:       "c2c text from alice",
			fixture:    "inbound_text_c2c.pb",
			wantSource: "yuanbao",
			wantConvID: "alice",
			wantMsgID:  "msg-001",
			wantRole:   "user",
			wantText:   "Hello, Gormes!",
		},
		{
			name:       "group text from bob",
			fixture:    "inbound_text_group.pb",
			wantSource: "yuanbao",
			wantConvID: "group-789",
			wantMsgID:  "msg-grp-9",
			wantRole:   "user",
			wantText:   "Group hi from Bob.",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			raw, err := os.ReadFile(filepath.Join("testdata", tc.fixture))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			env, err := DecodeInboundEnvelope(raw)
			if err != nil {
				t.Fatalf("DecodeInboundEnvelope: %v", err)
			}
			if env.Source != tc.wantSource {
				t.Errorf("Source = %q, want %q", env.Source, tc.wantSource)
			}
			if env.ConversationID != tc.wantConvID {
				t.Errorf("ConversationID = %q, want %q", env.ConversationID, tc.wantConvID)
			}
			if env.MessageID != tc.wantMsgID {
				t.Errorf("MessageID = %q, want %q", env.MessageID, tc.wantMsgID)
			}
			if env.AuthorRole != tc.wantRole {
				t.Errorf("AuthorRole = %q, want %q", env.AuthorRole, tc.wantRole)
			}
			if env.Text != tc.wantText {
				t.Errorf("Text = %q, want %q", env.Text, tc.wantText)
			}
		})
	}
}

func TestYuanbaoProto_MalformedReturnsDegradedEvidence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  []byte
	}{
		{name: "truncated varint", raw: []byte{0x0a, 0xff}},
		{name: "bad wire type", raw: []byte{0x07}},
		{name: "length overrun", raw: []byte{0x0a, 0x10, 0x01}},
		{name: "empty", raw: []byte{}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			env, err := DecodeInboundEnvelope(tc.raw)
			if err == nil {
				t.Fatalf("expected degraded error, got envelope %#v", env)
			}
			var deg *DegradedError
			if !errors.As(err, &deg) {
				t.Fatalf("expected *DegradedError, got %T: %v", err, err)
			}
			if deg.Code != DegradedProtocolUnavailable {
				t.Errorf("Code = %q, want %q", deg.Code, DegradedProtocolUnavailable)
			}
			if deg.Detail == "" {
				t.Errorf("Detail must be non-empty")
			}
		})
	}
}

func TestYuanbaoProto_UnknownEnvelopeReturnsDegradedEvidence(t *testing.T) {
	t.Parallel()

	// Structurally valid protobuf with no recognizable inbound-push fields:
	// a single varint at field 99. Tag = (99<<3)|0 = 792 = 0x98 0x06.
	raw := []byte{0x98, 0x06, 0x01}

	env, err := DecodeInboundEnvelope(raw)
	if err == nil {
		t.Fatalf("expected degraded error, got envelope %#v", env)
	}
	var deg *DegradedError
	if !errors.As(err, &deg) {
		t.Fatalf("expected *DegradedError, got %T: %v", err, err)
	}
	if deg.Code != DegradedProtocolUnavailable {
		t.Errorf("Code = %q, want %q", deg.Code, DegradedProtocolUnavailable)
	}
}

func TestYuanbaoProto_DecodeDoesNotPanicOnRandomBytes(t *testing.T) {
	t.Parallel()

	// 256 increasing bytes — must never panic, must always return *DegradedError or a benign envelope.
	raw := make([]byte, 256)
	for i := range raw {
		raw[i] = byte(i)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DecodeInboundEnvelope panicked: %v", r)
		}
	}()
	if _, err := DecodeInboundEnvelope(raw); err != nil {
		var deg *DegradedError
		if !errors.As(err, &deg) {
			t.Fatalf("expected *DegradedError, got %T: %v", err, err)
		}
	}
}
