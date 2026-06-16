package placeholder

import "testing"

func TestRunningPlaceholderIdleUsesHermesCopy(t *testing.T) {
	if got := RunningPlaceholder(false, []string{"queue"}); got != IdleEditorPlaceholder {
		t.Fatalf("RunningPlaceholder(idle) = %q, want %q", got, IdleEditorPlaceholder)
	}
}

func TestRunningPlaceholderBusyListsSlashAffordances(t *testing.T) {
	got := RunningPlaceholder(true, []string{"queue", "/steer", ""})
	want := "msg=interrupt · /queue · /steer · Ctrl+C cancel"
	if got != want {
		t.Fatalf("RunningPlaceholder(busy) = %q, want %q", got, want)
	}
}

func TestRunningPlaceholderBusyMinimum(t *testing.T) {
	got := RunningPlaceholder(true, nil)
	want := "msg=interrupt · Ctrl+C cancel"
	if got != want {
		t.Fatalf("RunningPlaceholder(busy nil) = %q, want %q", got, want)
	}
}
