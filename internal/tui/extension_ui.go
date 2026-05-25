package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

var _ kernel.ExtensionUI = ExtensionUIContext{}

// ExtensionUIContext is the small Go-native UI surface handed to trusted
// in-process Gormes extensions. It mutates only local TUI chrome state for the
// session that was active when the context was created; nil or non-interactive
// contexts return typed no-op evidence instead of panicking.
type ExtensionUIContext struct {
	model       *Model
	sessionID   string
	interactive bool
	reason      string
}

func NewExtensionUIContext(model *Model, interactive bool) ExtensionUIContext {
	if !interactive || model == nil {
		reason := "non-interactive extension UI unavailable"
		if model == nil {
			reason += ": nil TUI model"
		}
		return ExtensionUIContext{reason: reason}
	}
	return ExtensionUIContext{model: model, sessionID: model.SessionID(), interactive: true}
}

func (c ExtensionUIContext) SetStatus(key, text string) kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	key = extensionUIKey(key)
	if key == "" {
		return kernel.ExtensionUIResult{Status: kernel.ExtensionUINoop, Evidence: "extension UI status key required"}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return c.ClearStatus(key)
	}
	m.ensureExtensionUIState()
	m.extensionUI.statuses[key] = extensionUIText{sessionID: c.sessionID, text: text}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUIApplied, Evidence: fmt.Sprintf("extension UI status set: %s", key)}
}

func (c ExtensionUIContext) ClearStatus(key string) kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	key = extensionUIKey(key)
	if key == "" {
		return kernel.ExtensionUIResult{Status: kernel.ExtensionUINoop, Evidence: "extension UI status key required"}
	}
	if m.extensionUI.statuses != nil {
		delete(m.extensionUI.statuses, key)
	}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUICleared, Evidence: fmt.Sprintf("extension UI status cleared: %s", key)}
}

func (c ExtensionUIContext) SetWidget(key string, lines []string, opts kernel.ExtensionUIWidgetOptions) kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	key = extensionUIKey(key)
	if key == "" {
		return kernel.ExtensionUIResult{Status: kernel.ExtensionUINoop, Evidence: "extension UI widget key required"}
	}
	clean := extensionUILines(lines)
	if len(clean) == 0 {
		return c.ClearWidget(key)
	}
	m.ensureExtensionUIState()
	m.extensionUI.widgets[key] = extensionUIWidget{
		sessionID: c.sessionID,
		lines:     clean,
		placement: kernel.NormalizeExtensionUIWidgetPlacement(opts.Placement),
	}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUIApplied, Evidence: fmt.Sprintf("extension UI widget set: %s", key)}
}

func (c ExtensionUIContext) ClearWidget(key string) kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	key = extensionUIKey(key)
	if key == "" {
		return kernel.ExtensionUIResult{Status: kernel.ExtensionUINoop, Evidence: "extension UI widget key required"}
	}
	if m.extensionUI.widgets != nil {
		delete(m.extensionUI.widgets, key)
	}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUICleared, Evidence: fmt.Sprintf("extension UI widget cleared: %s", key)}
}

func (c ExtensionUIContext) SetFooter(lines []string) kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	clean := extensionUILines(lines)
	if len(clean) == 0 {
		return c.ClearFooter()
	}
	m.ensureExtensionUIState()
	m.extensionUI.footer = &extensionUILinesState{sessionID: c.sessionID, lines: clean}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUIApplied, Evidence: "extension UI footer set"}
}

func (c ExtensionUIContext) ClearFooter() kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	m.extensionUI.footer = nil
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUICleared, Evidence: "extension UI footer cleared"}
}

func (c ExtensionUIContext) SetWorkingIndicator(opts kernel.ExtensionUIWorkingIndicator) kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	frames := extensionUILines(opts.Frames)
	m.ensureExtensionUIState()
	m.extensionUI.working = &extensionUIWorking{
		sessionID:     c.sessionID,
		text:          strings.TrimSpace(opts.Text),
		frames:        frames,
		hideIndicator: opts.Frames != nil && len(frames) == 0,
	}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUIApplied, Evidence: "extension UI working indicator set"}
}

