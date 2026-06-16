package discord

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewRealSession_EnablesForumThreadLifecycleIntents(t *testing.T) {
	session, err := NewRealSession("token")
	if err != nil {
		t.Fatalf("NewRealSession() error = %v", err)
	}

	real, ok := session.(*realSession)
	if !ok {
		t.Fatalf("NewRealSession() returned %T, want *realSession", session)
	}

	want := discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent
	if got := real.s.Identify.Intents; got&want != want {
		t.Fatalf("Identify.Intents = %v, want all bits %v", got, want)
	}
}

func TestRealSessionReadAttachmentUsesBoundedHTTPClientAndKeepsAuthorization(t *testing.T) {
	var called bool
	real := &realSession{
		s: &discordgo.Session{Token: "Bot discord-token"},
		attachmentHTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			called = true
			if got := req.Header.Get("Authorization"); got != "Bot discord-token" {
				t.Fatalf("Authorization header = %q, want bot token", got)
			}
			if got := req.URL.Host; got != "cdn.discordapp.com" {
				t.Fatalf("attachment host = %q, want cdn.discordapp.com", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(testDiscordPNG)),
			}, nil
		})},
	}

	got, err := real.ReadAttachment(context.Background(), &discordgo.MessageAttachment{URL: "https://cdn.discordapp.com/attachments/x/photo.png"})
	if err != nil {
		t.Fatalf("ReadAttachment() error = %v", err)
	}
	if !called {
		t.Fatal("ReadAttachment did not use the configured HTTP client")
	}
	if !bytes.Equal(got, testDiscordPNG) {
		t.Fatalf("ReadAttachment bytes = %q, want test PNG", string(got))
	}
}
