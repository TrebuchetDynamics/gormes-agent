package modelselection

import (
	"context"
	"reflect"
	"testing"
)

func TestPublicFacadePreservesModelSelectionContracts(t *testing.T) {
	var modelSelector ModelSelector = ModelSelectorFunc(func(ctx context.Context, kind SelectionKind) (Selection, error) {
		return Selection{Provider: "anthropic", Model: "claude-sonnet-4", Account: "default"}, nil
	})
	got, err := modelSelector.Select(context.Background(), SelectionKindModel)
	if err != nil {
		t.Fatalf("ModelSelector.Select: unexpected error: %v", err)
	}
	want := Selection{Provider: "anthropic", Model: "claude-sonnet-4", Account: "default"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ModelSelector.Select: got %+v want %+v", got, want)
	}

	entries := HermesModelProviderMenu()
	if len(entries) == 0 || entries[0].ID == "" || entries[0].Label == "" {
		t.Fatalf("HermesModelProviderMenu returned invalid entries: %#v", entries)
	}
}
