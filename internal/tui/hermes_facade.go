// Package tui — Hermes-compatible chrome, panels, skin, and status-bar facades.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/banner"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/chrome"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/panels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal"

	tea "github.com/charmbracelet/bubbletea"
)

// ─── Hermes skin re-exports ─────────────────────────────────────────────────

type HermesSkin = skin.HermesSkin
type HermesSkinColors = skin.HermesSkinColors

func DefaultHermesSkin() HermesSkin { return skin.DefaultHermesSkin() }
func DefaultToolEmojis() map[string]string { return skin.DefaultToolEmojis() }
func ResolveBuiltinSkin(name string) (HermesSkin, bool) { return skin.ResolveBuiltinSkin(name) }
func BuiltinSkins() map[string]HermesSkin { return skin.BuiltinSkins() }

// ─── Hermes chrome re-exports ───────────────────────────────────────────────

const hermesMinimalChromeWidth = skin.MinimalChromeWidth

type HermesChromeInput = chrome.Input

func RenderHermesChrome(in HermesChromeInput) string {
	return chrome.Render(in)
}

func trimTrailingLineWhitespace(s string) string {
	return chrome.TrimTrailingLineWhitespace(s)
}

func HermesChromeUseAltScreen() bool {
	return chrome.UseAltScreen()
}

func HermesChromeAssistantLabel() string {
	return skin.DefaultResponseLabel()
}

// ─── Hermes panels re-exports ───────────────────────────────────────────────

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

// ─── Hermes status-bar re-exports ───────────────────────────────────────────

type HermesStatusModel = statusbar.HermesModel
type HermesStatusContextSeverity = statusbar.HermesContextSeverity

const (
	HermesStatusContextDim      HermesStatusContextSeverity = statusbar.HermesContextDim
	HermesStatusContextGood     HermesStatusContextSeverity = statusbar.HermesContextGood
	HermesStatusContextWarn     HermesStatusContextSeverity = statusbar.HermesContextWarn
	HermesStatusContextBad      HermesStatusContextSeverity = statusbar.HermesContextBad
	HermesStatusContextCritical HermesStatusContextSeverity = statusbar.HermesContextCritical
)

func HermesStatusBarContextSeverity(percent *int) HermesStatusContextSeverity {
	return statusbar.HermesContextSeverityFor(percent)
}

func RenderHermesStatusBar(model HermesStatusModel, width int) string {
	return renderHermesStatusBar(model, width)
}

func RenderHermesStatusBarWithSkin(model HermesStatusModel, width int, skin HermesSkin) string {
	line := renderHermesStatusBar(model, width)
	if line == "" {
		return ""
	}
	styles := SkinStylesFor(skin)
	line = styleHermesStatusBarSegments(line, model, width, styles)
	return styles.Status.Width(width).Render(line)
}

func styleHermesStatusBarSegments(line string, model HermesStatusModel, width int, styles SkinStyles) string {
	percent := hermesStatusPercent(model.ContextTokens, model.ContextLength)
	if percent != nil {
		percentLabel := fmt.Sprintf("%d%%", *percent)
		segment := percentLabel
		if width >= 76 {
			segment = fmt.Sprintf("[%s] %s", hermesStatusContextBar(*percent), percentLabel)
		}
		line = strings.Replace(line, segment, hermesStatusContextStyle(styles, percent).Render(segment), 1)
	}
	if model.HasPromptElapsed && width >= 76 {
		elapsed := hermesStatusPromptElapsed(model.PromptElapsed, model.PromptLive)
		line = strings.Replace(line, elapsed, styles.Warn.Background(styles.Status.GetBackground()).Render(elapsed), 1)
	}
	if cwd := strings.TrimSpace(model.CWDLabel); cwd != "" && width >= 76 {
		line = strings.Replace(line, cwd, styles.Dim.Background(styles.Status.GetBackground()).Render(cwd), 1)
	}
	return line
}

func hermesStatusContextStyle(styles SkinStyles, percent *int) lipgloss.Style {
	var style lipgloss.Style
	switch HermesStatusBarContextSeverity(percent) {
	case HermesStatusContextGood:
		style = styles.Good
	case HermesStatusContextWarn:
		style = styles.Warn
	case HermesStatusContextBad:
		style = styles.Bad
	case HermesStatusContextCritical:
		style = styles.Critical
	default:
		style = styles.Dim
	}
	return style.Background(styles.Status.GetBackground())
}

