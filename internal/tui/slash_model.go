package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// ModelPickerCatalogProvider is the TUI-local provider/model catalog shape
// consumed by the /model overlay.
type ModelPickerCatalogProvider struct {
	Provider ProviderEntry
	Models   []ModelEntry
}

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
	providers := llm.ListPickerProviders()
	out := make([]ModelPickerCatalogProvider, 0, len(providers))
	for _, provider := range providers {
		id := strings.TrimSpace(provider.Slug)
		if id == "" {
			continue
		}
		modelIDs := provider.Models
		if len(modelIDs) == 0 {
			modelIDs = llm.ProviderModelCatalogSuggestions(id, nil)
		}
		models := make([]ModelEntry, 0, len(modelIDs))
		for _, modelID := range modelIDs {
			modelID = strings.TrimSpace(modelID)
			if modelID == "" {
				continue
			}
			models = append(models, ModelEntry{ID: modelID, Label: modelID})
		}
		if len(models) == 0 {
			continue
		}
		out = append(out, ModelPickerCatalogProvider{
			Provider: ProviderEntry{ID: id, Label: firstNonEmptyString(strings.TrimSpace(provider.Label), id)},
			Models:   models,
		})
	}
	return out, nil
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
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	fields := strings.Fields(trimmed)
	if len(fields) <= 1 {
		return ""
	}
	idx := strings.Index(trimmed, fields[1])
	if idx < 0 {
		return strings.Join(fields[1:], " ")
	}
	return strings.TrimSpace(trimmed[idx:])
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
	out := make([]ModelPickerCatalogProvider, 0, len(catalog))
	for _, entry := range catalog {
		providerID := strings.TrimSpace(entry.Provider.ID)
		if providerID == "" {
			continue
		}
		label := firstNonEmptyString(strings.TrimSpace(entry.Provider.Label), providerID)
		models := make([]ModelEntry, 0, len(entry.Models))
		for _, model := range entry.Models {
			modelID := strings.TrimSpace(model.ID)
			if modelID == "" {
				continue
			}
			models = append(models, ModelEntry{
				ID:    modelID,
				Label: firstNonEmptyString(strings.TrimSpace(model.Label), modelID),
			})
		}
		if len(models) == 0 {
			continue
		}
		out = append(out, ModelPickerCatalogProvider{
			Provider: ProviderEntry{ID: providerID, Label: label},
			Models:   models,
		})
	}
	return out
}

func newModelPickerState(catalog []ModelPickerCatalogProvider, currentProvider, currentModel string, width, height int) ModelPickerState {
	providers := make([]ProviderEntry, 0, len(catalog))
	selectedProvider := 0
	for i, entry := range catalog {
		providers = append(providers, entry.Provider)
		if currentProvider != "" && strings.EqualFold(entry.Provider.ID, currentProvider) {
			selectedProvider = i
		}
	}
	models := modelsForProviderIndex(catalog, selectedProvider)
	return ModelPickerState{
		Width:                 width,
		Height:                height,
		Providers:             providers,
		SelectedProviderIndex: selectedProvider,
		Models:                models,
		SelectedModelIndex:    -1,
		CurrentProvider:       currentProvider,
		CurrentModel:          currentModel,
	}
}

func modelsForProviderIndex(catalog []ModelPickerCatalogProvider, idx int) []ModelEntry {
	if idx < 0 || idx >= len(catalog) {
		return nil
	}
	return append([]ModelEntry(nil), catalog[idx].Models...)
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
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	for _, entry := range m.modelPickerChoices {
		if !strings.EqualFold(entry.Provider.ID, provider) {
			continue
		}
		if model != "" {
			for _, candidate := range entry.Models {
				if candidate.ID == model {
					return provider, model
				}
			}
		}
		if len(entry.Models) > 0 {
			return provider, entry.Models[0].ID
		}
	}
	return provider, model
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
