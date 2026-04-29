package hermes

import (
	"net/http"
	"testing"
)

func TestClassifyProviderError_ImageTooLargeBodyHints(t *testing.T) {
	bodies := []string{
		`{"error":{"message":"image too large"}}`,
		`{"error":{"message":"image payload too large"}}`,
		`{"error":{"message":"max image size 5MB exceeded"}}`,
		`{"error":{"message":"unsupported image dimensions"}}`,
	}
	for _, body := range bodies {
		err := newHTTPError(http.StatusBadRequest, body, http.Header{})
		got := ClassifyProviderError(err)
		if got.Kind != ProviderErrorImageTooLarge {
			t.Errorf("body %q: Kind = %q, want %q", body, got.Kind, ProviderErrorImageTooLarge)
		}
	}
}

func TestClassifyProviderError_ImageTooLargeStatusAndBodyWins(t *testing.T) {
	// 413 Payload Too Large with an image-specific body: must classify as
	// image_too_large, not generic context overflow.
	err := newHTTPError(http.StatusRequestEntityTooLarge, `{"error":{"message":"image payload too large"}}`, http.Header{})
	got := ClassifyProviderError(err)
	if got.Kind != ProviderErrorImageTooLarge {
		t.Fatalf("Kind = %q, want %q (image hint must win over context)", got.Kind, ProviderErrorImageTooLarge)
	}
	if got.ShouldCompress {
		t.Fatal("ShouldCompress = true, want false for image_too_large (compression is for text context overflow)")
	}
	if got.Retryable {
		t.Fatal("Retryable = true, want false (image_too_large needs shrink, not blind retry)")
	}
}

func TestClassifyProviderError_TextContextStillContext(t *testing.T) {
	err := newHTTPError(http.StatusBadRequest, `{"error":{"message":"context length exceeded: prompt is too long"}}`, http.Header{})
	got := ClassifyProviderError(err)
	if got.Kind != ProviderErrorContext {
		t.Fatalf("Kind = %q, want %q (text context must not be reclassified as image_too_large)", got.Kind, ProviderErrorContext)
	}
	if !got.ShouldCompress {
		t.Fatal("ShouldCompress = false, want true for text context overflow")
	}
}
