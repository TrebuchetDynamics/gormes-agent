package hermes

import (
	"errors"
	"testing"
)

// fakeShrinker swaps every image_url part with a part marked "shrunk_<n>" so
// tests can prove the planner forwarded the parts to the shrinker exactly
// once without depending on real image bytes.
type fakeShrinker struct {
	calls int
	err   error
}

func (s *fakeShrinker) shrink(parts []MessageContentPart) ([]MessageContentPart, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	out := make([]MessageContentPart, len(parts))
	for i, p := range parts {
		if p.Type == "image_url" {
			p.ImageURL = "data:image/jpeg;base64,SHRUNK"
		}
		out[i] = p
	}
	return out, nil
}

func partsWithOneImage() []MessageContentPart {
	return []MessageContentPart{
		{Type: "text", Text: "describe this"},
		{Type: "image_url", ImageURL: "data:image/png;base64,AAAABIGPAYLOAD"},
	}
}

func TestImageShrinkRetry_PlansOneRetryForImageTooLarge(t *testing.T) {
	shrinker := &fakeShrinker{}
	in := partsWithOneImage()

	plan := PlanImageShrinkRetry(ImageShrinkRequest{
		Kind:     ProviderErrorImageTooLarge,
		Parts:    in,
		Attempts: 0,
		Shrinker: shrinker.shrink,
	})

	if !plan.Retry {
		t.Fatalf("expected Retry=true for image_too_large with shrinker, got false (evidence=%q)", plan.EvidenceCode)
	}
	if plan.EvidenceCode != "image_shrink_planned" {
		t.Fatalf("expected EvidenceCode=image_shrink_planned, got %q", plan.EvidenceCode)
	}
	if shrinker.calls != 1 {
		t.Fatalf("expected exactly 1 shrinker call, got %d", shrinker.calls)
	}
	if len(plan.NewParts) != len(in) {
		t.Fatalf("expected NewParts to mirror input length %d, got %d", len(in), len(plan.NewParts))
	}
	// New parts carry the shrunk payload.
	foundShrunk := false
	for _, p := range plan.NewParts {
		if p.Type == "image_url" && p.ImageURL == "data:image/jpeg;base64,SHRUNK" {
			foundShrunk = true
		}
	}
	if !foundShrunk {
		t.Fatalf("expected NewParts to contain shrunk image, got %+v", plan.NewParts)
	}
	// Original input must not be mutated.
	if in[1].ImageURL != "data:image/png;base64,AAAABIGPAYLOAD" {
		t.Fatalf("planner mutated original parts: %+v", in)
	}
}

func TestImageShrinkRetry_DoesNotRetryOtherProviderKinds(t *testing.T) {
	otherKinds := []ProviderErrorKind{
		ProviderErrorContext,
		ProviderErrorRateLimit,
		ProviderErrorAuth,
		ProviderErrorRetryable,
		ProviderErrorUnknown,
		ProviderErrorNonRetryable,
	}
	for _, kind := range otherKinds {
		t.Run(string(kind), func(t *testing.T) {
			shrinker := &fakeShrinker{}
			plan := PlanImageShrinkRetry(ImageShrinkRequest{
				Kind:     kind,
				Parts:    partsWithOneImage(),
				Attempts: 0,
				Shrinker: shrinker.shrink,
			})
			if plan.Retry {
				t.Fatalf("expected Retry=false for kind %s, got true", kind)
			}
			if shrinker.calls != 0 {
				t.Fatalf("expected shrinker not called for kind %s, got %d calls", kind, shrinker.calls)
			}
			if plan.NewParts != nil {
				t.Fatalf("expected NewParts=nil for non-image_too_large kind %s, got %+v", kind, plan.NewParts)
			}
		})
	}
}

func TestImageShrinkRetry_NoImagesReturnsEvidence(t *testing.T) {
	shrinker := &fakeShrinker{}
	textOnly := []MessageContentPart{{Type: "text", Text: "no images here"}}

	plan := PlanImageShrinkRetry(ImageShrinkRequest{
		Kind:     ProviderErrorImageTooLarge,
		Parts:    textOnly,
		Attempts: 0,
		Shrinker: shrinker.shrink,
	})

	if plan.Retry {
		t.Fatalf("expected Retry=false when no image parts, got true")
	}
	if plan.EvidenceCode != "image_shrink_no_images" {
		t.Fatalf("expected EvidenceCode=image_shrink_no_images, got %q", plan.EvidenceCode)
	}
	if shrinker.calls != 0 {
		t.Fatalf("expected shrinker not called when no images, got %d calls", shrinker.calls)
	}
}

func TestImageShrinkRetry_LimitReachedAfterOneAttempt(t *testing.T) {
	shrinker := &fakeShrinker{}

	plan := PlanImageShrinkRetry(ImageShrinkRequest{
		Kind:     ProviderErrorImageTooLarge,
		Parts:    partsWithOneImage(),
		Attempts: 1, // already shrunk once
		Shrinker: shrinker.shrink,
	})

	if plan.Retry {
		t.Fatalf("expected Retry=false after one attempt, got true")
	}
	if plan.EvidenceCode != "image_shrink_limit_reached" {
		t.Fatalf("expected EvidenceCode=image_shrink_limit_reached, got %q", plan.EvidenceCode)
	}
	if shrinker.calls != 0 {
		t.Fatalf("expected shrinker not called after limit reached, got %d calls", shrinker.calls)
	}
}

func TestImageShrinkRetry_ShrinkerFailureReturnsEvidence(t *testing.T) {
	shrinker := &fakeShrinker{err: errors.New("decode failed")}

	plan := PlanImageShrinkRetry(ImageShrinkRequest{
		Kind:     ProviderErrorImageTooLarge,
		Parts:    partsWithOneImage(),
		Attempts: 0,
		Shrinker: shrinker.shrink,
	})

	if plan.Retry {
		t.Fatalf("expected Retry=false on shrinker failure, got true")
	}
	if plan.EvidenceCode != "image_shrink_failed" {
		t.Fatalf("expected EvidenceCode=image_shrink_failed, got %q", plan.EvidenceCode)
	}
	if shrinker.calls != 1 {
		t.Fatalf("expected shrinker called once before reporting failure, got %d", shrinker.calls)
	}
	if plan.NewParts != nil {
		t.Fatalf("expected NewParts=nil on shrinker failure, got %+v", plan.NewParts)
	}
}

func TestImageShrinkRetry_NilShrinkerReturnsUnavailable(t *testing.T) {
	plan := PlanImageShrinkRetry(ImageShrinkRequest{
		Kind:     ProviderErrorImageTooLarge,
		Parts:    partsWithOneImage(),
		Attempts: 0,
		Shrinker: nil,
	})

	if plan.Retry {
		t.Fatalf("expected Retry=false with nil shrinker, got true")
	}
	if plan.EvidenceCode != "image_shrink_unavailable" {
		t.Fatalf("expected EvidenceCode=image_shrink_unavailable, got %q", plan.EvidenceCode)
	}
}
