package manifest

import (
	"errors"
	"testing"
)

func TestManifestFacadePreservesPublicContract(t *testing.T) {
	fixture, err := LoadUpstreamToolParityManifest()
	if err != nil {
		t.Fatalf("LoadUpstreamToolParityManifest: %v", err)
	}
	if _, ok := fixture.Tool("todo"); !ok {
		t.Fatal("facade fixture missing todo tool")
	}
	if err := fixture.AssertHandlerPortAllowed("future_tool_without_descriptor"); !errors.Is(err, ErrMissingToolParityRow) {
		t.Fatalf("unknown tool error = %v, want ErrMissingToolParityRow", err)
	}
	if len(RawUpstreamToolParityManifestJSON()) == 0 {
		t.Fatal("facade raw manifest JSON is empty")
	}
}
