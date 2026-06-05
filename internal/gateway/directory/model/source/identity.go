package source

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/policy"

func rememberedSourceID(entry RememberedSourceEntry) string {
	chatID := policy.TrimText(entry.ChatID)
	if chatID == "" {
		return ""
	}
	if threadID := policy.TrimText(entry.ThreadID); threadID != "" {
		return chatID + ":" + threadID
	}
	return chatID
}

func rememberedSourceName(entry RememberedSourceEntry) string {
	base := policy.TrimText(entry.ChatName)
	if base == "" {
		base = policy.TrimText(entry.UserName)
	}
	if base == "" {
		base = policy.TrimText(entry.ChatID)
	}
	if threadID := policy.TrimText(entry.ThreadID); threadID != "" {
		topic := policy.TrimText(entry.ChatTopic)
		if topic == "" {
			topic = "topic " + threadID
		}
		return base + " / " + topic
	}
	return base
}

func normalizedSourceChatType(source Source) string {
	if chatType := policy.TrimText(source.ChatType); chatType != "" {
		return chatType
	}
	if policy.TrimText(source.ThreadID) != "" {
		return "thread"
	}
	return "dm"
}