func renderHermesStatusBar(model HermesStatusModel, width int) string {
	return statusbar.RenderHermes(model, width)
}

func hermesStatusModelLabel(name, effort string, fast bool) string {
	return statusbar.HermesModelLabel(name, effort, fast)
}

func hermesStatusContextBar(percent int) string {
	return statusbar.HermesContextBar(percent)
}

func hermesStatusDurationLabel(seconds int64) string {
	return statusbar.HermesDurationLabel(seconds)
}

func hermesStatusPercent(tokens, length int) *int {
	return statusbar.HermesPercent(tokens, length)
}

func hermesStatusFormatTokenCount(value int) string {
	return statusbar.HermesFormatTokenCount(value)
}

func hermesStatusFormatContextLength(tokens int) string {
	return statusbar.HermesFormatContextLength(tokens)
}

func hermesStatusPromptElapsed(seconds int64, live bool) string {
	return statusbar.HermesPromptElapsed(seconds, live)
}

func hermesStatusTrimToWidth(text string, maxWidth int) string {
	return statusbar.HermesTrimToWidth(text, maxWidth)
}

// ─── Hermes status-bar extension types ──────────────────────────────────────

const ContextBarWidth = statusbar.ContextBarWidth

func RenderFaceTicker(state string, frame int) string {
	return statusbar.RenderFaceTicker(state, frame)
}

func RenderContextBar(pct float64) string {
	return statusbar.RenderContextBar(pct)
}

func ContextBarSeverity(pct float64) HermesStatusContextSeverity {
	return statusbar.ContextBarSeverity(pct)
}

func RenderContextBarWithLabel(pct float64) string {
	return statusbar.RenderContextBarWithLabel(pct)
}

// ─── Status bar mode ────────────────────────────────────────────────────────

type StatusBarMode = statusbar.Mode

const (
	StatusBarModeTop    = statusbar.ModeTop
	StatusBarModeBottom = statusbar.ModeBottom
	StatusBarModeOff    = statusbar.ModeOff
)

func normalizeStatusBarMode(mode StatusBarMode) StatusBarMode {
	return statusbar.NormalizeMode(mode)
}

// ─── Banner re-exports ──────────────────────────────────────────────────────

// SetWelcomeContext seeds the welcome panel with the operator-facing release
// version and agent tool count.
func SetWelcomeContext(version string, toolCount int, toolsets ...string) {
	banner.SetWelcomeContext(version, toolCount, toolsets...)
}

// ─── Shared style re-exports ────────────────────────────────────────────────

// SkinStyles is the shared Bubble Tea/Lip Gloss style set for all Gormes TUI
// surfaces. Chat, admin, and setup wizard code should derive local component
// chrome from this seam instead of rebuilding token mappings independently.
type SkinStyles = skin.SkinStyles

// NormalizeStyleSkin returns the default Hermes/Gormes skin when callers pass
// a zero-value skin. Subpackages use this to keep fallback policy in one place.
func NormalizeStyleSkin(s HermesSkin) HermesSkin { return skin.NormalizeStyleSkin(s) }

// SkinStylesFor resolves the shared semantic style set from a skin. It is the
// common source for chat transcript styles, admin shell chrome, setup wizard
// inputs, and future Bubble Tea overlays.
func SkinStylesFor(s HermesSkin) SkinStyles { return skin.SkinStylesFor(s) }

func ApplyTextareaSkin(input *textarea.Model, s HermesSkin) { skin.ApplyTextareaSkin(input, s) }

func ApplyTextInputSkin(input *textinput.Model, s HermesSkin) { skin.ApplyTextInputSkin(input, s) }

// chatStyles holds the lipgloss style for each semantic transcript role,
// derived from the active HermesSkin. view.go renders every role/chrome
// element through these named styles instead of scattered inline
// lipgloss.NewStyle calls.
type chatStyles = skin.ChatStyles

func chatStylesFor(s HermesSkin) chatStyles {
	return skin.ChatStylesFor(s)
}

// ─── Model picker re-exports ────────────────────────────────────────────────

type ProviderEntry = modelpicker.ProviderEntry
type ModelEntry = modelpicker.ModelEntry
type ModelPickerState = modelpicker.State
type ModelPickerResult = modelpicker.Result

type modelPickerConfirmedMsg ModelPickerResult

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

