package kernel

import (
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// Live regression 2026-05-09: a Telegram screenshot (243KB JPEG, ~325KB
// base64 data URI) caused validateTurnAdmission to return ErrInputTooLarge
// because image_url payloads were being summed into the same MaxBytes
// budget as user text. Image bytes are governed by the provider-side
// image_shrink_retry path; admission must gate text only.
func TestValidateTurnAdmission_ImageURLNotCountedAgainstByteLimit(t *testing.T) {
	admission := Admission{MaxBytes: 1024, MaxLines: 100}
	hugeDataURI := "data:image/jpeg;base64," + strings.Repeat("A", 500_000)

	err := validateTurnAdmission(admission, "what is in this picture?", []hermes.MessageContentPart{
		{Type: "image_url", ImageURL: hugeDataURI},
	})
	if err != nil {
		t.Fatalf("expected admission to pass when only image_url payload exceeds MaxBytes (image size governed by image_shrink_retry, not admission), got %v", err)
	}
}

func TestValidateTurnAdmission_ImageOnlyTurnPassesEmptyCheck(t *testing.T) {
	admission := Admission{MaxBytes: 1024, MaxLines: 100}
	dataURI := "data:image/jpeg;base64," + strings.Repeat("A", 100)

	err := validateTurnAdmission(admission, "", []hermes.MessageContentPart{
		{Type: "image_url", ImageURL: dataURI},
	})
	if err != nil {
		t.Fatalf("image-only turn (no text, one image_url) should pass admission, got %v", err)
	}
}

func TestValidateTurnAdmission_TextStillBoundedByMaxBytes(t *testing.T) {
	admission := Admission{MaxBytes: 32, MaxLines: 100}
	tooLong := strings.Repeat("A", 64)

	err := validateTurnAdmission(admission, tooLong, nil)
	if !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("text exceeding MaxBytes must still trip ErrInputTooLarge, got %v", err)
	}
}

func TestValidateTurnAdmission_TextPlusImagePartCountsTextOnly(t *testing.T) {
	admission := Admission{MaxBytes: 32, MaxLines: 100}
	hugeDataURI := "data:image/jpeg;base64," + strings.Repeat("A", 500_000)

	// Text fits within MaxBytes (10 chars + newline + ~10 = under 32). Image
	// payload is huge but must not count.
	err := validateTurnAdmission(admission, "short text", []hermes.MessageContentPart{
		{Type: "text", Text: "more text"},
		{Type: "image_url", ImageURL: hugeDataURI},
	})
	if err != nil {
		t.Fatalf("text-plus-image should pass when text fits, got %v", err)
	}
}

func TestValidateTurnAdmission_EmptyInputStillRejected(t *testing.T) {
	admission := Admission{MaxBytes: 1024, MaxLines: 100}
	if err := validateTurnAdmission(admission, "", nil); !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("empty input should still trip ErrEmptyInput, got %v", err)
	}
	if err := validateTurnAdmission(admission, "   ", nil); !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("whitespace-only input should still trip ErrEmptyInput, got %v", err)
	}
}
