package mattermost

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/threadtext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func postedEventPayload(postID, channelID, channelType, userID, message, rootID string) string {
	// Mattermost WebSocket delivers the post field as a JSON string
	// (double-encoded) inside the data object.
	post := `{"id":"` + postID + `","user_id":"` + userID + `","message":"` + message + `","channel_id":"` + channelID + `"}`
	if rootID != "" {
		post = `{"id":"` + postID + `","user_id":"` + userID + `","message":"` + message + `","channel_id":"` + channelID + `","root_id":"` + rootID + `"}`
	}
	return `{"event":"posted","data":{"post":"` + escapeJSON(post) + `","channel_type":"` + channelType + `","channel_id":"` + channelID + `","sender_name":"TestUser","post_id":"` + postID + `"}}`
}

func postedEventWithType(postID, channelID, channelType, userID, message, postType string) string {
	post := `{"id":"` + postID + `","user_id":"` + userID + `","message":"` + message + `","channel_id":"` + channelID + `","type":"` + postType + `"}`
	return `{"event":"posted","data":{"post":"` + escapeJSON(post) + `","channel_type":"` + channelType + `","channel_id":"` + channelID + `","sender_name":"TestUser","post_id":"` + postID + `"}}`
}

// escapeJSON escapes a string for embedding in a JSON string value.
func escapeJSON(s string) string {
	result := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		switch b {
		case '\\':
			result = append(result, '\\', '\\')
		case '"':
			result = append(result, '\\', '"')
		default:
			result = append(result, b)
		}
	}
	return string(result)
}

