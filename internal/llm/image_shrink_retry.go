package llm

// ImageShrinker re-encodes a content-part list at a smaller image size for
// retry after a provider image_too_large rejection. Implementations must
// not mutate the input slice; they should return a new slice (which may
// share unchanged parts by value).
//
// This is the pure-function seam for the bounded image-shrink retry
// planner. Real Pillow-like image decoding lives behind this callback so
// PlanImageShrinkRetry can stay deterministic and IO-free.
type ImageShrinker func(parts []MessageContentPart) ([]MessageContentPart, error)

// ImageShrinkPlan is the planner's verdict for one provider response.
//
// When Retry is true, NewParts is the rebuilt content-part list to send on
// the next attempt and EvidenceCode is image_shrink_planned. When Retry is
// false, EvidenceCode names the reason no retry is permitted and NewParts
// is nil.
type ImageShrinkPlan struct {
	Retry        bool
	NewParts     []MessageContentPart
	EvidenceCode string
}

// ImageShrinkRequest packages the inputs PlanImageShrinkRetry needs to
// decide whether one bounded retry should be attempted for an
// image_too_large failure.
//
// Attempts counts how many shrink retries have already been issued for the
// current message set. The planner permits at most one automatic retry, so
// any value greater than zero short-circuits to image_shrink_limit_reached.
//
// Shrinker may be nil when the caller has no image-resize dependency
// available; the planner then reports image_shrink_unavailable.
type ImageShrinkRequest struct {
	Kind     ProviderErrorKind
	Parts    []MessageContentPart
	Attempts int
	Shrinker ImageShrinker
}

// Evidence codes published by PlanImageShrinkRetry. They mirror the
// provider-status vocabulary in the progress.json row so callers can
// surface them in degraded-mode reports without reaching into the planner.
const (
	imageShrinkEvidencePlanned      = "image_shrink_planned"
	imageShrinkEvidenceUnavailable  = "image_shrink_unavailable"
	imageShrinkEvidenceNoImages     = "image_shrink_no_images"
	imageShrinkEvidenceLimitReached = "image_shrink_limit_reached"
	imageShrinkEvidenceFailed       = "image_shrink_failed"
)

// PlanImageShrinkRetry returns the bounded shrink-retry plan for a single
// provider failure. It is pure: no goroutines, no IO, no global state.
//
// Behavior:
//   - non-image_too_large kinds return Retry=false with empty evidence so
//     callers route them through their existing classification logic;
//   - already-attempted shrinks (Attempts > 0) return image_shrink_limit_reached;
//   - missing shrinker returns image_shrink_unavailable;
//   - parts without any image_url entry return image_shrink_no_images;
//   - shrinker errors return image_shrink_failed;
//   - otherwise the shrinker is invoked exactly once and its output is
//     packaged with image_shrink_planned.
//
// The original Parts slice is never mutated.
func PlanImageShrinkRetry(req ImageShrinkRequest) ImageShrinkPlan {
	if req.Kind != ProviderErrorImageTooLarge {
		return ImageShrinkPlan{}
	}
	if req.Attempts > 0 {
		return ImageShrinkPlan{EvidenceCode: imageShrinkEvidenceLimitReached}
	}
	if req.Shrinker == nil {
		return ImageShrinkPlan{EvidenceCode: imageShrinkEvidenceUnavailable}
	}
	if !hasImagePart(req.Parts) {
		return ImageShrinkPlan{EvidenceCode: imageShrinkEvidenceNoImages}
	}

	// Hand the shrinker a defensive copy so it cannot mutate the caller's
	// slice even if its implementation is careless.
	input := append([]MessageContentPart(nil), req.Parts...)
	resized, err := req.Shrinker(input)
	if err != nil {
		return ImageShrinkPlan{EvidenceCode: imageShrinkEvidenceFailed}
	}
	return ImageShrinkPlan{
		Retry:        true,
		NewParts:     resized,
		EvidenceCode: imageShrinkEvidencePlanned,
	}
}

func hasImagePart(parts []MessageContentPart) bool {
	for _, p := range parts {
		if p.Type == "image_url" && p.ImageURL != "" {
			return true
		}
	}
	return false
}
