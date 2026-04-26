package builderloop

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestClassifyBackendFailurePrioritizesKilledOverStdinNoise(t *testing.T) {
	failure := ClassifyBackendFailure(errors.New("signal: killed"), "Reading additional input from stdin...\n", "")

	if failure.Status != "backend_killed" {
		t.Fatalf("Status = %q, want backend_killed", failure.Status)
	}
	for _, want := range []string{"backend_killed", "signal: killed", "Reading additional input from stdin"} {
		if !strings.Contains(failure.Detail, want) {
			t.Fatalf("Detail = %q, want containing %q", failure.Detail, want)
		}
	}
}

func TestClassifyBackendFailureDetectsStdinWaitWithoutProcessKill(t *testing.T) {
	failure := ClassifyBackendFailure(errors.New("exit status 1"), "Reading additional input from stdin...\n", "")

	if failure.Status != "backend_waiting_for_stdin" {
		t.Fatalf("Status = %q, want backend_waiting_for_stdin", failure.Status)
	}
	if !strings.Contains(failure.Detail, "backend_waiting_for_stdin") {
		t.Fatalf("Detail = %q, want classified prefix", failure.Detail)
	}
}

func TestClassifyBackendFailureDetectsNoProgressTimeout(t *testing.T) {
	failure := ClassifyBackendFailure(context.DeadlineExceeded, "", "")

	if failure.Status != "backend_no_progress" {
		t.Fatalf("Status = %q, want backend_no_progress", failure.Status)
	}
	if !strings.Contains(failure.Detail, context.DeadlineExceeded.Error()) {
		t.Fatalf("Detail = %q, want deadline detail", failure.Detail)
	}
}