func TestMattermostSeamParsePostedEvent(t *testing.T) {
	raw := postedEventPayload("post-1", "town-square", "O", "user-1", "hello world", "thread-1")

	seam := NewSeam(threadtext.ReplyModeThread, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	ev, ok := seam.ParsePostedEvent(raw)
	if !ok {
		t.Fatal("ParsePostedEvent ok=false, want true")
	}

	if ev.ChatID != "town-square" {
		t.Fatalf("ChatID=%q, want town-square", ev.ChatID)
	}
	if ev.UserID != "user-1" {
		t.Fatalf("UserID=%q, want user-1", ev.UserID)
	}
	if ev.Kind != gateway.EventSubmit {
		t.Fatalf("Kind=%v, want EventSubmit", ev.Kind)
	}
	if ev.Text != "hello world" {
		t.Fatalf("Text=%q, want hello world", ev.Text)
	}
	if ev.ThreadID != "thread-1" {
		t.Fatalf("ThreadID=%q, want thread-1", ev.ThreadID)
	}
	if ev.Platform != "mattermost" {
		t.Fatalf("Platform=%q, want mattermost", ev.Platform)
	}
}

func TestMattermostSeamDropsSelfPost(t *testing.T) {
	raw := postedEventPayload("post-1", "town-square", "O", "bot-123", "self message", "")

	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	_, ok := seam.ParsePostedEvent(raw)
	if ok {
		t.Fatal("ParsePostedEvent ok=true for self post, want false")
	}
}

func TestMattermostSeamDropsSystemPost(t *testing.T) {
	raw := postedEventWithType("post-2", "town-square", "O", "user-1", "system event", "system_join_channel")

	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	_, ok := seam.ParsePostedEvent(raw)
	if ok {
		t.Fatal("ParsePostedEvent ok=true for system post, want false")
	}
}

func TestMattermostSeamDropsDuplicatePost(t *testing.T) {
	raw := postedEventPayload("post-3", "town-square", "O", "user-1", "first message", "")

	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	ev1, ok := seam.ParsePostedEvent(raw)
	if !ok {
		t.Fatal("first ParsePostedEvent ok=false, want true")
	}
	if ev1.MsgID != "post-3" {
		t.Fatalf("first MsgID=%q, want post-3", ev1.MsgID)
	}

	ev2, ok := seam.ParsePostedEvent(raw)
	if ok {
		t.Fatalf("second ParsePostedEvent ok=true for duplicate, want false (got %+v)", ev2)
	}
}

func TestMattermostSeamDropsMalformedJSON(t *testing.T) {
	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	_, ok := seam.ParsePostedEvent("not json")
	if ok {
		t.Fatal("ParsePostedEvent ok=true for malformed JSON, want false")
	}

	_, ok = seam.ParsePostedEvent(`{"event":"posted"}`)
	if ok {
		t.Fatal("ParsePostedEvent ok=true for event with no data, want false")
	}
}

func TestMattermostSeamDropsNonPostedEvent(t *testing.T) {
	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	nonPosted := `{"event":"typing","data":{"post":"{\"id\":\"p1\"}"}}`
	_, ok := seam.ParsePostedEvent(nonPosted)
	if ok {
		t.Fatal("ParsePostedEvent ok=true for typing event, want false")
	}
}

func TestMattermostSeamReplyModeThreadSetsRootID(t *testing.T) {
	raw := postedEventPayload("post-5", "town-square", "O", "user-1", "hello", "")

	seam := NewSeam(threadtext.ReplyModeThread, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	ev, ok := seam.ParsePostedEvent(raw)
	if !ok {
		t.Fatal("ParsePostedEvent ok=false, want true")
	}

	msg := threadtext.InboundMessage{
		ChatID:    ev.ChatID,
		UserID:    ev.UserID,
		MessageID: ev.MsgID,
		Text:      ev.Text,
	}

	target, ok2 := seam.ResolveReplyTarget(msg)
	if !ok2 {
		t.Fatal("ResolveReplyTarget ok=false, want true")
	}
	if target.ThreadID != ev.MsgID {
		t.Fatalf("reply_mode=thread: ThreadID=%q, want %q (msgID as root)", target.ThreadID, ev.MsgID)
	}
	if target.ReplyToMessageID != ev.MsgID {
		t.Fatalf("reply_mode=thread: ReplyToMessageID=%q, want %q", target.ReplyToMessageID, ev.MsgID)
	}
}

func TestMattermostSeamReplyModeOffOmitsRootID(t *testing.T) {
	raw := postedEventPayload("post-6", "town-square", "O", "user-1", "hello", "")

	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	ev, ok := seam.ParsePostedEvent(raw)
	if !ok {
		t.Fatal("ParsePostedEvent ok=false, want true")
	}

	msg := threadtext.InboundMessage{
		ChatID:    ev.ChatID,
		UserID:    ev.UserID,
		MessageID: ev.MsgID,
		Text:      ev.Text,
	}

	target, ok2 := seam.ResolveReplyTarget(msg)
	if !ok2 {
		t.Fatal("ResolveReplyTarget ok=false, want true")
	}
	if target.ThreadID != "" {
		t.Fatalf("reply_mode=off: ThreadID=%q, want empty", target.ThreadID)
	}
}

func TestMattermostSeamMentionGatingDM(t *testing.T) {
	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindGated,
	}, "bot-123", nil)

	// DM channels bypass mention gating
	raw := postedEventPayload("post-7", "dm-channel", "D", "user-1", "no mention needed", "")
	ev, ok := seam.ParsePostedEvent(raw)
	if !ok {
		t.Fatal("ParsePostedEvent ok=false for DM, want true (DMs bypass gating)")
	}
	_ = ev
}

func TestMattermostSeamMentionGatingRequiresMention(t *testing.T) {
	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindGated,
	}, "bot-123", nil)

	// Non-DM channel without @mention should be dropped
	raw := postedEventPayload("post-8", "town-square", "O", "user-1", "no mention", "")
	_, ok := seam.ParsePostedEvent(raw)
	if ok {
		t.Fatal("ParsePostedEvent ok=true for non-DM without mention, want false")
	}

	// With @mention, should pass
	rawWithMention := postedEventPayload("post-9", "town-square", "O", "user-1", "@bot-123 help me", "")
	ev, ok2 := seam.ParsePostedEvent(rawWithMention)
	if !ok2 {
		t.Fatal("ParsePostedEvent ok=false for non-DM with mention, want true")
	}
	// @mention should be stripped
	if ev.Text == "@bot-123 help me" {
		t.Fatalf("Text=%q, @mention should be stripped", ev.Text)
	}
}

