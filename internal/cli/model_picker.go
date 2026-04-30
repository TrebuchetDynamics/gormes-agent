package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ProviderMenuEntry is one provider option rendered by the interactive model
// picker. Label is user-visible and may be annotated by the picker for the
// active provider; ID is the canonical provider identifier persisted on select.
type ProviderMenuEntry struct {
	ID    string
	Label string
}

var (
	ErrModelPickerRequiresTTY = errors.New("model_picker_requires_tty")
	ErrModelPickerCancelled   = errors.New("model_picker_cancelled")
	ErrModelPickerNoProviders = errors.New("model_picker_no_providers")
)

type ModelPickerOptions struct {
	IsTTY func() bool

	LoadCurrent   func() (ProviderModel, error)
	ListProviders func() ([]ProviderMenuEntry, error)

	ChooseProvider func(entries []ProviderMenuEntry, defaultIndex int) (int, error)
	ChooseModel    func(provider string, current string) (string, error)

	PersistSelection func(Selection) error

	// AuthFlow is intentionally never invoked by ModelPicker. It exists as an
	// injectable canary for tests and documents the row contract: provider auth
	// belongs to `gormes auth add <provider>`, not the selection-only picker.
	AuthFlow func(provider string) error
}

type ModelPicker struct {
	opts ModelPickerOptions
}

func NewModelPicker(opts ModelPickerOptions) *ModelPicker {
	return &ModelPicker{opts: opts}
}

func (p *ModelPicker) Pick(ctx context.Context) (Selection, error) {
	if p.opts.IsTTY == nil || !p.opts.IsTTY() {
		return Selection{}, ErrModelPickerRequiresTTY
	}
	if err := ctx.Err(); err != nil {
		return Selection{}, err
	}
	if p.opts.LoadCurrent == nil || p.opts.ListProviders == nil || p.opts.ChooseProvider == nil {
		return Selection{}, fmt.Errorf("model picker: %w", ErrSelectorHelperUnavailable)
	}
	current, err := p.opts.LoadCurrent()
	if err != nil {
		return Selection{}, fmt.Errorf("model picker: load current: %w", err)
	}
	providers, err := p.opts.ListProviders()
	if err != nil {
		return Selection{}, fmt.Errorf("model picker: list providers: %w", err)
	}
	if len(providers) == 0 {
		return Selection{}, ErrModelPickerNoProviders
	}
	entries, defaultIndex := annotateCurrentProvider(providers, current.Provider)
	idx, err := p.opts.ChooseProvider(entries, defaultIndex)
	if err != nil {
		if errors.Is(err, ErrModelPickerCancelled) {
			return Selection{}, ErrModelPickerCancelled
		}
		return Selection{}, fmt.Errorf("model picker: choose provider: %w", err)
	}
	if idx < 0 || idx >= len(providers) {
		return Selection{}, ErrModelPickerCancelled
	}
	provider := strings.TrimSpace(providers[idx].ID)
	if p.opts.ChooseModel == nil || p.opts.PersistSelection == nil {
		return Selection{}, fmt.Errorf("model picker: %w", ErrSelectorHelperUnavailable)
	}
	model, err := p.opts.ChooseModel(provider, current.Model)
	if err != nil {
		if errors.Is(err, ErrModelPickerCancelled) {
			return Selection{}, ErrModelPickerCancelled
		}
		return Selection{}, fmt.Errorf("model picker: choose model: %w", err)
	}
	selection := Selection{Provider: provider, Model: strings.TrimSpace(model)}
	if selection.Provider == "" || selection.Model == "" {
		return Selection{}, ErrSelectorNoMatch
	}
	if err := p.opts.PersistSelection(selection); err != nil {
		return Selection{}, fmt.Errorf("model picker: persist selection: %w", err)
	}
	return selection, nil
}

func annotateCurrentProvider(providers []ProviderMenuEntry, activeProvider string) ([]ProviderMenuEntry, int) {
	entries := append([]ProviderMenuEntry(nil), providers...)
	defaultIndex := 0
	activeProvider = strings.TrimSpace(activeProvider)
	for i := range entries {
		if activeProvider != "" && strings.TrimSpace(entries[i].ID) == activeProvider {
			entries[i].Label = strings.TrimSpace(entries[i].Label) + "  ← currently active"
			defaultIndex = i
		}
	}
	return entries, defaultIndex
}
