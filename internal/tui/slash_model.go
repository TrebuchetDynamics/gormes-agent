package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker"
)

// ModelPickerCatalogProvider is the TUI-local provider/model catalog shape
// consumed by the /model overlay.
type ModelPickerCatalogProvider = modelpicker.CatalogProvider

// ModelPickerCatalogFunc returns a fresh provider/model catalog for the TUI
// /model picker. It is injected so local startup can use the built-in Hermes
// catalog while tests can fixture degraded states without network calls.
type ModelPickerCatalogFunc func() ([]ModelPickerCatalogProvider, error)

type modelSessionSetMsg struct {
	Provider string
	Model    string
	Err      error
}

// DefaultModelPickerCatalog adapts the shared Hermes picker provider list and
// curated model suggestions into the pure renderer entries used by
// ModelPickerState.
func DefaultModelPickerCatalog() ([]ModelPickerCatalogProvider, error) {
	return modelpicker.DefaultCatalog()
}

func modelSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "model: TUI unavailable"}
	}
	if model.inFlight {
		return SlashResult{Handled: true, StatusMessage: "model: cannot switch models while a turn is running"}
	}
	if arg := modelSlashArgument(input); arg != "" {
		return model.applyModelSelection(model.currentModelProvider(), arg)
	}
	catalog, err := model.loadModelPickerCatalog()
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "model: catalog unavailable: " + err.Error()}
	}
	if len(catalog) == 0 {
		return SlashResult{Handled: true, StatusMessage: "model: catalog unavailable"}
	}
	state := newModelPickerState(catalog, model.currentModelProvider(), model.currentModelName(), model.width, model.height)
	model.modelPicker = &state
	model.modelPickerChoices = catalog
	return SlashResult{Handled: true, StatusMessage: "model: select provider/model"}
}

func modelSlashArgument(input string) string {
	return modelpicker.SlashArgument(input)
}

func (m *Model) loadModelPickerCatalog() ([]ModelPickerCatalogProvider, error) {
	fn := m.modelPickerCatalog
	if fn == nil {
		return nil, fmt.Errorf("no model catalog configured")
	}
	catalog, err := fn()
	if err != nil {
		return nil, err
	}
	return normalizeModelPickerCatalog(catalog), nil
}

func normalizeModelPickerCatalog(catalog []ModelPickerCatalogProvider) []ModelPickerCatalogProvider {
	return modelpicker.NormalizeCatalog(catalog)
}

func newModelPickerState(catalog []ModelPickerCatalogProvider, currentProvider, currentModel string, width, height int) ModelPickerState {
	return modelpicker.NewState(catalog, currentProvider, currentModel, width, height)
}

func modelsForProviderIndex(catalog []ModelPickerCatalogProvider, idx int) []ModelEntry {
	return modelpicker.ModelsForProviderIndex(catalog, idx)
}

func (m *Model) updateModelPickerForKey(msg tea.KeyMsg) tea.Cmd {
	if m.modelPicker == nil {
		return nil
	}
	next, cmd := UpdateModelPicker(msg, *m.modelPicker)
	next.Width = m.width
	next.Height = m.height
	if next.SelectedProviderIndex >= 0 && next.SelectedProviderIndex < len(m.modelPickerChoices) {
		next.Models = modelsForProviderIndex(m.modelPickerChoices, next.SelectedProviderIndex)
		if next.SelectedModelIndex >= len(next.Models) {
			next.SelectedModelIndex = len(next.Models) - 1
		}
	}
	m.modelPicker = &next
	return cmd
}

func (m *Model) handleModelPickerConfirmed(result ModelPickerResult) tea.Cmd {
	m.modelPicker = nil
	if strings.TrimSpace(result.Provider) == "" {
		m.statusMessage = "model: unchanged"
		return nil
	}
	provider, model := m.normalizeConfirmedModelSelection(result.Provider, result.Model)
	if strings.TrimSpace(model) == "" {
		m.statusMessage = "model: no model selected"
		return nil
	}
	res := m.applyModelSelection(provider, model)
	if res.StatusMessage != "" {
		m.statusMessage = res.StatusMessage
	}
	return res.Cmd
}

func (m *Model) normalizeConfirmedModelSelection(provider, model string) (string, string) {
	return modelpicker.NormalizeConfirmedSelection(m.modelPickerChoices, provider, model)
}

func (m *Model) applyModelSelection(provider, model string) SlashResult {
	model = strings.TrimSpace(model)
	if model == "" {
		return SlashResult{Handled: true, StatusMessage: "model: no model selected"}
	}
	provider = strings.TrimSpace(provider)
	if m.setSessionModel == nil {
		return SlashResult{Handled: true, StatusMessage: "model: switch unavailable"}
	}
	return SlashResult{
		Handled: true,
		Cmd: func() tea.Msg {
			err := m.setSessionModel(provider, model)
			return modelSessionSetMsg{Provider: provider, Model: model, Err: err}
		},
	}
}

func (m *Model) handleModelSessionSet(msg modelSessionSetMsg) {
	if msg.Err != nil {
		m.statusMessage = "model: " + msg.Err.Error()
		return
	}
	provider := strings.TrimSpace(msg.Provider)
	model := strings.TrimSpace(msg.Model)
	if provider != "" {
		m.modelProvider = provider
	}
	if model != "" {
		m.modelName = model
		m.frame.Model = model
	}
	m.statusMessage = fmt.Sprintf("model -> %s", model)
}

func (m *Model) currentModelProvider() string {
	return strings.TrimSpace(m.modelProvider)
}

func (m *Model) currentModelName() string {
	if model := strings.TrimSpace(m.frame.Model); model != "" {
		return model
	}
	return strings.TrimSpace(m.modelName)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
