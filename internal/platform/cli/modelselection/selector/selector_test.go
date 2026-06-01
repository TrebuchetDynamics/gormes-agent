package selector

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile"
)

// TestModelSelectorContract proves the public Go contract: the package
// exports ModelSelector and ProfileSelector interfaces with the documented
// Select signatures and the matching small result records.
func TestModelSelectorContract(t *testing.T) {
	var modelSelector ModelSelector = ModelSelectorFunc(func(ctx context.Context, kind SelectionKind) (Selection, error) {
		return Selection{Provider: "anthropic", Model: "claude-sonnet-4", Account: "default"}, nil
	})
	var profileSelector ProfileSelector = ProfileSelectorFunc(func(ctx context.Context) (Profile, error) {
		return Profile{Name: "main", RootPath: "/tmp/gormes/profiles/main"}, nil
	})

	got, err := modelSelector.Select(context.Background(), SelectionKindModel)
	if err != nil {
		t.Fatalf("ModelSelector.Select: unexpected error: %v", err)
	}
	want := Selection{Provider: "anthropic", Model: "claude-sonnet-4", Account: "default"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ModelSelector.Select: got %+v want %+v", got, want)
	}

	gotProfile, err := profileSelector.Select(context.Background())
	if err != nil {
		t.Fatalf("ProfileSelector.Select: unexpected error: %v", err)
	}
	wantProfile := Profile{Name: "main", RootPath: "/tmp/gormes/profiles/main"}
	if !reflect.DeepEqual(gotProfile, wantProfile) {
		t.Fatalf("ProfileSelector.Select: got %+v want %+v", gotProfile, wantProfile)
	}

	// Selection kind constants must exist and be distinct so callers can
	// branch the model/provider/account selection request without parsing
	// strings.
	if SelectionKindModel == SelectionKindProvider {
		t.Fatal("SelectionKindModel and SelectionKindProvider must be distinct")
	}
	if SelectionKindAccount == SelectionKindModel {
		t.Fatal("SelectionKindAccount and SelectionKindModel must be distinct")
	}
}

// TestModelSelectorDefaultConsumesExistingResolverRows proves the default
// implementation reaches into the Hermes config.yaml model/provider runtime
// bridge (via an injected config-read seam) and the static alias resolver
// (via an injected alias-resolve seam) instead of re-deriving alias logic.
func TestModelSelectorDefaultConsumesExistingResolverRows(t *testing.T) {
	configCalls := 0
	aliasCalls := 0
	var lastAliasInput string

	sel := NewDefaultModelSelector(DefaultModelSelectorOptions{
		ReadInferenceDefaults: func() (ProviderModel, error) {
			configCalls++
			return ProviderModel{Provider: "anthropic", Model: "sonnet"}, nil
		},
		ResolveAlias: func(provider, alias string) (string, error) {
			aliasCalls++
			lastAliasInput = alias
			if alias == "sonnet" {
				return "claude-sonnet-4-20250514", nil
			}
			return "", ErrSelectorNoMatch
		},
		ResolveAccount: func(provider string) (string, error) {
			return "default", nil
		},
	})

	got, err := sel.Select(context.Background(), SelectionKindModel)
	if err != nil {
		t.Fatalf("default ModelSelector.Select: unexpected error: %v", err)
	}
	if configCalls != 1 {
		t.Fatalf("default ModelSelector should call ReadInferenceDefaults exactly once, got %d", configCalls)
	}
	if aliasCalls != 1 {
		t.Fatalf("default ModelSelector should call ResolveAlias exactly once, got %d", aliasCalls)
	}
	if lastAliasInput != "sonnet" {
		t.Fatalf("default ModelSelector should pass the configured alias (%q), got %q", "sonnet", lastAliasInput)
	}
	want := Selection{Provider: "anthropic", Model: "claude-sonnet-4-20250514", Account: "default"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default ModelSelector.Select: got %+v want %+v", got, want)
	}
}

