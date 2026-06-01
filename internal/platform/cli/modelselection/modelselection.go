// Package modelselection preserves the historical public model-selection
// import path while implementation code lives in responsibility-focused
// subpackages.
package modelselection

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection/catalog"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection/menu"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection/picker"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection/selector"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection/tasks"
)

type SelectionKind = selector.SelectionKind

const (
	SelectionKindModel    = selector.SelectionKindModel
	SelectionKindProvider = selector.SelectionKindProvider
	SelectionKindAccount  = selector.SelectionKindAccount
)

type Selection = selector.Selection
type Profile = selector.Profile
type ModelSelector = selector.ModelSelector
type ProfileSelector = selector.ProfileSelector
type ModelSelectorFunc = selector.ModelSelectorFunc
type ProfileSelectorFunc = selector.ProfileSelectorFunc

var ErrSelectorNoMatch = selector.ErrSelectorNoMatch
var ErrSelectorHelperUnavailable = selector.ErrSelectorHelperUnavailable
var ErrSelectorAliasResolutionFailed = selector.ErrSelectorAliasResolutionFailed

type ProviderModel = selector.ProviderModel
type ReadInferenceDefaultsFunc = selector.ReadInferenceDefaultsFunc
type ResolveAliasFunc = selector.ResolveAliasFunc
type ResolveAccountFunc = selector.ResolveAccountFunc
type ReadActiveProfileNameFunc = selector.ReadActiveProfileNameFunc
type ValidateProfileNameFunc = selector.ValidateProfileNameFunc
type ResolveProfileRootFunc = selector.ResolveProfileRootFunc
type DefaultModelSelectorOptions = selector.DefaultModelSelectorOptions
type DefaultProfileSelectorOptions = selector.DefaultProfileSelectorOptions

func NewDefaultModelSelector(opts DefaultModelSelectorOptions) ModelSelector {
	return selector.NewDefaultModelSelector(opts)
}
func NewDefaultProfileSelector(opts DefaultProfileSelectorOptions) ProfileSelector {
	return selector.NewDefaultProfileSelector(opts)
}

type ProviderMenuEntry = menu.ProviderEntry

var ErrModelPickerRequiresTTY = picker.ErrModelPickerRequiresTTY
var ErrModelPickerCancelled = picker.ErrModelPickerCancelled
var ErrModelPickerNoProviders = picker.ErrModelPickerNoProviders

type ModelPickerOptions = picker.ModelPickerOptions
type ModelPicker = picker.ModelPicker
type AuxiliaryTaskEntry = tasks.AuxiliaryTaskEntry

func DefaultAuxiliaryTaskEntries() []AuxiliaryTaskEntry {
	return tasks.DefaultAuxiliaryTaskEntries()
}
func NewModelPicker(opts ModelPickerOptions) *ModelPicker { return picker.NewModelPicker(opts) }

const (
	ProviderCatalogAuxConfig      = catalog.ProviderCatalogAuxConfig
	ProviderCatalogLeaveUnchanged = catalog.ProviderCatalogLeaveUnchanged
)

type ProviderCatalogEntry = catalog.ProviderCatalogEntry

func HermesProviderCatalog() []ProviderCatalogEntry { return catalog.HermesProviderCatalog() }
func HermesModelProviderCatalog() []ProviderCatalogEntry {
	return catalog.HermesModelProviderCatalog()
}
func HermesModelProviderMenu() []ProviderMenuEntry { return catalog.HermesModelProviderMenu() }
func HermesProviderCatalogMenu(activeProvider string) ([]ProviderMenuEntry, int) {
	return catalog.HermesProviderCatalogMenu(activeProvider)
}
