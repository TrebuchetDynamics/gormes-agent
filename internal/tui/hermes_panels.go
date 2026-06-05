// Package tui preserves the public Hermes-compatible modal/tool-progress panel
// renderer seam while the pure implementations live in internal/tui/panels.
package tui

import panels "github.com/TrebuchetDynamics/gormes-agent/internal/tui/panels"

// ToolSpinnerState is the state injected by the kernel/tool layer when a
// tool.started event is observed.
type ToolSpinnerState = panels.ToolSpinnerState

func RenderToolSpinner(s ToolSpinnerState) string { return panels.RenderToolSpinner(s) }

type ToolScrollMode = panels.ToolScrollMode

const (
	ToolScrollAll ToolScrollMode = panels.ToolScrollAll
	ToolScrollNew ToolScrollMode = panels.ToolScrollNew
)

type ToolCompletion = panels.ToolCompletion

func RenderToolScrollback(items []ToolCompletion, mode ToolScrollMode) []string {
	return panels.RenderToolScrollback(items, mode)
}

type ApprovalChoice = panels.ApprovalChoice

const (
	ApprovalOnce    ApprovalChoice = panels.ApprovalOnce
	ApprovalSession ApprovalChoice = panels.ApprovalSession
	ApprovalAlways  ApprovalChoice = panels.ApprovalAlways
	ApprovalDeny    ApprovalChoice = panels.ApprovalDeny
	ApprovalView    ApprovalChoice = panels.ApprovalView
)

type ApprovalPanelState = panels.ApprovalPanelState

func RenderApprovalPanel(s ApprovalPanelState) string { return panels.RenderApprovalPanel(s) }

func RenderApprovalPanelWithSkin(s ApprovalPanelState, skin HermesSkin) string {
	return panels.RenderApprovalPanelWithStyles(s, panelStylesForSkin(skin))
}

type ClarifyPanelState = panels.ClarifyPanelState

func RenderClarifyPanel(s ClarifyPanelState) string { return panels.RenderClarifyPanel(s) }

func RenderClarifyPanelWithSkin(s ClarifyPanelState, skin HermesSkin) string {
	return panels.RenderClarifyPanelWithStyles(s, panelStylesForSkin(skin))
}

type SecretPanelMode = panels.SecretPanelMode

const (
	SecretPanelSudo      SecretPanelMode = panels.SecretPanelSudo
	SecretPanelArbitrary SecretPanelMode = panels.SecretPanelArbitrary
)

type SecretPanelState = panels.SecretPanelState

func RenderSecretPanel(s SecretPanelState) string { return panels.RenderSecretPanel(s) }

func RenderSecretPanelWithSkin(s SecretPanelState, skin HermesSkin) string {
	return panels.RenderSecretPanelWithStyles(s, panelStylesForSkin(skin))
}

func panelStylesForSkin(skin HermesSkin) panels.Styles {
	styles := SkinStylesFor(skin)
	return panels.Styles{
		Critical:  styles.Critical,
		Bad:       styles.Bad,
		Normal:    styles.Normal,
		Selected:  styles.Selected,
		Dim:       styles.Dim,
		Title:     styles.Title,
		Text:      styles.Text,
		Prompt:    styles.Prompt,
		Separator: styles.Separator,
		Warn:      styles.Warn,
	}
}
