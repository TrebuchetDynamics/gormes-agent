package telegram

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var errNetworkError = errors.New("telegram: NetworkError: connection reset")
var errTimedOut = errors.New("telegram: TimedOut: request timeout")

func TestBot_SendThread_RetriesTransientNetworkError(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		if callCount < 3 {
			return nil, errNetworkError
		}
		return client.uploadSuccess(1001), nil
	}
	b := New(Config{}, client, nil)

	msgID, err := b.SendThread(context.Background(), "42", "777", "hello")
	if err != nil {
		t.Fatalf("SendThread should succeed after retries: %v", err)
	}
	if msgID != "1001" {
		t.Fatalf("msgID = %q, want 1001", msgID)
	}
	if callCount != 3 {
		t.Fatalf("callCount = %d, want 3 (2 retries then success)", callCount)
	}
}

func TestBot_SendThread_NetworkErrorExhaustsRetries(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		return nil, errNetworkError
	}
	b := New(Config{}, client, nil)

	_, err := b.SendThread(context.Background(), "42", "777", "hello")
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if !errors.Is(err, errNetworkError) {
		t.Fatalf("err = %v, want errNetworkError", err)
	}
	if callCount != maxSendRetries {
		t.Fatalf("callCount = %d, want %d", callCount, maxSendRetries)
	}
}

func TestBot_SendThread_TimedOutDoesNotRetry(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		return nil, errTimedOut
	}
	b := New(Config{}, client, nil)

	_, err := b.SendThread(context.Background(), "42", "777", "hello")
	if err == nil {
		t.Fatal("expected TimedOut error to fail immediately")
	}
	if !errors.Is(err, errTimedOut) {
		t.Fatalf("err = %v, want errTimedOut", err)
	}
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1 (TimedOut must not retry)", callCount)
	}
}

func TestBot_SendThread_ThreadNotFoundTakesPrecedenceOverNetworkRetry(t *testing.T) {
	// Thread-not-found MUST be handled before generic retry: it retries exactly
	// once without thread, not up to maxSendRetries with thread.
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		if callCount == 1 {
			return nil, errThreadNotFound
		}
		// Second call succeeds (thread removed)
		return client.uploadSuccess(2002), nil
	}
	b := New(Config{}, client, nil)

	msgID, err := b.SendThread(context.Background(), "42", "777", "hello")
	if err != nil {
		t.Fatalf("SendThread should succeed after thread-not-found retry: %v", err)
	}
	if msgID != "2002" {
		t.Fatalf("msgID = %q, want 2002", msgID)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2 (thread-not-found retry, not 3-attempt network retry)", callCount)
	}
}

func TestBot_SendThreadChatAction_RetriesTransientNetworkError(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		if callCount < 3 {
			return nil, errNetworkError
		}
		return &tgbotapi.APIResponse{Ok: true}, nil
	}
	b := New(Config{}, client, nil)

	err := b.SendThreadChatAction(context.Background(), "-100123", "777", "typing")
	if err != nil {
		t.Fatalf("SendThreadChatAction should succeed after retries: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("callCount = %d, want 3", callCount)
	}
}

func TestBot_SendThreadChatAction_TimedOutDoesNotRetry(t *testing.T) {
	client := newMockClient()
	callCount := 0
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		callCount++
		return nil, errTimedOut
	}
	b := New(Config{}, client, nil)

	err := b.SendThreadChatAction(context.Background(), "-100123", "777", "typing")
	if err == nil {
		t.Fatal("expected TimedOut error to fail immediately")
	}
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1", callCount)
	}
}
