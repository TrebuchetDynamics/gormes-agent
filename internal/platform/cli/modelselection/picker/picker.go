package picker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection/menu"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection/selector"
)

type ProviderMenuEntry = menu.ProviderEntry
type ProviderModel = selector.ProviderModel
type Selection = selector.Selection

var (
	ErrModelPickerRequiresTTY = errors.New("model_picker_requires_tty")
	ErrModelPickerCancelled   = errors.New("model_picker_cancelled")
	ErrModelPickerNoProviders = errors.New("model_picker_no_providers")
)

type ModelPickerOptions struct {
	IsTTY func() bool

	LoadCurrent   func() (selector.ProviderModel, error)
	ListProviders func() ([]ProviderMenuEntry, error)

	ChooseProvider func(entries []ProviderMenuEntry, defaultIndex int) (int, error)
	ChooseModel    func(provider string, current string) (string, error)

	PersistSelection func(selector.Selection) error

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

func (p *ModelPicker) Pick(ctx context.Context) (selector.Selection, error) {
	if p.opts.IsTTY == nil || !p.opts.IsTTY() {
		return selector.Selection{}, ErrModelPickerRequiresTTY
	}
	if err := ctx.Err(); err != nil {
		return selector.Selection{}, err
	}
	if p.opts.LoadCurrent == nil || p.opts.ListProviders == nil || p.opts.ChooseProvider == nil {
		return selector.Selection{}, fmt.Errorf("model picker: %w", selector.ErrSelectorHelperUnavailable)
	}
	current, err := p.opts.LoadCurrent()
	if err != nil {
		return selector.Selection{}, fmt.Errorf("model picker: load current: %w", err)
	}
	providers, err := p.opts.ListProviders()
	if err != nil {
		return selector.Selection{}, fmt.Errorf("model picker: list providers: %w", err)
	}
	if len(providers) == 0 {
		return selector.Selection{}, ErrModelPickerNoProviders
	}
	entries, defaultIndex := menu.AnnotateCurrentProvider(providers, current.Provider, 0)
	idx, err := p.opts.ChooseProvider(entries, defaultIndex)
	if err != nil {
		if errors.Is(err, ErrModelPickerCancelled) {
			return selector.Selection{}, ErrModelPickerCancelled
		}
		return selector.Selection{}, fmt.Errorf("model picker: choose provider: %w", err)
	}
	if idx < 0 || idx >= len(providers) {
		return selector.Selection{}, ErrModelPickerCancelled
	}
	provider := strings.TrimSpace(providers[idx].ID)
	if p.opts.ChooseModel == nil || p.opts.PersistSelection == nil {
		return selector.Selection{}, fmt.Errorf("model picker: %w", selector.ErrSelectorHelperUnavailable)
	}
	model, err := p.opts.ChooseModel(provider, current.Model)
	if err != nil {
		if errors.Is(err, ErrModelPickerCancelled) {
			return selector.Selection{}, ErrModelPickerCancelled
		}
		return selector.Selection{}, fmt.Errorf("model picker: choose model: %w", err)
	}
	selection := selector.Selection{Provider: provider, Model: strings.TrimSpace(model)}
	if selection.Provider == "" || selection.Model == "" {
		return selector.Selection{}, selector.ErrSelectorNoMatch
	}
	if err := p.opts.PersistSelection(selection); err != nil {
		return selector.Selection{}, fmt.Errorf("model picker: persist selection: %w", err)
	}
	return selection, nil
}