func (c ExtensionUIContext) ClearWorkingIndicator() kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	m.extensionUI.working = nil
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUICleared, Evidence: "extension UI working indicator cleared"}
}

func (c ExtensionUIContext) modelForMutation() (*Model, bool) {
	return c.model, c.interactive && c.model != nil
}

func (c ExtensionUIContext) unavailable() kernel.ExtensionUIResult {
	reason := strings.TrimSpace(c.reason)
	if reason == "" {
		reason = "non-interactive extension UI unavailable"
	}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUIUnavailable, Evidence: reason}
}

type extensionUIState struct {
	statuses map[string]extensionUIText
	widgets  map[string]extensionUIWidget
	footer   *extensionUILinesState
	working  *extensionUIWorking
}

type extensionUIText struct {
	sessionID string
	text      string
}

type extensionUILinesState struct {
	sessionID string
	lines     []string
}

type extensionUIWidget struct {
	sessionID string
	lines     []string
	placement kernel.ExtensionUIWidgetPlacement
}

type extensionUIWorking struct {
	sessionID     string
	text          string
	frames        []string
	hideIndicator bool
}

func (m *Model) ensureExtensionUIState() {
	if m.extensionUI.statuses == nil {
		m.extensionUI.statuses = make(map[string]extensionUIText)
	}
	if m.extensionUI.widgets == nil {
		m.extensionUI.widgets = make(map[string]extensionUIWidget)
	}
}

func extensionUIKey(key string) string {
	return strings.TrimSpace(key)
}

func extensionUILines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r\n")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func (m Model) renderExtensionStatusLine(base string, width int) string {
	entries := m.extensionStatusEntries()
	if len(entries) == 0 {
		return hermesStatusTrimToWidth(base, width)
	}
	suffix := " │ " + strings.Join(entries, " │ ")
	suffix = hermesStatusTrimToWidth(suffix, width)
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(suffix) >= width {
		return suffix
	}
	baseWidth := width - lipgloss.Width(suffix)
	return hermesStatusTrimToWidth(base, baseWidth) + suffix
}

func (m Model) renderExtensionFooter(width int) (string, bool) {
	footer := m.extensionUI.footer
	if footer == nil || footer.sessionID != m.SessionID() {
		return "", false
	}
	return extensionUIRenderLines(footer.lines, width), true
}

func (m Model) renderExtensionWidgets(placement kernel.ExtensionUIWidgetPlacement, width int) string {
	placement = kernel.NormalizeExtensionUIWidgetPlacement(placement)
	if len(m.extensionUI.widgets) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m.extensionUI.widgets))
	for key, widget := range m.extensionUI.widgets {
		if widget.sessionID == m.SessionID() && kernel.NormalizeExtensionUIWidgetPlacement(widget.placement) == placement {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	blocks := make([]string, 0, len(keys))
	for _, key := range keys {
		blocks = append(blocks, extensionUIRenderLines(m.extensionUI.widgets[key].lines, width))
	}
	return strings.Join(blocks, "\n")
}

func (m Model) extensionStatusEntries() []string {
	if len(m.extensionUI.statuses) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m.extensionUI.statuses))
	for key, entry := range m.extensionUI.statuses {
		if entry.sessionID == m.SessionID() {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		text := strings.TrimSpace(m.extensionUI.statuses[key].text)
		if text != "" {
			entries = append(entries, text)
		}
	}
	return entries
}

func (m Model) extensionWorkingIndicator() (extensionUIWorking, bool) {
	working := m.extensionUI.working
	if working == nil || working.sessionID != m.SessionID() {
		return extensionUIWorking{}, false
	}
	return *working, true
}

func extensionUIRenderLines(lines []string, width int) string {
	if width <= 0 {
		return ""
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, hermesStatusTrimToWidth(line, width))
	}
	return strings.Join(out, "\n")
}
