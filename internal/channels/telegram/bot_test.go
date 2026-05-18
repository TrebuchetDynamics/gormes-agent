package telegram

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestBot_Name(t *testing.T) {
	b := New(Config{AllowedChatID: 42}, newMockClient(), nil)
	if got := b.Name(); got != "telegram" {
		t.Errorf("Name() = %q, want telegram", got)
	}
}

func TestBot_RunRegistersHermesTelegramCommands(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = b.Run(ctx, inbox) }()
	defer cancel()

	cfg := waitForSetMyCommands(t, mc)
	seen := map[string]bool{}
	for _, cmd := range cfg.Commands {
		seen[cmd.Command] = true
	}
	for _, want := range []string{"new", "retry", "undo", "title", "branch", "compress", "rollback", "snapshot", "stop", "approve", "deny", "background", "btw", "agents", "queue", "steer", "status", "topic"} {
		if !seen[want] {
			t.Fatalf("setMyCommands missing %q in %#v", want, cfg.Commands)
		}
	}
}

func TestBot_RunRegistersDynamicSkillTelegramCommands(t *testing.T) {
	mc := newMockClient()
	b := New(Config{
		AllowedChatID: 42,
		DynamicCommands: []gateway.PlatformCommand{
			{Name: "jellyfin-jellystat-24h-summary", Description: "Summarize media stats"},
			{Name: "", Description: "ignored"},
		},
	}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = b.Run(ctx, inbox) }()
	defer cancel()

	cfg := waitForSetMyCommands(t, mc)
	for _, cmd := range cfg.Commands {
		if cmd.Command == "jellyfin_jellystat_24h_summary" && cmd.Description == "Summarize media stats" {
			return
		}
	}
	t.Fatalf("setMyCommands missing dynamic skill command in %#v", cfg.Commands)
}

func TestBot_RunCapsTelegramCommandsAtPlatformLimit(t *testing.T) {
	mc := newMockClient()
	dynamic := make([]gateway.PlatformCommand, 0, 120)
	for i := 0; i < 120; i++ {
		dynamic = append(dynamic, gateway.PlatformCommand{Name: strings.ReplaceAll("skill-extra-"+time.Unix(int64(i), 0).UTC().Format("150405"), ":", ""), Description: "Extra skill"})
	}
	var logs bytes.Buffer
	b := New(Config{AllowedChatID: 42, DynamicCommands: dynamic}, mc, slog.New(slog.NewTextHandler(&logs, nil)))
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = b.Run(ctx, inbox) }()
	defer cancel()

	cfg := waitForSetMyCommands(t, mc)
	if len(cfg.Commands) != telegramCommandLimit {
		t.Fatalf("registered command count = %d, want %d", len(cfg.Commands), telegramCommandLimit)
	}
	if !strings.Contains(logs.String(), "hidden_count") {
		t.Fatalf("cap log = %q, want hidden_count evidence", logs.String())
	}
}

