package dingtalk

import (
	"context"
	"testing"
)

var _ Client = (*StreamClient)(nil)

func TestNewStreamClient_AcceptsValidCredentials(t *testing.T) {
	sc := NewStreamClient("my-client-id", "my-client-secret", nil)
	if sc == nil {
		t.Fatal("NewStreamClient returned nil")
	}
	if sc.Events() == nil {
		t.Fatal("Events() channel is nil")
	}
}

func TestStreamClient_SendReply_RejectsEmptyWebhook(t *testing.T) {
	sc := NewStreamClient("x", "x", nil)
	_, err := sc.SendReply(context.Background(), "", "hello")
	if err == nil {
		t.Fatal("expected error for empty webhook, got nil")
	}
}

func TestStreamClient_CloseReturnsNil(t *testing.T) {
	sc := NewStreamClient("x", "x", nil)
	if err := sc.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}
}
