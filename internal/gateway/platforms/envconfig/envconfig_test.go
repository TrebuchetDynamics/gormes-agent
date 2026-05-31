package envconfig

import (
	"testing"
	"time"
)

func TestFloatSecondsRejectsNonFiniteValues(t *testing.T) {
	fallback := 3 * time.Second
	for _, raw := range []string{"NaN", "+Inf", "-Inf", "100000000000000000000"} {
		t.Run("positive/"+raw, func(t *testing.T) {
			if got := PositiveFloatSeconds(raw, fallback); got != fallback {
				t.Fatalf("PositiveFloatSeconds(%q) = %s, want fallback %s", raw, got, fallback)
			}
		})
		t.Run("non-negative/"+raw, func(t *testing.T) {
			if got := NonNegativeFloatSeconds(raw, fallback); got != fallback {
				t.Fatalf("NonNegativeFloatSeconds(%q) = %s, want fallback %s", raw, got, fallback)
			}
		})
	}
}

func TestNonNegativeFloatSecondsClampsFiniteNegativeValues(t *testing.T) {
	if got := NonNegativeFloatSeconds("-0.25", time.Second); got != 0 {
		t.Fatalf("NonNegativeFloatSeconds(-0.25) = %s, want 0", got)
	}
}
