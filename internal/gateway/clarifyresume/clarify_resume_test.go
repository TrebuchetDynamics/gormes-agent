package clarifyresume

import (
	"context"
	"errors"
	"strings"
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

func TestClarifyResumeBroker_PendingSanitizesHiddenFormattingText(t *testing.T) {
	broker := NewClarifyResumeBroker(func() time.Time { return time.Unix(1700000000, 0).UTC() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errs := make(chan error, 1)
	go func() {
		_, err := broker.Await(ctx, ClarifyResumeRoute{SessionID: "session-1", Platform: "telegram", ChatID: "42"}, tools.ClarifyRequest{
			Question: "Deploy\u202e now?",
			Choices:  []string{"yes\u200d", "no\u2066"},
		})
		errs <- err
	}()

	waitFor(t, 200*time.Millisecond, func() bool {
		_, ok := broker.Pending("telegram", "42")
		return ok
	})
	pending, ok := broker.Pending("telegram", "42")
	if !ok {
		t.Fatal("Pending() = false, want pending route")
	}
	for _, got := range append([]string{pending.Question}, pending.Choices...) {
		for _, forbidden := range []string{"\u202e", "\u200d", "\u2066"} {
			if contains := strings.Contains(got, forbidden); contains {
				t.Fatalf("pending clarify text leaked hidden formatting rune %q in %+v", forbidden, pending)
			}
		}
	}
	if pending.Question != "Deploy now?" || pending.Choices[0] != "yes" || pending.Choices[1] != "no" {
		t.Fatalf("pending clarify text = %+v, want hidden formatting removed", pending)
	}
	cancel()
	<-errs
}

func TestClarifyResumeBroker_AwaitAllowsNilContext(t *testing.T) {
	broker := NewClarifyResumeBroker(func() time.Time { return time.Unix(1700000000, 0).UTC() })
	responses := make(chan tools.ClarifyResponse, 1)
	errs := make(chan error, 1)
	panics := make(chan any, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panics <- r
			}
		}()
		resp, err := broker.Await(nil, ClarifyResumeRoute{SessionID: "session-nil", Platform: "telegram", ChatID: "42"}, tools.ClarifyRequest{Question: "Continue?"})
		responses <- resp
		errs <- err
	}()

	select {
	case r := <-panics:
		t.Fatalf("Await panicked with nil context: %v", r)
	case <-time.After(20 * time.Millisecond):
	}
	waitFor(t, 200*time.Millisecond, func() bool {
		_, ok := broker.Pending("telegram", "42")
		return ok
	})
	if ok := broker.Resume("telegram", "42", "yes"); !ok {
		t.Fatal("Resume() = false, want true for nil-context await")
	}
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("Await() error = %v", err)
		}
	case r := <-panics:
		t.Fatalf("Await panicked with nil context: %v", r)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Await() did not return after nil-context resume")
	}
	if resp := <-responses; resp.UserResponse != "yes" {
		t.Fatalf("UserResponse = %q, want yes", resp.UserResponse)
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

func TestClarifyResumeBroker_NormalizesPlatformCaseForResume(t *testing.T) {
	broker := NewClarifyResumeBroker(func() time.Time { return time.Unix(1700000000, 0).UTC() })
	route := ClarifyResumeRoute{SessionID: "session-1", Platform: " Telegram ", ChatID: "42", MsgID: "m1"}

	responses := make(chan tools.ClarifyResponse, 1)
	errs := make(chan error, 1)
	go func() {
		resp, err := broker.Await(context.Background(), route, tools.ClarifyRequest{Question: "Deploy now?"})
		responses <- resp
		errs <- err
	}()

	waitFor(t, 200*time.Millisecond, func() bool {
		_, ok := broker.Pending("telegram", "42")
		return ok
	})
	if ok := broker.Resume("telegram", "42", "yes"); !ok {
		t.Fatal("Resume() = false, want true for case-normalized platform")
	}
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("Await() error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Await() did not return after case-normalized resume")
	}
	if resp := <-responses; resp.UserResponse != "yes" {
		t.Fatalf("UserResponse = %q, want yes", resp.UserResponse)
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

func TestClarifyResumeBroker_RejectsHiddenFormattingRoute(t *testing.T) {
	broker := NewClarifyResumeBroker(nil)
	for _, route := range []ClarifyResumeRoute{
		{Platform: "telegram\u200b", ChatID: "42"},
		{Platform: "telegram", ChatID: "42\u202e"},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		_, err := broker.Await(ctx, route, tools.ClarifyRequest{Question: "Continue?"})
		cancel()
		if !errors.Is(err, tools.ErrClarifyRouteMissing) {
			t.Fatalf("Await(%+v) error = %v, want ErrClarifyRouteMissing", route, err)
		}
		if _, ok := broker.Pending(route.Platform, route.ChatID); ok {
			t.Fatalf("hidden-formatting route left pending entry for %+v", route)
		}
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
