package tui

import (
	"fmt"
	"strings"
)

func skinSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "skin: TUI unavailable"}
	}
	name := parseSkinSlashName(input)
	if model.skinConfig == nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "skin: configuration unavailable"}
	}
	result, err := model.skinConfig(SkinConfigRequest{Name: name, SessionID: model.SessionID()})
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "skin: " + err.Error()}
	}
	if name == "" {
		line := "skin: " + skinDisplayName(result.Name)
		model.transientPage = &TransientPageState{Title: "Skin", Body: line}
		return SlashResult{Handled: true, StatusMessage: line}
	}
	accepted := strings.TrimSpace(result.Name)
	if accepted == "" {
		accepted = name
	}
	accepted, err = model.applySkinName(accepted)
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "skin: " + err.Error()}
	}
	line := fmt.Sprintf("skin → %s", accepted)
	model.transientPage = &TransientPageState{Title: "Skin", Body: line}
	return SlashResult{Handled: true, StatusMessage: line}
}

func parseSkinSlashName(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	_, rest, ok := strings.Cut(trimmed, " ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(rest)
}

func skinDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}

func (m *Model) applySkinName(name string) (string, error) {
	skin, ok := ResolveBuiltinSkin(name)
	if !ok {
		return "", fmt.Errorf("unknown skin %q", strings.TrimSpace(name))
	}
	prompt, _ := skin.PromptSymbols("default")
	m.activeSkinName = skin.Name
	m.activeSkin = skin
	m.editor.Prompt = prompt
	return skin.Name, nil
}

func (m Model) currentSkin() HermesSkin {
	if strings.TrimSpace(m.activeSkin.Name) != "" {
		return m.activeSkin
	}
	return DefaultHermesSkin()
}