func UpdateModelPicker(msg tea.Msg, state ModelPickerState) (ModelPickerState, tea.Cmd) {
	state, result, emit := modelpicker.Update(msg, state)
	if !emit {
		return state, nil
	}
	return state, func() tea.Msg { return modelPickerConfirmedMsg(result) }
}

func HermesModelPickerProviders() []ProviderEntry {
	catalog := modelpicker.HermesProviders()
	entries := make([]ProviderEntry, 0, len(catalog))
	for _, entry := range catalog {
		entries = append(entries, ProviderEntry{ID: entry.ID, Label: entry.Label})
	}
	return entries
}

// ─── Composer ingress facades ──────────────────────────────────────────────

type ComposerDropOptions = composer.ComposerDropOptions
type ComposerDropResult = composer.ComposerDropResult
type ComposerPasteOptions = composer.ComposerPasteOptions
type ComposerPasteSnippet = composer.ComposerPasteSnippet
type ComposerPasteResult = composer.ComposerPasteResult
type ComposerCopyResult = composer.ComposerCopyResult

func DetectComposerDroppedFile(input string, opts ComposerDropOptions) ComposerDropResult {
	return composer.DetectComposerDroppedFile(input, opts)
}

func LooksLikeComposerDroppedPath(text string) bool {
	return composer.LooksLikeComposerDroppedPath(text)
}

func IsComposerImagePath(path string) bool {
	return composer.IsComposerImagePath(path)
}

func CollapseComposerPaste(text string, opts ComposerPasteOptions) ComposerPasteResult {
	return composer.CollapseComposerPaste(text, opts)
}

func ExpandComposerPasteSnippets(input string, snippets []ComposerPasteSnippet, readFile func(string) ([]byte, error)) (string, error) {
	return composer.ExpandComposerPasteSnippets(input, snippets, readFile)
}

func RecoverComposerBracketedPaste(input string) string {
	return composer.RecoverComposerBracketedPaste(input)
}

func SelectComposerCopyText(history []llm.Message, arg string) ComposerCopyResult {
	return composer.SelectComposerCopyText(history, arg)
}

func StripComposerReasoningBlocks(text string) string {
	return composer.StripComposerReasoningBlocks(text)
}

// ─── Composer input chrome facades ──────────────────────────────────────────

type TextInputChrome struct {
	Width int
	Label string
	Hint  string
	Value string
	Skin  HermesSkin
}

func RenderTextInputChrome(in TextInputChrome) string {
	styles := SkinStylesFor(in.Skin)
	return chrome.RenderTextInput(chrome.TextInput{
		Width: in.Width,
		Label: in.Label,
		Hint:  in.Hint,
		Value: in.Value,
		Styles: chrome.TextInputStyles{
			Label: styles.Label,
			Dim:   styles.Dim,
		},
	})
}

type ComposerInputChrome struct {
	Width     int
	Prompt    string
	Draft     string
	Skin      HermesSkin
	Focused   bool
	Multiline bool
}

func RenderComposerInputChrome(in ComposerInputChrome) string {
	styles := SkinStylesFor(in.Skin)
	return chrome.RenderComposerInput(chrome.ComposerInput{
		Width:     in.Width,
		Prompt:    in.Prompt,
		Draft:     in.Draft,
		Focused:   in.Focused,
		Multiline: in.Multiline,
		Styles: chrome.TextInputStyles{
			Label: styles.Label,
			Dim:   styles.Dim,
		},
		KeyHelp: keyHelpStyles(in.Skin),
	})
}

func (in ComposerInputChrome) KeyHelp() []KeyHelp {
	return chrome.ComposerKeyHelp(in.Draft, in.Multiline)
}

func composerInputChromeExtraRows(width int) int {
	return chrome.ComposerInputExtraRows(width)
}

func showComposerInputChrome(width int, height int) bool {
	return chrome.ShowComposerInput(width, height)
}

// ─── Running placeholder ────────────────────────────────────────────────────

const idleEditorPlaceholder = composer.IdleEditorPlaceholder
const cancelHotkey = composer.CancelHotkey

// ─── Fast echo shape gates ──────────────────────────────────────────────────

func CanFastAppendShape(current string, cursor int, text string, columns int, currentLineWidth int) bool {
	return composer.CanFastAppendShape(current, cursor, text, columns, currentLineWidth)
}

