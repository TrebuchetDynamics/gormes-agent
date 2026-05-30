package directory

import "testing"

func TestRememberedSourceEntryFormatsHermesSessionDirectoryFields(t *testing.T) {
	telegram := RememberedSourceEntryFromSource(Source{
		Platform:  "telegram",
		ChatID:    " -1001 ",
		ChatName:  " Coaching Chat ",
		ChatType:  "group",
		UserID:    " 77 ",
		UserName:  " Juan ",
		ThreadID:  "17585",
		MessageID: "msg-9",
	})
	if telegram.ID != "-1001:17585" {
		t.Fatalf("telegram ID = %q, want composite chat_id:thread_id", telegram.ID)
	}
	if telegram.Name != "Coaching Chat / topic 17585" {
		t.Fatalf("telegram Name = %q, want Hermes topic fallback", telegram.Name)
	}
	if telegram.Type != "group" || telegram.ChatID != "-1001" || telegram.ThreadID != "17585" || telegram.UserID != "77" || telegram.UserName != "Juan" || telegram.MessageID != "msg-9" {
		t.Fatalf("telegram entry = %+v, want trimmed metadata preserved", telegram)
	}

	discord := RememberedSourceEntryFromSource(Source{
		Platform:     "discord",
		ChatID:       "thread-2",
		ChatName:     "triage",
		ChatType:     "thread",
		GuildID:      "guild-1",
		ParentChatID: "forum-9",
		ThreadID:     "thread-2",
	})
	if discord.ID != "thread-2:thread-2" || discord.GuildID != "guild-1" || discord.ParentChatID != "forum-9" {
		t.Fatalf("discord entry = %+v, want guild and parent metadata preserved", discord)
	}
	if got := discord.ChannelDirectoryEntry(); got.Guild != "guild-1" || got.ChatID != "thread-2" || got.ThreadID != "thread-2" || got.Type != "thread" {
		t.Fatalf("ChannelDirectoryEntry = %+v, want Discord directory metadata", got)
	}
}
