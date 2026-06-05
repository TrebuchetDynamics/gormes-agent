package googlechat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

func TestGoogleChatStandaloneSenderValidatesResourceNames(t *testing.T) {
	for _, tc := range []struct {
		name    string
		chatID  string
		thread  string
		wantErr string
	}{
		{name: "bad_space_path", chatID: "spaces/../secret", wantErr: "chat_id"},
		{name: "wrong_space_prefix", chatID: "rooms/AAA", wantErr: "chat_id"},
		{name: "query_injection", chatID: "spaces/AAA?alt=json", wantErr: "chat_id"},
		{name: "bad_thread_path", chatID: "spaces/AAA", thread: "spaces/AAA/threads/../secret", wantErr: "thread_id"},
		{name: "wrong_thread_prefix", chatID: "spaces/AAA", thread: "rooms/AAA/threads/T1", wantErr: "thread_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tokenSource := &fakeGoogleChatTokenSource{token: "secret-token"}
			poster := &fakeGoogleChatPoster{messageName: "spaces/AAA/messages/msg-1"}
			sender := StandaloneSender{TokenSource: tokenSource, Poster: poster}

			_, err := sender.SendText(context.Background(), tc.chatID, tc.thread, "hello")
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("SendText error = %v, want %q", err, tc.wantErr)
			}
			if tokenSource.calls != 0 {
				t.Fatalf("token source calls = %d, want validation before token lookup", tokenSource.calls)
			}
			if len(poster.requests) != 0 {
				t.Fatalf("poster requests = %d, want validation before post", len(poster.requests))
			}
		})
	}

	tokenSource := &fakeGoogleChatTokenSource{token: "secret-token"}
	poster := &fakeGoogleChatPoster{messageName: "users/BBB/messages/msg-2"}
	sender := StandaloneSender{TokenSource: tokenSource, Poster: poster}
	for _, target := range []struct {
		chatID string
		thread string
	}{
		{chatID: "spaces/AAA"},
		{chatID: "users/BBB"},
		{chatID: "spaces/AAA", thread: "spaces/AAA/threads/thread_1-2"},
	} {
		if _, err := sender.SendText(context.Background(), target.chatID, target.thread, "hello"); err != nil {
			t.Fatalf("SendText(%q,%q) error = %v, want nil", target.chatID, target.thread, err)
		}
	}
}

func TestGoogleChatStandaloneSenderPostsTextOnlyMessage(t *testing.T) {
	tokenSource := &fakeGoogleChatTokenSource{token: "secret-token"}
	poster := &fakeGoogleChatPoster{messageName: "spaces/AAA/messages/msg-1"}
	sender := StandaloneSender{TokenSource: tokenSource, Poster: poster}

	messageName, err := sender.SendText(context.Background(), "spaces/AAA", "spaces/AAA/threads/thread-1", "hello from cron")
	if err != nil {
		t.Fatalf("SendText error = %v", err)
	}
	if messageName != "spaces/AAA/messages/msg-1" {
		t.Fatalf("messageName = %q", messageName)
	}

	target := cron.DeliveryTarget{Platform: PlatformName, ChatID: "spaces/AAA", ThreadID: "spaces/AAA/threads/thread-1"}
	err = sender.DeliverCronStandalone(context.Background(), target, "standalone cron response", []cron.MediaAttachment{{Path: "/tmp/report.png"}})
	if err != nil {
		t.Fatalf("DeliverCronStandalone error = %v", err)
	}
	if len(poster.requests) != 2 {
		t.Fatalf("poster requests = %d, want 2", len(poster.requests))
	}
	req := poster.requests[1]
	if req.URL != "https://chat.googleapis.com/v1/spaces/AAA/messages" {
		t.Fatalf("URL = %q", req.URL)
	}
	if got := req.Headers["Authorization"]; got != "Bearer secret-token" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := req.Headers["Content-Type"]; got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}

	var body map[string]any
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatalf("unmarshal body %s: %v", req.Body, err)
	}
	if body["text"] != "standalone cron response" {
		t.Fatalf("text body = %#v", body)
	}
	thread, ok := body["thread"].(map[string]any)
	if !ok || thread["name"] != "spaces/AAA/threads/thread-1" {
		t.Fatalf("thread body = %#v", body["thread"])
	}
	if _, ok := body["media"]; ok {
		t.Fatalf("body contains media field despite text-only standalone path: %#v", body)
	}
}

