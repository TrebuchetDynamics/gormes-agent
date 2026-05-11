package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

const (
	modelPickerCBProvider   = "mp"
	modelPickerCBModel      = "mm"
	modelPickerCBGroupPage  = "mg"
	modelPickerCBBack       = "mb"
	modelPickerCBCancel     = "mx"
)

func modelPickerProviderData(slug string) string {
	return fmt.Sprintf("%s:%s", modelPickerCBProvider, slug)
}

func modelPickerModelData(index int) string {
	return fmt.Sprintf("%s:%d", modelPickerCBModel, index)
}

func modelPickerGroupPageData(page int) string {
	return fmt.Sprintf("%s:%d", modelPickerCBGroupPage, page)
}

func parseModelPickerCallback(data string) (prefix, value string, ok bool) {
	data = strings.TrimSpace(data)
	idx := strings.IndexByte(data, ':')
	if idx < 0 {
		return "", "", false
	}
	prefix = data[:idx]
	switch prefix {
	case modelPickerCBProvider, modelPickerCBModel, modelPickerCBGroupPage, modelPickerCBBack, modelPickerCBCancel:
		return prefix, data[idx+1:], true
	}
	return "", "", false
}

func buildProviderKeyboard(providers []hermes.PickerProvider) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, p := range providers {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(p.Label, modelPickerProviderData(p.Slug)))
		if len(row) == 2 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Cancel", modelPickerCBCancel+":"),
	))
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func buildModelKeyboard(models []string, page, modelsPerPage int) tgbotapi.InlineKeyboardMarkup {
	start := page * modelsPerPage
	if start >= len(models) {
		start = 0
	}
	end := start + modelsPerPage
	if end > len(models) {
		end = len(models)
	}
	pageModels := models[start:end]

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, m := range pageModels {
		idx := start + i
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(m, modelPickerModelData(idx)),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("◀ Prev", modelPickerGroupPageData(page-1)))
	}
	if end < len(models) {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Next ▶", modelPickerGroupPageData(page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Back", modelPickerCBBack+":"),
		tgbotapi.NewInlineKeyboardButtonData("Cancel", modelPickerCBCancel+":"),
	))
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func buildModelPickerProviderText() string {
	return "⚙ *Model Configuration*\n\n*Select a provider:*"
}

func buildModelPickerModelText(providerLabel string) string {
	return fmt.Sprintf("⚙ *Model Configuration*\n\nProvider: *%s*\n\n*Select a model:*", providerLabel)
}

func buildModelPickerConfirmationText(model, provider string) string {
	provSlug := strings.Title(strings.NewReplacer("-", " ", "_", " ").Replace(provider))
	return fmt.Sprintf("⚙ *Model Configuration*\n\nModel set to `%s`\nProvider: *%s*", model, provSlug)
}

func modelPickerModelIndex(v string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(v))
}
