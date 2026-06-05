// Package tui — panel wiring connects kernel RenderFrame panel state to the
// hermes_panels.go renderers.
//
// This file provides:
//   - Extraction helpers that project panel state from kernel.RenderFrame
//     onto the TUI Model fields.
//   - A rendering helper that calls the correct hermes_panels renderer based
//     on which panel state is active.
//   - Keybinding integration: when a panel is active, key events route to
//     HermesActionCancelModal instead of the normal turn handling.
//
// The wiring is intentionally one-way: kernel → Model → View. The TUI never
// mutates kernel panel state; user choices are sent back via the normal
// PlatformEventSubmit path with a prefixed choice marker (e.g. "/approve 1").
package tui

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// ExtractPanelStateFromFrame updates the receiver Model's panel state fields
// to reflect the current RenderFrame. It is called by Update() on every
// frameMsg so that View() always renders from a consistent snapshot.
//
// Priority order when multiple panels are set: Approval > Clarify > Secret.
// This mirrors the upstream Hermes ui-tui priority in
// ../hermes-agent/ui-tui/render.py.
func (m *Model) ExtractPanelStateFromFrame(f kernel.RenderFrame) {
	m.ApprovalState = f.ApprovalState
	m.ClarifyState = f.ClarifyState
	m.SecretState = f.SecretState
}

// ActivePanelKind identifies which panel is currently active, if any.
type ActivePanelKind int

const (
	ActivePanelNone ActivePanelKind = iota
	ActivePanelApproval
	ActivePanelClarify
	ActivePanelSecret
)

// ActivePanel returns which panel is currently active and the corresponding
// state pointer. When no panel is active, it returns ActivePanelNone with
// nil state.
func (m *Model) ActivePanel() (ActivePanelKind, interface{}) {
	if m.ApprovalState != nil {
		return ActivePanelApproval, m.ApprovalState
	}
	if m.ClarifyState != nil {
		return ActivePanelClarify, m.ClarifyState
	}
	if m.SecretState != nil {
		return ActivePanelSecret, m.SecretState
	}
	return ActivePanelNone, nil
}

// IsPanelActive reports whether any modal panel is currently displayed.
// This is used by the Hermes keybinding resolver to route Ctrl+C to
// HermesActionCancelModal instead of the normal turn interrupt path.
func (m *Model) IsPanelActive() bool {
	return m.ApprovalState != nil || m.ClarifyState != nil || m.SecretState != nil
}

// RenderActivePanel returns the rendered string for whichever panel is
// currently active, or an empty string if no panel is active. The rendered
// output includes the full panel chrome (box drawing characters).
//
// Width and height hints are taken from the panel state when non-zero;
// otherwise a sensible default is used so the panel always fits.
func (m *Model) RenderActivePanel(width, height int) string {
	kind, state := m.ActivePanel()
	switch kind {
	case ActivePanelApproval:
		aps, ok := state.(*kernel.KernelApprovalState)
		if !ok || aps == nil {
			return ""
		}
		// Map kernel approval choices to hermes_panels ApprovalChoice.
		choices := make([]ApprovalChoice, len(aps.Choices))
		for i, c := range aps.Choices {
			choices[i] = ApprovalChoice(c)
		}
		panelState := ApprovalPanelState{
			Description:  aps.Description,
			Command:      aps.Command,
			Choices:      choices,
			Selected:     ApprovalChoice(aps.Selected),
			ViewExpanded: aps.ViewExpanded,
		}
		if aps.Width > 0 {
			panelState.Width = aps.Width
		} else {
			panelState.Width = width
		}
		if aps.Height > 0 {
			panelState.Height = aps.Height
		} else {
			panelState.Height = height
		}
		return RenderApprovalPanelWithSkin(panelState, m.currentSkin())

	case ActivePanelClarify:
		kcs, ok := state.(*kernel.KernelClarifyState)
		if !ok || kcs == nil {
			return ""
		}
		panelState := ClarifyPanelState{
			Question:    kcs.Question,
			Choices:     kcs.Choices,
			Selected:    kcs.Selected,
			TimeoutHint: kcs.TimeoutHint,
		}
		if kcs.Width > 0 {
			panelState.Width = kcs.Width
		} else {
			panelState.Width = width
		}
		if kcs.Height > 0 {
			panelState.Height = kcs.Height
		} else {
			panelState.Height = height
		}
		return RenderClarifyPanelWithSkin(panelState, m.currentSkin())

	case ActivePanelSecret:
		kss, ok := state.(*kernel.KernelSecretState)
		if !ok || kss == nil {
			return ""
		}
		panelState := SecretPanelState{
			Mode:       SecretPanelMode(kss.Mode),
			PromptText: kss.PromptText,
			Countdown:  kss.Countdown,
			SecretLen:  kss.SecretLen,
			Hint:       kss.Hint,
		}
		return RenderSecretPanelWithSkin(panelState, m.currentSkin())

	default:
		return ""
	}
}

// BuildHermesInputStateForPanel extends the normal HermesInputState with
// panel-aware ModalActive flag. When a panel is active, the keybinding
// resolver routes Ctrl+C to HermesActionCancelModal.
func (m *Model) BuildHermesInputStateForPanel(st HermesInputState) HermesInputState {
	st.ModalActive = m.IsPanelActive()
	return st
}
