package clarifyresume

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !cond() {
		t.Fatalf("condition not met within %s", timeout)
	}
}

func TestClarifyResumeBroker_PersistsOneShotRouteAndClearsAfterReply(t *testing.T) {
	broker := NewClarifyResumeBroker(func() time.Time { return time.Unix(1700000000, 0).UTC() })
	route := ClarifyResumeRoute{SessionID: "session-1", Platform: "telegram", ChatID: "42", MsgID: "m1"}
	req := tools.ClarifyRequest{Question: "Deploy now?", Choices: []string{"yes", "no"}}

	responses := make(chan tools.ClarifyResponse, 1)
	errs := make(chan error, 1)
	go func() {
		resp, err := broker.Await(context.Background(), route, req)
		responses <- resp
		errs <- err
	}()

	waitFor(t, 200*time.Millisecond, func() bool {
		pending, ok := broker.Pending("telegram", "42")
		return ok && pending.Question == "Deploy now?" && len(pending.Choices) == 2
	})

	if ok := broker.Resume("telegram", "42", "  yes  "); !ok {
		t.Fatal("Resume() = false, want true for pending route")
	}

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("Await() error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Await() did not return after resume")
	}
	resp := <-responses
	if resp.UserResponse != "yes" {
		t.Fatalf("UserResponse = %q, want trimmed answer", resp.UserResponse)
	}
	if _, ok := broker.Pending("telegram", "42"); ok {
		t.Fatal("route still pending after first resume; want one-shot cleanup")
	}
	if ok := broker.Resume("telegram", "42", "again"); ok {
		t.Fatal("second Resume() = true, want false after one-shot cleanup")
	}
}

func TestClarifyResumeBroker_AwaitTimeoutClearsRoute(t *testing.T) {
	broker := NewClarifyResumeBroker(func() time.Time { return time.Unix(1700000000, 0).UTC() })
	route := ClarifyResumeRoute{SessionID: "session-2", Platform: "telegram", ChatID: "42"}
	ctx, cancel := context.WithCancel(context.Background())

	errs := make(chan error, 1)
	go func() {
		_, err := broker.Await(ctx, route, tools.ClarifyRequest{Question: "Continue?"})
		errs <- err
	}()
	waitFor(t, 200*time.Millisecond, func() bool {
		_, ok := broker.Pending("telegram", "42")
		return ok
	})
	cancel()

	select {
	case err := <-errs:
		if !errors.Is(err, tools.ErrClarifyTimeout) {
			t.Fatalf("Await() error = %v, want ErrClarifyTimeout", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Await() did not return after context cancellation")
	}
	if _, ok := broker.Pending("telegram", "42"); ok {
		t.Fatal("route still pending after timeout; want cleanup")
	}
}

func TestClarifyResumeBroker_RejectsMissingRoute(t *testing.T) {
	broker := NewClarifyResumeBroker(nil)
	if ok := broker.Resume("telegram", "42", "answer"); ok {
		t.Fatal("Resume() = true, want false when no route is pending")
	}
	_, err := broker.Await(context.Background(), ClarifyResumeRoute{Platform: "telegram"}, tools.ClarifyRequest{Question: "Missing chat?"})
	if !errors.Is(err, tools.ErrClarifyRouteMissing) {
		t.Fatalf("Await() error = %v, want ErrClarifyRouteMissing", err)
	}
}
