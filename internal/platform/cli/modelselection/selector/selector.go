package selector

import (
	"context"
	"errors"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile"
)

// SelectionKind identifies which dimension of the (provider, model, account)
// triple a caller wants the selector to resolve. Constants are typed so the
// gateway slash-command path and the Cobra command path share one switch
// without parsing strings.
type SelectionKind int

const (
	// SelectionKindModel resolves the active model alias to the canonical
	// model id, alongside its provider and account.
	SelectionKindModel SelectionKind = iota
	// SelectionKindProvider resolves the active provider, leaving the model
	// alias to the caller.
	SelectionKindProvider
	// SelectionKindAccount resolves the active account/credential id used
	// for the configured provider+model pair.
	SelectionKindAccount
)

// Selection is the small result record returned by ModelSelector. It is the
// canonical (provider, model, account) triple consumed by the Cobra `gormes
// model` command and the gateway `/model` slash command. Concrete struct on
// purpose so callers can compose without runtime reflection; the seam itself
// remains an interface.
type Selection struct {
	Provider string
	Model    string
	Account  string
}

// Profile is the small result record returned by ProfileSelector: an active
// profile name plus the resolved on-disk root.
type Profile struct {
	Name     string
	RootPath string
}

// ModelSelector is the canonical entry point for resolving the active
// provider/model/account triple. Both `gormes model` (Cobra) and `/model`
// (gateway slash) construct one of these and ask it for a Selection.
type ModelSelector interface {
	Select(ctx context.Context, kind SelectionKind) (Selection, error)
}

// ProfileSelector is the canonical entry point for resolving the active
// profile descriptor. Both `gormes profile` (Cobra) and `/profile` (gateway
// slash) construct one of these and ask it for a Profile.
type ProfileSelector interface {
	Select(ctx context.Context) (Profile, error)
}

// ModelSelectorFunc adapts a plain function into a ModelSelector.
type ModelSelectorFunc func(ctx context.Context, kind SelectionKind) (Selection, error)

// Select implements ModelSelector.
func (fn ModelSelectorFunc) Select(ctx context.Context, kind SelectionKind) (Selection, error) {
	return fn(ctx, kind)
}

// ProfileSelectorFunc adapts a plain function into a ProfileSelector.
type ProfileSelectorFunc func(ctx context.Context) (Profile, error)

// Select implements ProfileSelector.
func (fn ProfileSelectorFunc) Select(ctx context.Context) (Profile, error) {
	return fn(ctx)
}

// Sentinel errors surface the `degraded_mode` typed evidence the row
// declares: selector_no_match, selector_helper_unavailable, and
// selector_alias_resolution_failed. Callers branch with errors.Is.
var (
	// ErrSelectorNoMatch is returned when no configured provider/model
	// alias resolves and there is nothing for the selector to return.
	ErrSelectorNoMatch = errors.New("selector_no_match")
	// ErrSelectorHelperUnavailable is returned when one of the upstream
	// helper rows (config bridge, alias resolver, profile helpers) is not
	// wired into the selector.
	ErrSelectorHelperUnavailable = profile.ErrSelectorHelperUnavailable
	// ErrSelectorAliasResolutionFailed is returned when the alias resolver
	// itself fails (network, fixture, or invariant). The original backend
	// error is wrapped with %w so callers can inspect both.
	ErrSelectorAliasResolutionFailed = errors.New("selector_alias_resolution_failed")
)

// ProviderModel is the small read-model returned by the Hermes config.yaml
// model/provider runtime bridge seam. The selector consumes it as a typed
// pair so it never re-derives config loading.
type ProviderModel struct {
	Provider string
	Model    string
}

// ReadInferenceDefaultsFunc is the function-shaped seam over the Hermes
// config.yaml model/provider runtime bridge row. The default ModelSelector
// calls this seam exactly once per Select to obtain the configured
// provider/model defaults.
type ReadInferenceDefaultsFunc func() (ProviderModel, error)

// ResolveAliasFunc is the function-shaped seam over the TUI launch model
// override + static alias resolver row. The default ModelSelector passes the
// configured provider plus the configured alias and expects the canonical
// model id back.
type ResolveAliasFunc func(provider, alias string) (string, error)

// ResolveAccountFunc is the function-shaped seam that maps a resolved
// provider to the account/credential id that should serve the next turn.
// Tests inject fakes; callers in production wire the credential pool.
type ResolveAccountFunc func(provider string) (string, error)

// ReadActiveProfileNameFunc is the function-shaped seam over the CLI
// active-profile store row.
type ReadActiveProfileNameFunc = profile.ReadActiveProfileNameFunc

// ValidateProfileNameFunc is the function-shaped seam over the CLI profile
// name validator row.
type ValidateProfileNameFunc func(name string) error

