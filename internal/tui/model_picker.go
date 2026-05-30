// Package tui — Hermes-compatible ModelPicker overlay renderer and updater.
//
// ModelPickerState + RenderModelPicker implement the 2-step provider→model
// selection overlay that upstream Hermes exposes via modelPicker.tsx. The Go
// port is a pure renderer pair: state in → string out for rendering, and
// UpdateModelPicker for keyboard navigation. Neither function allocates
// goroutines or reads wall clocks.
//
// Provider column layout matches the upstream 2-per-row grid. The model
// column appears only after a provider is selected and shows a scrolling
// list with the current model marked by "*".
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker"
)

// ProviderEntry is one provider option in the picker.
type ProviderEntry = modelpicker.ProviderEntry

// ModelEntry is one model option shown after a provider is selected.
type ModelEntry = modelpicker.ModelEntry

// ModelPickerState is the complete state for the model picker overlay.
// Width and Height carry the terminal dimensions for layout calculations.
type ModelPickerState = modelpicker.State

// ModelPickerResult is returned when the user confirms a model selection.
// It carries the chosen provider and model IDs.
type ModelPickerResult = modelpicker.Result

// modelPickerConfirmedMsg is the internal Bubble Tea message emitted when
// the user confirms a model selection.
type modelPickerConfirmedMsg ModelPickerResult

// RenderModelPicker renders the model picker overlay as a string.
func RenderModelPicker(state ModelPickerState) string {
	return RenderModelPickerWithSkin(state, DefaultHermesSkin())
}

func RenderModelPickerWithSkin(state ModelPickerState, skin HermesSkin) string {
	styles := SkinStylesFor(skin)
	return modelpicker.Render(state, modelpicker.Styles{
		ActivePill: styles.ActivePill,
		Label:      styles.Label,
		Selected:   styles.Selected,
		Normal:     styles.Normal,
		Good:       styles.Good,
		Dim:        styles.Dim,
	})
}

// UpdateModelPicker handles keyboard events for the model picker overlay.
// It returns the updated state and an optional Bubble Tea command to execute.
func UpdateModelPicker(msg tea.Msg, state ModelPickerState) (ModelPickerState, tea.Cmd) {
	state, result, emit := modelpicker.Update(msg, state)
	if !emit {
		return state, nil
	}
	return state, func() tea.Msg { return modelPickerConfirmedMsg(result) }
}