func TestMattermostSeamMentionGatingFreeChannel(t *testing.T) {
	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind:           KindGated,
		FreeChannelIDs: map[string]bool{"lounge": true},
	}, "bot-123", nil)

	// Free channel bypasses mention gating
	raw := postedEventPayload("post-10", "lounge", "O", "user-1", "no mention needed in free channel", "")
	ev, ok := seam.ParsePostedEvent(raw)
	if !ok {
		t.Fatal("ParsePostedEvent ok=false for free channel without mention, want true")
	}
	_ = ev
}

func TestMattermostSeamExistingThreadPreservesThreadID(t *testing.T) {
	// When a message is a reply in an existing thread, root_id is the thread ID
	raw := postedEventPayload("post-11", "town-square", "O", "user-1", "thread reply", "existing-thread-1")

	seam := NewSeam(threadtext.ReplyModeThread, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	ev, ok := seam.ParsePostedEvent(raw)
	if !ok {
		t.Fatal("ParsePostedEvent ok=false, want true")
	}

	if ev.ThreadID != "existing-thread-1" {
		t.Fatalf("ThreadID=%q, want existing-thread-1 (root_id)", ev.ThreadID)
	}

	msg := threadtext.InboundMessage{
		ChatID:       ev.ChatID,
		UserID:       ev.UserID,
		MessageID:    ev.MsgID,
		Text:         ev.Text,
		ThreadRootID: "existing-thread-1",
	}

	target, ok2 := seam.ResolveReplyTarget(msg)
	if !ok2 {
		t.Fatal("ResolveReplyTarget ok=false, want true")
	}
	if target.ThreadID != "existing-thread-1" {
		t.Fatalf("ThreadID=%q, want existing-thread-1", target.ThreadID)
	}
}

func TestMattermostSeamAllowsCustomChannel(t *testing.T) {
	// Test that allowed channels whitelist works
	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind:            KindGated,
		AllowedChannels: map[string]bool{"ops-room": true},
	}, "bot-123", nil)

	// Allowed channel with mention passes
	rawAllowed := postedEventPayload("post-12", "ops-room", "O", "user-1", "@bot-123 hello", "")
	ev, ok := seam.ParsePostedEvent(rawAllowed)
	if !ok {
		t.Fatal("ParsePostedEvent ok=false for allowed channel with mention, want true")
	}
	_ = ev

	// Non-allowed channel should be dropped even with mention
	rawNotAllowed := postedEventPayload("post-13", "town-square", "O", "user-1", "@bot-123 hello", "")
	_, ok2 := seam.ParsePostedEvent(rawNotAllowed)
	if ok2 {
		t.Fatal("ParsePostedEvent ok=true for non-allowed channel, want false")
	}
}

func TestMattermostSeamCancelSuppressesHooks(t *testing.T) {
	startCalled := false
	completeCalled := false
	cancelCalled := false

	var hooks = &ProcessingHooks{
		OnStart: func(_ threadtext.InboundMessage) { startCalled = true },
		OnComplete: func(_ threadtext.InboundMessage) {
			completeCalled = true
		},
		OnCancel: func(_ threadtext.InboundMessage) { cancelCalled = true },
	}

	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", hooks)

	// Cancel before processing
	seam.Cancel(threadtext.InboundMessage{})
	if !seam.Cancelled() {
		t.Fatal("Cancelled()=false after Cancel() call")
	}
	if !cancelCalled {
		t.Fatal("OnCancel hook not called")
	}

	// After cancel, OnComplete should not fire
	if completeCalled {
		t.Fatal("OnComplete fired after cancel")
	}

	// start not called either (no message processed)
	if startCalled {
		t.Fatal("OnStart fired but no message was processed")
	}
}

func TestMattermostSeamNormalizeInbound(t *testing.T) {
	raw := postedEventPayload("post-14", "town-square", "O", "user-1", "/start hi", "")

	seam := NewSeam(threadtext.ReplyModeFlat, MentionGatingInputs{
		Kind: KindFree,
	}, "bot-123", nil)

	ev, ok := seam.ParsePostedEvent(raw)
	if !ok {
		t.Fatal("ParsePostedEvent ok=false, want true")
	}

	// /start should be parsed as a command
	if ev.Kind != gateway.EventStart {
		t.Fatalf("Kind=%v, want EventStart for /start", ev.Kind)
	}
}