// ResolveProfileRootFunc is the function-shaped seam over the CLI profile
// root resolver row.
type ResolveProfileRootFunc func(name string) (string, error)

// DefaultModelSelectorOptions carries the function-shaped seams the default
// ModelSelector consumes. Each field corresponds to one of the upstream
// blocked_by rows so the selector never re-derives their logic.
type DefaultModelSelectorOptions struct {
	ReadInferenceDefaults ReadInferenceDefaultsFunc
	ResolveAlias          ResolveAliasFunc
	ResolveAccount        ResolveAccountFunc
}

// DefaultProfileSelectorOptions carries the function-shaped seams the
// default ProfileSelector consumes; one field per profile-helper row.
type DefaultProfileSelectorOptions struct {
	ReadActiveProfileName ReadActiveProfileNameFunc
	ValidateProfileName   ValidateProfileNameFunc
	ResolveProfileRoot    ResolveProfileRootFunc
}

// NewDefaultModelSelector returns the production ModelSelector wired to the
// supplied helper-row seams. Tests inject fakes; production wiring lives in
// the Cobra/gateway entry points (next slice).
func NewDefaultModelSelector(opts DefaultModelSelectorOptions) ModelSelector {
	return &defaultModelSelector{opts: opts}
}

// NewDefaultProfileSelector returns the production ProfileSelector wired to
// the supplied helper-row seams.
func NewDefaultProfileSelector(opts DefaultProfileSelectorOptions) ProfileSelector {
	return &defaultProfileSelector{opts: opts}
}

type defaultModelSelector struct {
	opts DefaultModelSelectorOptions
}

func (s *defaultModelSelector) Select(ctx context.Context, kind SelectionKind) (Selection, error) {
	if s.opts.ReadInferenceDefaults == nil {
		return Selection{}, fmt.Errorf("model selector: ReadInferenceDefaults: %w", ErrSelectorHelperUnavailable)
	}
	defaults, err := s.opts.ReadInferenceDefaults()
	if err != nil {
		return Selection{}, fmt.Errorf("model selector: read inference defaults: %w", err)
	}
	if defaults.Provider == "" && defaults.Model == "" {
		return Selection{}, fmt.Errorf("model selector: empty inference defaults: %w", ErrSelectorNoMatch)
	}

	resolvedModel := defaults.Model
	if kind == SelectionKindModel && defaults.Model != "" {
		if s.opts.ResolveAlias == nil {
			return Selection{}, fmt.Errorf("model selector: ResolveAlias: %w", ErrSelectorHelperUnavailable)
		}
		canonical, aliasErr := s.opts.ResolveAlias(defaults.Provider, defaults.Model)
		switch {
		case errors.Is(aliasErr, ErrSelectorNoMatch):
			return Selection{}, fmt.Errorf("model selector: alias %q: %w", defaults.Model, ErrSelectorNoMatch)
		case aliasErr != nil:
			return Selection{}, fmt.Errorf("model selector: %w: %w", ErrSelectorAliasResolutionFailed, aliasErr)
		}
		resolvedModel = canonical
	}

	account := ""
	if s.opts.ResolveAccount != nil {
		acct, acctErr := s.opts.ResolveAccount(defaults.Provider)
		if acctErr != nil {
			return Selection{}, fmt.Errorf("model selector: resolve account: %w", acctErr)
		}
		account = acct
	}

	return Selection{
		Provider: defaults.Provider,
		Model:    resolvedModel,
		Account:  account,
	}, nil
}

type defaultProfileSelector struct {
	opts DefaultProfileSelectorOptions
}

func (s *defaultProfileSelector) Select(ctx context.Context) (Profile, error) {
	if s.opts.ReadActiveProfileName == nil {
		return Profile{}, fmt.Errorf("profile selector: ReadActiveProfileName: %w", ErrSelectorHelperUnavailable)
	}
	name, err := s.opts.ReadActiveProfileName()
	if err != nil {
		if errors.Is(err, profile.ErrActiveProfileUnset) {
			return Profile{}, fmt.Errorf("profile selector: active profile unset: %w", ErrSelectorNoMatch)
		}
		return Profile{}, fmt.Errorf("profile selector: read active profile: %w", err)
	}

	validate := s.opts.ValidateProfileName
	if validate == nil {
		validate = profile.ValidateProfileName
	}
	if err := validate(name); err != nil {
		return Profile{}, fmt.Errorf("profile selector: validate %q: %w", name, err)
	}

	if s.opts.ResolveProfileRoot == nil {
		return Profile{}, fmt.Errorf("profile selector: ResolveProfileRoot: %w", ErrSelectorHelperUnavailable)
	}
	root, err := s.opts.ResolveProfileRoot(name)
	if err != nil {
		return Profile{}, fmt.Errorf("profile selector: resolve root for %q: %w", name, err)
	}

	return Profile{Name: name, RootPath: root}, nil
}