func waitForSetMyCommands(t *testing.T, mc *mockClient) tgbotapi.SetMyCommandsConfig {
	t.Helper()
	deadline := time.After(200 * time.Millisecond)
	for {
		requests := mc.requestMessages()
		for _, req := range requests {
			cfg, ok := req.(tgbotapi.SetMyCommandsConfig)
			if ok {
				return cfg
			}
		}
		select {
		case <-deadline:
			t.Fatalf("Run did not issue setMyCommands; requests=%#v", requests)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestBot_ToInboundEvent_Submit(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.pushTextUpdate(42, "hello there")

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit || ev.Text != "hello there" {
			t.Errorf("got %+v", ev)
		}
		if ev.Platform != "telegram" || ev.ChatID != "42" {
			t.Errorf("got %+v", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_AccountID(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42, AccountID: "mineru"}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.pushTextUpdate(42, "hello from mineru")

	select {
	case ev := <-inbox:
		if ev.AccountID != "mineru" {
			t.Errorf("AccountID = %q, want mineru", ev.AccountID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_TopicCommandPreservesPrivateMetadata(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.updatesCh <- tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			MessageID: 99,
			Text:      "/topic help",
			Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
			From:      &tgbotapi.User{ID: 7, FirstName: "juan"},
		},
	}

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventTopic {
			t.Fatalf("Kind = %v, want EventTopic", ev.Kind)
		}
		if ev.Text != "/topic help" {
			t.Fatalf("Text = %q, want /topic help", ev.Text)
		}
		if ev.ChatType != "private" {
			t.Fatalf("ChatType = %q, want private", ev.ChatType)
		}
		if ev.UserID != "7" {
			t.Fatalf("UserID = %q, want 7", ev.UserID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_VoiceMessageIsNotBlank(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.pushVoiceUpdate(42, tgbotapi.Voice{
		FileID:       "voice-file-id",
		FileUniqueID: "voice-unique-id",
		Duration:     66,
		MimeType:     "audio/ogg",
	})

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit {
			t.Fatalf("Kind = %v, want submit", ev.Kind)
		}
		if strings.TrimSpace(ev.Text) == "" {
			t.Fatalf("voice event text is blank: %+v", ev)
		}
		if !strings.Contains(ev.Text, "Telegram voice message") || !strings.Contains(ev.Text, "66s") {
			t.Fatalf("voice marker text = %q", ev.Text)
		}
		submit := ev.SubmitText()
		if strings.TrimSpace(submit) == "" || !strings.Contains(submit, "Telegram voice message") {
			t.Fatalf("SubmitText() = %q, want nonblank voice attachment evidence", submit)
		}
		if !strings.Contains(submit, "audio transcription is not configured") {
			t.Fatalf("SubmitText() = %q, want sanitized STT configuration marker", submit)
		}
		for _, forbidden := range []string{"voice-file-id", "sourceId=", ".cache/gormes", "~/.cache/gormes"} {
			if strings.Contains(submit, forbidden) {
				t.Fatalf("SubmitText() leaks transport/cache detail %q: %q", forbidden, submit)
			}
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_VoiceMessageIncludesTranscriptWhenTranscriberConfigured(t *testing.T) {
	mc := newMockClient()
	mc.telegramFiles["voice-file-id"] = tgbotapi.File{FileID: "voice-file-id", FilePath: "voice/file_0.ogg"}
	mc.downloads["voice/file_0.ogg"] = []byte("ogg bytes")
	b := New(Config{
		AllowedChatID: 42,
		AudioTranscriber: fakeAudioTranscriber{
			Transcript: "please check the Gormes audio path",
		},
	}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.pushVoiceUpdate(42, tgbotapi.Voice{
		FileID:       "voice-file-id",
		FileUniqueID: "voice-unique-id",
		Duration:     9,
		MimeType:     "audio/ogg",
	})

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit {
			t.Fatalf("Kind = %v, want submit", ev.Kind)
		}
		if !strings.Contains(ev.Text, `[The user sent a voice message~ Here's what they said: "please check the Gormes audio path"]`) {
			t.Fatalf("voice text = %q, want Hermes-style transcript before marker", ev.Text)
		}
		if !strings.Contains(ev.Text, "Telegram voice message") {
			t.Fatalf("voice text = %q, want marker preserved", ev.Text)
		}
		if len(ev.Attachments) != 1 || ev.Attachments[0].Kind != "voice" || ev.Attachments[0].Error != "" {
			t.Fatalf("attachments = %+v", ev.Attachments)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_VoiceMessageFallsBackWhenDownloadFails(t *testing.T) {
	mc := newMockClient()
	mc.getFileErr = errTelegramTestDownload
	b := New(Config{
		AllowedChatID:    42,
		AudioTranscriber: fakeAudioTranscriber{Transcript: "should not run"},
	}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.pushVoiceUpdate(42, tgbotapi.Voice{FileID: "voice-file-id", Duration: 9, MimeType: "audio/ogg"})

	select {
	case ev := <-inbox:
		if !strings.Contains(ev.Text, "Telegram voice message") {
			t.Fatalf("voice text = %q, want marker fallback", ev.Text)
		}
		if strings.Contains(ev.Text, "voice-file-id") || strings.Contains(ev.Text, "bot") || strings.Contains(ev.Text, "token") {
			t.Fatalf("voice text leaks transport details: %q", ev.Text)
		}
		submit := ev.SubmitText()
		for _, forbidden := range []string{"voice-file-id", "sourceId=", ".cache/gormes", "~/.cache/gormes"} {
			if strings.Contains(submit, forbidden) {
				t.Fatalf("SubmitText() leaks transport/cache detail %q: %q", forbidden, submit)
			}
		}
		if !strings.Contains(submit, "telegram getFile failed") {
			t.Fatalf("SubmitText() = %q, want sanitized transcription failure marker", submit)
		}
		if len(ev.Attachments) != 1 || ev.Attachments[0].Error == "" {
			t.Fatalf("attachments = %+v, want sanitized error", ev.Attachments)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_AudioMessageWithCaptionPreservesCaptionAndMarker(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.pushAudioUpdate(42, "please transcribe", tgbotapi.Audio{
		FileID:       "audio-file-id",
		FileUniqueID: "audio-unique-id",
		Duration:     12,
		MimeType:     "audio/mpeg",
		FileName:     "sample.mp3",
	})

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit {
			t.Fatalf("Kind = %v, want submit", ev.Kind)
		}
		if !strings.Contains(ev.Text, "please transcribe") || !strings.Contains(ev.Text, "Telegram audio message") {
			t.Fatalf("audio text = %q", ev.Text)
		}
		if len(ev.Attachments) != 1 || ev.Attachments[0].Kind != "audio" || ev.Attachments[0].FileName != "sample.mp3" {
			t.Fatalf("attachments = %+v", ev.Attachments)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_Commands(t *testing.T) {
	cases := []struct {
		text string
		want gateway.EventKind
	}{
		{"/help", gateway.EventStart},
		{"/start", gateway.EventStart},
		{"/stop", gateway.EventCancel},
		{"/new", gateway.EventReset},
		{"/gibberish", gateway.EventUnknown},
		{"plain text", gateway.EventSubmit},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			mc := newMockClient()
			b := New(Config{AllowedChatID: 42}, mc, nil)
			inbox := make(chan gateway.InboundEvent, 1)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = b.Run(ctx, inbox) }()

			mc.pushTextUpdate(42, c.text)

			select {
			case ev := <-inbox:
				if ev.Kind != c.want {
					t.Errorf("text=%q got Kind=%v want=%v", c.text, ev.Kind, c.want)
				}
			case <-time.After(200 * time.Millisecond):
				t.Fatal("no inbound event")
			}
		})
	}
}

func TestBot_Send(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	id, err := b.Send(context.Background(), "42", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatalf("empty msg ID")
	}

	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d", len(sent))
	}
	if _, ok := sent[0].(tgbotapi.MessageConfig); !ok {
		t.Errorf("sent type = %T", sent[0])
	}
	if mc.lastSentText() != "hello" {
		t.Errorf("lastSentText = %q", mc.lastSentText())
	}
}

func TestBot_SendReplySetsTelegramReplyToMessageID(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	sender, ok := any(b).(interface {
		SendReply(context.Context, string, string, string) (string, error)
	})
	if !ok {
		t.Fatal("Bot does not expose SendReply for gateway reply quoting")
	}
	id, err := sender.SendReply(context.Background(), "42", "99", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty msg ID")
	}

	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent type = %T want MessageConfig", sent[0])
	}
	if msg.ReplyToMessageID != 99 {
		t.Fatalf("ReplyToMessageID = %d, want 99", msg.ReplyToMessageID)
	}
}

func TestBot_SendMediaUsesTelegramVoiceAndAudio(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	voicePath := writeTelegramTestMedia(t, "voice.ogg")
	voiceID, err := b.SendMedia(context.Background(), "42", "99", gateway.OutboundMedia{
		Path:    voicePath,
		AsVoice: true,
	})
	if err != nil {
		t.Fatalf("SendMedia voice: %v", err)
	}
	if voiceID == "" {
		t.Fatal("voice msg ID empty")
	}

	audioPath := writeTelegramTestMedia(t, "audio.mp3")
	if _, err := b.SendMedia(context.Background(), "42", "", gateway.OutboundMedia{Path: audioPath}); err != nil {
		t.Fatalf("SendMedia audio: %v", err)
	}

	sent := mc.sentMessages()
	if len(sent) != 2 {
		t.Fatalf("sent count = %d, want 2", len(sent))
	}
	voice, ok := sent[0].(tgbotapi.VoiceConfig)
	if !ok {
		t.Fatalf("sent[0] type = %T, want VoiceConfig", sent[0])
	}
	if voice.ReplyToMessageID != 99 {
		t.Fatalf("voice ReplyToMessageID = %d, want 99", voice.ReplyToMessageID)
	}
	if _, ok := voice.File.(tgbotapi.FilePath); !ok {
		t.Fatalf("voice file = %T, want FilePath", voice.File)
	}
	if _, ok := sent[1].(tgbotapi.AudioConfig); !ok {
		t.Fatalf("sent[1] type = %T, want AudioConfig", sent[1])
	}
}

func TestBot_SendReplyThreadsBrowserArtifactMarkdown(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	body := gateway.FormatBrowserArtifactTelegram(tools.BrowserResultEnvelope{
		State: tools.BrowserPageState{Title: "Browser [Docs]", ScreenshotPath: "[browser_artifact_path_redacted]"},
		Tool:  tools.ToolResultEvidence{Artifact: "browser/snapshot.txt", Bytes: 512},
	})

	if _, err := b.SendReply(context.Background(), "42", "77", body); err != nil {
		t.Fatal(err)
	}
	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent type = %T, want MessageConfig", sent[0])
	}
	if msg.ReplyToMessageID != 77 {
		t.Fatalf("ReplyToMessageID = %d, want 77", msg.ReplyToMessageID)
	}
	if msg.ParseMode != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("ParseMode = %q, want MarkdownV2", msg.ParseMode)
	}
	if strings.Contains(msg.Text, "/tmp/") || strings.Contains(msg.Text, "[Docs]") {
		t.Fatalf("browser artifact reply not sanitized/escaped: %q", msg.Text)
	}
}

func TestBot_SendPlaceholder(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	id, err := b.SendPlaceholder(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatalf("placeholder id empty")
	}
	if !strings.Contains(mc.lastSentText(), "⏳") {
		t.Errorf("placeholder text = %q", mc.lastSentText())
	}
}

func writeTelegramTestMedia(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBot_EditMessage(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)

	if err := b.EditMessage(context.Background(), "42", "1234", "updated"); err != nil {
		t.Fatal(err)
	}

	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d", len(sent))
	}
	if _, ok := sent[0].(tgbotapi.EditMessageTextConfig); !ok {
		t.Errorf("sent type = %T want EditMessageTextConfig", sent[0])
	}
	if mc.lastSentText() != "updated" {
		t.Errorf("edit text = %q", mc.lastSentText())
	}
}

func TestBot_EditMessage_BadMsgID(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	if err := b.EditMessage(context.Background(), "42", "nope", "x"); err == nil {
		t.Fatal("expected error for non-numeric msgID")
	}
}

func TestBot_DeleteMessage(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	if _, ok := any(b).(gateway.MessageDeleter); !ok {
		t.Fatal("Bot does not implement gateway.MessageDeleter")
	}

	if err := b.DeleteMessage(context.Background(), "42", "1234"); err != nil {
		t.Fatal(err)
	}

	deletes := mc.deleteRequests()
	if len(deletes) != 1 {
		t.Fatalf("delete request count = %d, want 1", len(deletes))
	}
	req, ok := deletes[0].(tgbotapi.DeleteMessageConfig)
	if !ok {
		t.Fatalf("delete request type = %T, want tgbotapi.DeleteMessageConfig", deletes[0])
	}
	if req.ChatID != 42 || req.MessageID != 1234 {
		t.Fatalf("delete request = %+v, want chat_id=42 message_id=1234", req)
	}

	if err := b.DeleteMessage(context.Background(), "nope", "1234"); err == nil || !strings.Contains(err.Error(), "invalid chat ID") {
		t.Fatalf("bad chat id error = %v, want invalid chat ID", err)
	}
	if err := b.DeleteMessage(context.Background(), "42", "nope"); err == nil || !strings.Contains(err.Error(), "invalid msgID") {
		t.Fatalf("bad msg id error = %v, want invalid msgID", err)
	}
}

// TestTelegramBot_StatusReplyThreadsToTriggeringMessage proves the Telegram
// adapter captures the inbound MessageID for /status and threads it through
// to a SendReply call so the outbound Telegram API invocation sets
// ReplyToMessageID to the same id, matching Sidon/Hermes reply quoting.
func TestTelegramBot_StatusReplyThreadsToTriggeringMessage(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	// Push a /status update with a specific MessageID; the bot must surface
	// that ID on the InboundEvent so the gateway can thread it back as the
	// ReplyToMessageID on the outbound status response.
	mc.updatesCh <- tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			MessageID: 42,
			Text:      "/status",
			Chat:      &tgbotapi.Chat{ID: 42},
			From:      &tgbotapi.User{ID: 7, FirstName: "juan"},
		},
	}

	var ev gateway.InboundEvent
	select {
	case ev = <-inbox:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
	if ev.Kind != gateway.EventStatus {
		t.Fatalf("Kind = %v, want EventStatus", ev.Kind)
	}
	if ev.MsgID != "42" {
		t.Fatalf("ev.MsgID = %q, want \"42\"", ev.MsgID)
	}
	if ev.MessageID != "42" {
		t.Fatalf("ev.MessageID = %q, want \"42\"", ev.MessageID)
	}

	// Now exercise SendReply with the captured MsgID; assert the outbound
	// MessageConfig carries ReplyToMessageID=42.
	if _, err := b.SendReply(context.Background(), ev.ChatID, ev.MsgID, "status body"); err != nil {
		t.Fatal(err)
	}
	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	msg, ok := sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("sent type = %T, want MessageConfig", sent[0])
	}
	if msg.ReplyToMessageID != 42 {
		t.Fatalf("ReplyToMessageID = %d, want 42", msg.ReplyToMessageID)
	}
	if msg.ParseMode != tgbotapi.ModeMarkdownV2 {
		t.Fatalf("ParseMode = %q, want MarkdownV2 so the bold status labels render", msg.ParseMode)
	}
}

func TestBot_NilMessage_Skipped(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.updatesCh <- tgbotapi.Update{UpdateID: 7}

	select {
	case ev := <-inbox:
		t.Fatalf("expected no inbound, got %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}
