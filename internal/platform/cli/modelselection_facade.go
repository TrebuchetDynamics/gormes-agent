package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/modelselection"

type SelectionKind = modelselection.SelectionKind

const (
	SelectionKindModel    = modelselection.SelectionKindModel
	SelectionKindProvider = modelselection.SelectionKindProvider
	SelectionKindAccount  = modelselection.SelectionKindAccount
)

type Selection = modelselection.Selection
type Profile = modelselection.Profile
type ModelSelector = modelselection.ModelSelector
type ProfileSelector = modelselection.ProfileSelector
type ModelSelectorFunc = modelselection.ModelSelectorFunc
type ProfileSelectorFunc = modelselection.ProfileSelectorFunc

var ErrSelectorNoMatch = modelselection.ErrSelectorNoMatch
var ErrSelectorHelperUnavailable = modelselection.ErrSelectorHelperUnavailable
var ErrSelectorAliasResolutionFailed = modelselection.ErrSelectorAliasResolutionFailed

type ProviderModel = modelselection.ProviderModel
type ReadInferenceDefaultsFunc = modelselection.ReadInferenceDefaultsFunc
type ResolveAliasFunc = modelselection.ResolveAliasFunc
type ResolveAccountFunc = modelselection.ResolveAccountFunc
type ReadActiveProfileNameFunc = modelselection.ReadActiveProfileNameFunc
type ValidateProfileNameFunc = modelselection.ValidateProfileNameFunc
type ResolveProfileRootFunc = modelselection.ResolveProfileRootFunc
type DefaultModelSelectorOptions = modelselection.DefaultModelSelectorOptions
type DefaultProfileSelectorOptions = modelselection.DefaultProfileSelectorOptions

func NewDefaultModelSelector(opts DefaultModelSelectorOptions) ModelSelector {
	return modelselection.NewDefaultModelSelector(opts)
}
func NewDefaultProfileSelector(opts DefaultProfileSelectorOptions) ProfileSelector {
	return modelselection.NewDefaultProfileSelector(opts)
}

type ProviderMenuEntry = modelselection.ProviderMenuEntry

var ErrModelPickerRequiresTTY = modelselection.ErrModelPickerRequiresTTY
var ErrModelPickerCancelled = modelselection.ErrModelPickerCancelled
var ErrModelPickerNoProviders = modelselection.ErrModelPickerNoProviders

type ModelPickerOptions = modelselection.ModelPickerOptions
type ModelPicker = modelselection.ModelPicker
type AuxiliaryTaskEntry = modelselection.AuxiliaryTaskEntry

func DefaultAuxiliaryTaskEntries() []AuxiliaryTaskEntry {
	return modelselection.DefaultAuxiliaryTaskEntries()
}
func NewModelPicker(opts ModelPickerOptions) *ModelPicker { return modelselection.NewModelPicker(opts) }

const (
	ProviderCatalogAuxConfig      = modelselection.ProviderCatalogAuxConfig
	ProviderCatalogLeaveUnchanged = modelselection.ProviderCatalogLeaveUnchanged
)

type ProviderCatalogEntry = modelselection.ProviderCatalogEntry

func HermesProviderCatalog() []ProviderCatalogEntry { return modelselection.HermesProviderCatalog() }
func HermesModelProviderCatalog() []ProviderCatalogEntry {
	return modelselection.HermesModelProviderCatalog()
}
func HermesModelProviderMenu() []ProviderMenuEntry { return modelselection.HermesModelProviderMenu() }
func HermesProviderCatalogMenu(activeProvider string) ([]ProviderMenuEntry, int) {
	return modelselection.HermesProviderCatalogMenu(activeProvider)
}
