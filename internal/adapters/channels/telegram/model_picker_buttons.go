package telegram

import telegramcallbacks "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/callbacks"

func parseModelPickerCallback(data string) (prefix, value string, ok bool) {
	return telegramcallbacks.ParseModelPickerCallback(data)
}
