package tui

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/extensionui"
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
	key = extensionui.Key(key)
	if key == "" {
		return kernel.ExtensionUIResult{Status: kernel.ExtensionUINoop, Evidence: "extension UI status key required"}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return c.ClearStatus(key)
	}
	m.ensureExtensionUIState()
	m.extensionUI.statuses[key] = extensionui.Text{SessionID: c.sessionID, Text: text}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUIApplied, Evidence: fmt.Sprintf("extension UI status set: %s", key)}
}

func (c ExtensionUIContext) ClearStatus(key string) kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	key = extensionui.Key(key)
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
	key = extensionui.Key(key)
	if key == "" {
		return kernel.ExtensionUIResult{Status: kernel.ExtensionUINoop, Evidence: "extension UI widget key required"}
	}
	clean := extensionui.Lines(lines)
	if len(clean) == 0 {
		return c.ClearWidget(key)
	}
	m.ensureExtensionUIState()
	m.extensionUI.widgets[key] = extensionui.Widget{
		SessionID: c.sessionID,
		Lines:     clean,
		Placement: kernel.NormalizeExtensionUIWidgetPlacement(opts.Placement),
	}
	return kernel.ExtensionUIResult{Status: kernel.ExtensionUIApplied, Evidence: fmt.Sprintf("extension UI widget set: %s", key)}
}

func (c ExtensionUIContext) ClearWidget(key string) kernel.ExtensionUIResult {
	m, ok := c.modelForMutation()
	if !ok {
		return c.unavailable()
	}
	key = extensionui.Key(key)
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
	clean := extensionui.Lines(lines)
	if len(clean) == 0 {
		return c.ClearFooter()
	}
	m.ensureExtensionUIState()
	m.extensionUI.footer = &extensionui.LinesState{SessionID: c.sessionID, Lines: clean}
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
	frames := extensionui.Lines(opts.Frames)
	m.ensureExtensionUIState()
	m.extensionUI.working = &extensionui.Working{
		SessionID:     c.sessionID,
		Text:          strings.TrimSpace(opts.Text),
		Frames:        frames,
		HideIndicator: opts.Frames != nil && len(frames) == 0,
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
	statuses map[string]extensionui.Text
	widgets  map[string]extensionui.Widget
	footer   *extensionui.LinesState
	working  *extensionui.Working
}

func (m *Model) ensureExtensionUIState() {
	if m.extensionUI.statuses == nil {
		m.extensionUI.statuses = make(map[string]extensionui.Text)
	}
	if m.extensionUI.widgets == nil {
		m.extensionUI.widgets = make(map[string]extensionui.Widget)
	}
}

func (m Model) renderExtensionStatusLine(base string, width int) string {
	return extensionui.StatusLine(base, m.extensionStatusEntries(), width, hermesStatusTrimToWidth)
}

func (m Model) renderExtensionFooter(width int) (string, bool) {
	footer := m.extensionUI.footer
	if footer == nil || footer.SessionID != m.SessionID() {
		return "", false
	}
	return extensionui.RenderLines(footer.Lines, width, hermesStatusTrimToWidth), true
}

func (m Model) renderExtensionWidgets(placement kernel.ExtensionUIWidgetPlacement, width int) string {
	return extensionui.WidgetBlocks(m.extensionUI.widgets, m.SessionID(), placement, width, hermesStatusTrimToWidth)
}

func (m Model) extensionStatusEntries() []string {
	return extensionui.StatusEntries(m.extensionUI.statuses, m.SessionID())
}

func (m Model) extensionWorkingIndicator() (extensionui.Working, bool) {
	working := m.extensionUI.working
	if working == nil || working.SessionID != m.SessionID() {
		return extensionui.Working{}, false
	}
	return *working, true
}
