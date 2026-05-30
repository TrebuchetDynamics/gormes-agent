package tui

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
)

func skinSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "skin: TUI unavailable"}
	}
	var configure skin.ConfigFunc
	if model.skinConfig != nil {
		configure = skin.ConfigFunc(model.skinConfig)
	}
	result := skin.HandleSlash(input, model.SessionID(), configure)
	if result.Err != nil || !result.Apply {
		if result.Body != "" {
			model.transientPage = &TransientPageState{Title: "Skin", Body: result.Body}
		} else {
			model.transientPage = nil
		}
		return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
	}
	accepted, err := model.applySkinName(result.AcceptedName)
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "skin: " + err.Error()}
	}
	line := fmt.Sprintf("skin → %s", accepted)
	model.transientPage = &TransientPageState{Title: "Skin", Body: line}
	return SlashResult{Handled: true, StatusMessage: line}
}

func parseSkinSlashName(input string) string {
	return skin.SlashName(input)
}

func skinDisplayName(name string) string {
	return skin.DisplayName(name)
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
	ApplyTextareaSkin(&m.editor, skin)
	return skin.Name, nil
}

func (m Model) currentSkin() HermesSkin {
	if strings.TrimSpace(m.activeSkin.Name) != "" {
		return m.activeSkin
	}
	return DefaultHermesSkin()
}
