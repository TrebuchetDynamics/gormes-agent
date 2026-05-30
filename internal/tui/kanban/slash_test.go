package kanban

import (
	"errors"
	"strings"
	"testing"
)

func TestHandleSlashRunsInjectedCommand(t *testing.T) {
	var gotInput string
	result := HandleSlash("/kanban list", func(input string) (string, error) {
		gotInput = input
		return " No Kanban tasks.\n", nil
	})
	if gotInput != "/kanban list" {
		t.Fatalf("runner input = %q, want full slash input", gotInput)
	}
	if result.StatusMessage != "No Kanban tasks." {
		t.Fatalf("StatusMessage = %q, want command output", result.StatusMessage)
	}
}

func TestHandleSlashFailureIncludesEvidence(t *testing.T) {
	result := HandleSlash("/kanban list", func(string) (string, error) {
		return "partial output", errors.New("kanban db unavailable")
	})
	if got, want := result.StatusMessage, "kanban: kanban db unavailable: partial output"; got != want {
		t.Fatalf("StatusMessage = %q, want %q", got, want)
	}
}

func TestHandleSlashUnavailable(t *testing.T) {
	if got := HandleSlash("/kanban list", nil).StatusMessage; got != "kanban: command runner unavailable" {
		t.Fatalf("StatusMessage = %q, want unavailable evidence", got)
	}
}

func TestBoundStatusTrimsAndBounds(t *testing.T) {
	if got := BoundStatus(" task\n done "); got != "task done" {
		t.Fatalf("BoundStatus whitespace = %q, want collapsed", got)
	}
	long := strings.Repeat("x", MaxStatusRunes+10)
	got := BoundStatus(long)
	if len([]rune(got)) != MaxStatusRunes+len("...") {
		t.Fatalf("bounded rune count = %d, want %d", len([]rune(got)), MaxStatusRunes+len("..."))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("bounded status missing suffix: %q", got)
	}
}