func TestGoogleChatStandaloneSenderFailureModes(t *testing.T) {
	target := cron.DeliveryTarget{Platform: PlatformName, ChatID: "spaces/AAA"}

	err := (&StandaloneSender{}).DeliverCronStandalone(context.Background(), target, "hello", nil)
	if !errors.Is(err, cron.ErrStandaloneSenderUnavailable) {
		t.Fatalf("nil sender error = %v, want ErrStandaloneSenderUnavailable", err)
	}

	emptyToken := &StandaloneSender{
		TokenSource: &fakeGoogleChatTokenSource{},
		Poster:      &fakeGoogleChatPoster{messageName: "spaces/AAA/messages/msg-1"},
	}
	err = emptyToken.DeliverCronStandalone(context.Background(), target, "hello", nil)
	if err == nil || !strings.Contains(err.Error(), "empty bearer token") {
		t.Fatalf("empty token error = %v", err)
	}

	postFailure := &StandaloneSender{
		TokenSource: &fakeGoogleChatTokenSource{token: "secret-token"},
		Poster:      &fakeGoogleChatPoster{err: errors.New("backend unavailable")},
	}
	err = postFailure.DeliverCronStandalone(context.Background(), target, "hello", nil)
	if err == nil || !strings.Contains(err.Error(), "post failed") {
		t.Fatalf("post failure error = %v", err)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("post failure leaked token in error: %v", err)
	}
}

func TestGoogleChatStandaloneSenderImplementsCronAdapter(t *testing.T) {
	var _ cron.StandaloneDeliveryAdapter = (*StandaloneSender)(nil)

	sender := &StandaloneSender{
		TokenSource: &fakeGoogleChatTokenSource{token: "secret-token"},
		Poster:      &fakeGoogleChatPoster{messageName: "spaces/AAA/messages/msg-1"},
	}
	err := sender.DeliverCronStandalone(context.Background(), cron.DeliveryTarget{Platform: "teams", ChatID: "spaces/AAA"}, "hello", nil)
	if !errors.Is(err, cron.ErrStandaloneSenderUnavailable) {
		t.Fatalf("unsupported platform error = %v, want ErrStandaloneSenderUnavailable", err)
	}
}

type fakeGoogleChatTokenSource struct {
	token string
	err   error
	calls int
}

func (s *fakeGoogleChatTokenSource) GoogleChatBearerToken(context.Context) (string, error) {
	s.calls++
	return s.token, s.err
}

type fakeGoogleChatPoster struct {
	messageName string
	err         error
	requests    []GoogleChatStandalonePostRequest
}

func (p *fakeGoogleChatPoster) PostGoogleChatMessage(_ context.Context, req GoogleChatStandalonePostRequest) (GoogleChatStandalonePostResponse, error) {
	p.requests = append(p.requests, cloneGoogleChatStandalonePostRequest(req))
	if p.err != nil {
		return GoogleChatStandalonePostResponse{}, p.err
	}
	return GoogleChatStandalonePostResponse{Name: p.messageName}, nil
}

func cloneGoogleChatStandalonePostRequest(req GoogleChatStandalonePostRequest) GoogleChatStandalonePostRequest {
	clone := GoogleChatStandalonePostRequest{
		URL:     req.URL,
		Headers: map[string]string{},
		Body:    append([]byte(nil), req.Body...),
	}
	for k, v := range req.Headers {
		clone.Headers[k] = v
	}
	return clone
}
