package googlechat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

var (
	googleChatChatIDPattern  = regexp.MustCompile(`^(?:spaces|users)/[A-Za-z0-9_-]+$`)
	googleChatThreadPattern  = regexp.MustCompile(`^spaces/[A-Za-z0-9_-]+/threads/[A-Za-z0-9_-]+$`)
	googleChatMessageBaseURL = "https://chat.googleapis.com/v1/"
)

type GoogleChatStandaloneTokenSource interface {
	GoogleChatBearerToken(ctx context.Context) (string, error)
}

type GoogleChatStandalonePoster interface {
	PostGoogleChatMessage(ctx context.Context, req GoogleChatStandalonePostRequest) (GoogleChatStandalonePostResponse, error)
}

type GoogleChatStandalonePostRequest struct {
	URL     string
	Headers map[string]string
	Body    []byte
}

type GoogleChatStandalonePostResponse struct {
	Name string
}

type StandaloneSender struct {
	TokenSource GoogleChatStandaloneTokenSource
	Poster      GoogleChatStandalonePoster
}

func (s *StandaloneSender) DeliverCronStandalone(ctx context.Context, target cron.DeliveryTarget, text string, media []cron.MediaAttachment) error {
	if !strings.EqualFold(strings.TrimSpace(target.Platform), PlatformName) {
		return cron.ErrStandaloneSenderUnavailable
	}
	_, err := s.SendText(ctx, target.ChatID, target.ThreadID, text)
	return err
}

func (s *StandaloneSender) SendText(ctx context.Context, chatID, threadID, text string) (string, error) {
	chatID = strings.TrimSpace(chatID)
	threadID = strings.TrimSpace(threadID)
	if !googleChatChatIDPattern.MatchString(chatID) {
		return "", fmt.Errorf("googlechat standalone send: chat_id %q must match spaces/<id> or users/<id>", chatID)
	}
	if threadID != "" && !googleChatThreadPattern.MatchString(threadID) {
		return "", fmt.Errorf("googlechat standalone send: thread_id %q must match spaces/<id>/threads/<id>", threadID)
	}
	if s == nil || s.TokenSource == nil || s.Poster == nil {
		return "", cron.ErrStandaloneSenderUnavailable
	}
	token, err := s.TokenSource.GoogleChatBearerToken(ctx)
	if err != nil {
		return "", fmt.Errorf("googlechat standalone send: token source failed: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("googlechat standalone send: empty bearer token")
	}
	body, err := googleChatStandaloneMessageBody(text, threadID)
	if err != nil {
		return "", err
	}
	resp, err := s.Poster.PostGoogleChatMessage(ctx, GoogleChatStandalonePostRequest{
		URL: googleChatMessageBaseURL + chatID + "/messages",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		},
		Body: body,
	})
	if err != nil {
		return "", fmt.Errorf("googlechat standalone send: post failed: %w", err)
	}
	return resp.Name, nil
}

func googleChatStandaloneMessageBody(text, threadID string) ([]byte, error) {
	payload := struct {
		Text   string `json:"text"`
		Thread *struct {
			Name string `json:"name"`
		} `json:"thread,omitempty"`
	}{Text: text}
	if threadID != "" {
		payload.Thread = &struct {
			Name string `json:"name"`
		}{Name: threadID}
	}
	return json.Marshal(payload)
}
