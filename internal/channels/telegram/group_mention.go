package telegram

import (
	"strings"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func telegramGroupMentionGateAddressed(text string, entities []tgbotapi.MessageEntity, expectedBotUsername string, requireMention bool) bool {
	if !requireMention {
		return true
	}

	expected := normalizeTelegramBotUsername(expectedBotUsername)
	if expected == "" {
		return false
	}

	for _, entity := range entities {
		entityText, ok := telegramEntityText(text, entity)
		if !ok {
			continue
		}

		switch entity.Type {
		case "mention":
			if strings.EqualFold(strings.TrimSpace(entityText), "@"+expected) {
				return true
			}
		case "bot_command":
			at := strings.LastIndex(entityText, "@")
			if at < 0 {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(entityText[at+1:]), expected) {
				return true
			}
		}
	}

	return false
}

func normalizeTelegramBotUsername(username string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(username)), "@")
}

func telegramEntityText(text string, entity tgbotapi.MessageEntity) (string, bool) {
	if entity.Offset < 0 || entity.Length <= 0 {
		return "", false
	}

	start, end, ok := utf16EntityByteRange(text, entity.Offset, entity.Length)
	if !ok {
		return "", false
	}
	return text[start:end], true
}

func utf16EntityByteRange(text string, offset, length int) (int, int, bool) {
	startUnit := offset
	endUnit := offset + length
	currentUnit := 0
	startByte := -1
	endByte := -1

	for byteIndex, r := range text {
		if currentUnit == startUnit {
			startByte = byteIndex
		}
		if currentUnit == endUnit {
			endByte = byteIndex
			break
		}
		currentUnit += len(utf16.Encode([]rune{r}))
	}

	if currentUnit == startUnit {
		startByte = len(text)
	}
	if currentUnit == endUnit {
		endByte = len(text)
	}
	if startByte < 0 || endByte < 0 || startByte > endByte {
		return 0, 0, false
	}
	return startByte, endByte, true
}
