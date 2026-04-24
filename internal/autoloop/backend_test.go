package autoloop

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildBackendCommandCodexuSafe(t *testing.T) {
	got, err := BuildBackendCommand("codexu", "safe")
	if err != nil {
		t.Fatalf("BuildBackendCommand() error = %v", err)
	}

	want := []string{"codexu", "exec", "--json", "-m", "gpt-5.5", "-c", "approval_policy=never", "--sandbox", "workspace-write"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildBackendCommand() = %#v, want %#v", got, want)
	}
}

func TestBuildBackendCommandCodexuFull(t *testing.T) {
	got, err := BuildBackendCommand("codexu", "full")
	if err != nil {
		t.Fatalf("BuildBackendCommand() error = %v", err)
	}

	want := []string{"codexu", "exec", "--json", "-m", "gpt-5.5", "-c", "approval_policy=never", "--sandbox", "danger-full-access"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildBackendCommand() = %#v, want %#v", got, want)
	}
}

func TestBuildBackendCommandClaudeuUsesShimShape(t *testing.T) {
	got, err := BuildBackendCommand("claudeu", "safe")
	if err != nil {
		t.Fatalf("BuildBackendCommand() error = %v", err)
	}

	wantPrefix := []string{"claudeu", "exec", "--json"}
	if len(got) < len(wantPrefix) || !reflect.DeepEqual(got[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("BuildBackendCommand() = %#v, want prefix %#v", got, wantPrefix)
	}
}

func TestBuildBackendCommandRejectsInvalidMode(t *testing.T) {
	_, err := BuildBackendCommand("codexu", "turbo")
	if err == nil {
		t.Fatal("BuildBackendCommand() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "invalid MODE") {
		t.Fatalf("BuildBackendCommand() error = %q, want invalid MODE message", err)
	}
}
