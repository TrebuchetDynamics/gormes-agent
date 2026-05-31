package model

func rememberedSourceID(entry RememberedSourceEntry) string {
	chatID := trimText(entry.ChatID)
	if chatID == "" {
		return ""
	}
	if threadID := trimText(entry.ThreadID); threadID != "" {
		return chatID + ":" + threadID
	}
	return chatID
}

func rememberedSourceName(entry RememberedSourceEntry) string {
	base := trimText(entry.ChatName)
	if base == "" {
		base = trimText(entry.UserName)
	}
	if base == "" {
		base = trimText(entry.ChatID)
	}
	if threadID := trimText(entry.ThreadID); threadID != "" {
		topic := trimText(entry.ChatTopic)
		if topic == "" {
			topic = "topic " + threadID
		}
		return base + " / " + topic
	}
	return base
}

func normalizedSourceChatType(source Source) string {
	if chatType := trimText(source.ChatType); chatType != "" {
		return chatType
	}
	if trimText(source.ThreadID) != "" {
		return "thread"
	}
	return "dm"
}