func CanFastBackspaceShape(current string, cursor int, columns ...int) bool {
	return composer.CanFastBackspaceShape(current, cursor, columns...)
}

func SupportsFastEchoTerminal(env map[string]string) bool {
	return composer.SupportsFastEchoTerminal(env)
}

// ─── Terminal setup facades ────────────────────────────────────────────────

type TerminalSetupFileOps = terminal.TerminalSetupFileOps
type TerminalSetupOptions = terminal.TerminalSetupOptions
type TerminalSetupResult = terminal.TerminalSetupResult
type TruecolorResult = terminal.TruecolorResult
type TerminalParityHint = terminal.TerminalParityHint

func DetectVSCodeLikeTerminal(env map[string]string) string {
	return terminal.DetectVSCodeLikeTerminal(env)
}

func VSCodeStyleConfigDir(app, platform string, env map[string]string, home string) string {
	return terminal.VSCodeStyleConfigDir(app, platform, env, home)
}

func StripJSONComments(input string) string {
	return terminal.StripJSONComments(input)
}

func ConfigureDetectedTerminalKeybindings(opts TerminalSetupOptions) TerminalSetupResult {
	return terminal.ConfigureDetectedTerminalKeybindings(opts)
}

func ConfigureTerminalKeybindings(kind string, opts TerminalSetupOptions) TerminalSetupResult {
	return terminal.ConfigureTerminalKeybindings(kind, opts)
}

func ShouldPromptForTerminalSetup(opts TerminalSetupOptions) bool {
	return terminal.ShouldPromptForTerminalSetup(opts)
}

func TruecolorDecision(env map[string]string) TruecolorResult {
	return terminal.TruecolorDecision(env)
}

func TerminalParityHints(env map[string]string, opts TerminalSetupOptions) []TerminalParityHint {
	return terminal.TerminalParityHints(env, opts)
}

// ─── Selection help facades ────────────────────────────────────────────────

const TerminalNativeSelectionHelp = terminal.NativeSelectionHelp

func SelectionHelpLine() string {
	return terminal.SelectionHelpLine()
}

// ─── OSC52 clipboard facades ───────────────────────────────────────────────

const OSC52ClipboardQuery = terminal.OSC52ClipboardQuery

type OSC52Response = terminal.OSC52Response
type OSC52SendFunc = terminal.OSC52SendFunc

func BuildOSC52ClipboardQuery(env map[string]string) string {
	return terminal.BuildOSC52ClipboardQuery(env)
}

func ParseOSC52ClipboardData(data string) (string, bool) {
	return terminal.ParseOSC52ClipboardData(data)
}

func ReadOSC52Clipboard(env map[string]string, send OSC52SendFunc, flush func() error) ClipboardTextResult {
	return terminal.ReadOSC52Clipboard(env, send, flush)
}

// ─── Clipboard facades ─────────────────────────────────────────────────────

type ClipboardCommandOptions = terminal.ClipboardCommandOptions
type ClipboardRunResult = terminal.ClipboardRunResult
type ClipboardRunFunc = terminal.ClipboardRunFunc
type ClipboardReadRequest = terminal.ClipboardReadRequest
type ClipboardTextResult = terminal.ClipboardTextResult
type ClipboardStartFunc = terminal.ClipboardStartFunc
type ClipboardWriteRequest = terminal.ClipboardWriteRequest
type ClipboardWriteResult = terminal.ClipboardWriteResult

func ReadClipboardText(req ClipboardReadRequest) ClipboardTextResult {
	return terminal.ReadClipboardText(req)
}

func IsUsableClipboardText(text string) bool {
	return terminal.IsUsableClipboardText(text)
}

func WriteClipboardText(req ClipboardWriteRequest) ClipboardWriteResult {
	return terminal.WriteClipboardText(req)
}

// ─── Mouse tracking facades ────────────────────────────────────────────────

const mouseSlashUsage = terminal.MouseSlashUsage

type mouseSlashResult struct {
	handled bool
	valid   bool
	next    bool
	message string
}

func parseMouseTrackingSlash(input string, current bool) mouseSlashResult {
	result := terminal.ParseMouseTrackingSlash(input, current)
	return mouseSlashResult{
		handled: result.Handled,
		valid:   result.Valid,
		next:    result.Next,
		message: result.Message,
	}
}

func defaultMouseModeCmd(enabled bool) tea.Cmd {
	return terminal.DefaultMouseModeCmd(enabled)
}