// TestProfileSelectorDefaultConsumesProfileHelpers proves the default
// implementation invokes the profile name validator → root resolver →
// active-profile store helpers in that order, with the active store's
// returned name flowing into the validator.
func TestProfileSelectorDefaultConsumesProfileHelpers(t *testing.T) {
	var calls []string

	sel := NewDefaultProfileSelector(DefaultProfileSelectorOptions{
		ReadActiveProfileName: func() (string, error) {
			calls = append(calls, "read")
			return "myprof", nil
		},
		ValidateProfileName: func(name string) error {
			calls = append(calls, "validate:"+name)
			return profile.ValidateProfileName(name)
		},
		ResolveProfileRoot: func(name string) (string, error) {
			calls = append(calls, "resolve:"+name)
			return profile.ResolveProfileRoot(name, filepath.Join("/tmp", "xdg"))
		},
	})

	got, err := sel.Select(context.Background())
	if err != nil {
		t.Fatalf("default ProfileSelector.Select: unexpected error: %v", err)
	}

	wantCalls := []string{"read", "validate:myprof", "resolve:myprof"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("default ProfileSelector helper order: got %v want %v", calls, wantCalls)
	}

	wantProfile := Profile{Name: "myprof", RootPath: "/tmp/xdg/gormes/profiles/myprof"}
	if !reflect.DeepEqual(got, wantProfile) {
		t.Fatalf("default ProfileSelector.Select: got %+v want %+v", got, wantProfile)
	}
}

// TestSelectorReturnsTypedErrorOnNoMatch proves both selectors surface the
// canonical typed evidence (selector_no_match, selector_helper_unavailable,
// selector_alias_resolution_failed) instead of panicking, returning nil, or
// emitting unstructured fmt.Errorf wrappers.
func TestSelectorReturnsTypedErrorOnNoMatch(t *testing.T) {
	t.Run("model_no_match", func(t *testing.T) {
		sel := NewDefaultModelSelector(DefaultModelSelectorOptions{
			ReadInferenceDefaults: func() (ProviderModel, error) {
				// No configured model: nothing for the selector to match against.
				return ProviderModel{}, nil
			},
			ResolveAlias: func(provider, alias string) (string, error) {
				return "", ErrSelectorNoMatch
			},
			ResolveAccount: func(provider string) (string, error) {
				return "", nil
			},
		})
		got, err := sel.Select(context.Background(), SelectionKindModel)
		if err == nil {
			t.Fatalf("expected error, got nil result %+v", got)
		}
		if !errors.Is(err, ErrSelectorNoMatch) {
			t.Fatalf("expected ErrSelectorNoMatch, got %v", err)
		}
		if got != (Selection{}) {
			t.Fatalf("expected zero Selection on no-match, got %+v", got)
		}
	})

	t.Run("model_helper_unavailable", func(t *testing.T) {
		sel := NewDefaultModelSelector(DefaultModelSelectorOptions{
			// No ReadInferenceDefaults injected: the helper row is not wired.
		})
		_, err := sel.Select(context.Background(), SelectionKindModel)
		if !errors.Is(err, ErrSelectorHelperUnavailable) {
			t.Fatalf("expected ErrSelectorHelperUnavailable, got %v", err)
		}
	})

	t.Run("model_alias_resolution_failed", func(t *testing.T) {
		boom := errors.New("alias backend offline")
		sel := NewDefaultModelSelector(DefaultModelSelectorOptions{
			ReadInferenceDefaults: func() (ProviderModel, error) {
				return ProviderModel{Provider: "anthropic", Model: "sonnet"}, nil
			},
			ResolveAlias: func(provider, alias string) (string, error) {
				return "", boom
			},
		})
		_, err := sel.Select(context.Background(), SelectionKindModel)
		if !errors.Is(err, ErrSelectorAliasResolutionFailed) {
			t.Fatalf("expected ErrSelectorAliasResolutionFailed, got %v", err)
		}
		if !errors.Is(err, boom) {
			t.Fatalf("expected wrapped backend error, got %v", err)
		}
	})

	t.Run("profile_no_match", func(t *testing.T) {
		sel := NewDefaultProfileSelector(DefaultProfileSelectorOptions{
			ReadActiveProfileName: func() (string, error) {
				return "", profile.ErrActiveProfileUnset
			},
		})
		_, err := sel.Select(context.Background())
		if !errors.Is(err, ErrSelectorNoMatch) {
			t.Fatalf("expected ErrSelectorNoMatch when active profile is unset, got %v", err)
		}
	})

	t.Run("profile_helper_unavailable", func(t *testing.T) {
		sel := NewDefaultProfileSelector(DefaultProfileSelectorOptions{
			// No ReadActiveProfileName injected.
		})
		_, err := sel.Select(context.Background())
		if !errors.Is(err, ErrSelectorHelperUnavailable) {
			t.Fatalf("expected ErrSelectorHelperUnavailable, got %v", err)
		}
	})
}
