package bedrock

import (
	"errors"
	"testing"
)

func TestCompatibilityFacade_AuthAndStaleRuntimeAPIs(t *testing.T) {
	evidence := ResolveBedrockAuth(map[string]string{
		"AWS_ACCESS_KEY_ID":     "AKIA_TEST",
		"AWS_SECRET_ACCESS_KEY": "secret-test-value",
	})
	if evidence.Source != "AWS_ACCESS_KEY_ID" || evidence.State != BedrockAuthStatePresent {
		t.Fatalf("ResolveBedrockAuth() = %+v, want static key evidence", evidence)
	}
	if got := ResolveBedrockRegion(map[string]string{"AWS_DEFAULT_REGION": "us-west-2"}); got != "us-west-2" {
		t.Fatalf("ResolveBedrockRegion() = %q, want us-west-2", got)
	}

	classification := ClassifyBedrockStaleError(errors.Join(ErrBedrockReadTimeout))
	if !classification.Stale || classification.Status != BedrockStaleTransportStatus {
		t.Fatalf("ClassifyBedrockStaleError() = %+v, want stale transport", classification)
	}

	cache := NewBedrockRuntimeCache(nil)
	if _, err := cache.Converse(t.Context(), "us-east-1", BedrockRuntimeRequest{Model: "test"}); !errors.Is(err, ErrBedrockRuntimeClientMissing) {
		t.Fatalf("Converse() error = %v, want ErrBedrockRuntimeClientMissing", err)
	}
}
