package kanbancmd

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunSlashNormalizesInput(t *testing.T) {
	var gotInput string
	out, err := RunSlash(context.Background(), func(_ context.Context, input string) (string, error) {
		gotInput = input
		return "ok", nil
	}, "  /kanban list  ")
	if err != nil {
		t.Fatalf("RunSlash err = %v", err)
	}
	if out != "ok" || gotInput != "/kanban list" {
		t.Fatalf("out/input = %q/%q, want ok//kanban list", out, gotInput)
	}
}

func TestRunSlashDefaultsEmptyInput(t *testing.T) {
	var gotInput string
	_, err := RunSlash(context.Background(), func(_ context.Context, input string) (string, error) {
		gotInput = input
		return "help", nil
	}, "")
	if err != nil {
		t.Fatalf("RunSlash err = %v", err)
	}
	if gotInput != "/kanban" {
		t.Fatalf("runner input = %q, want /kanban", gotInput)
	}
}

func TestRunSlashRequiresRunner(t *testing.T) {
	_, err := RunSlash(context.Background(), nil, "/kanban")
	if err == nil || !strings.Contains(err.Error(), "runner unavailable") {
		t.Fatalf("RunSlash nil err = %v, want runner unavailable", err)
	}
}

func TestBoundOutput(t *testing.T) {
	if got := BoundOutput("  \n "); got != "(no output)" {
		t.Fatalf("empty output = %q", got)
	}
	if got := BoundOutput(" ok \n"); got != "ok" {
		t.Fatalf("trimmed output = %q", got)
	}
	long := strings.Repeat("x", MaxOutputBytes+10)
	got := BoundOutput(long)
	if len(got) <= MaxOutputBytes || !strings.Contains(got, "gormes kanban") || strings.Contains(got, "hermes kanban") {
		t.Fatalf("bounded output missing truncation guidance: len=%d text=%q", len(got), got[len(got)-80:])
	}
}

func TestRunSlashPropagatesRunnerError(t *testing.T) {
	want := errors.New("sqlite failed")
	_, err := RunSlash(context.Background(), func(context.Context, string) (string, error) { return "partial", want }, "/kanban")
	if !errors.Is(err, want) {
		t.Fatalf("RunSlash err = %v, want %v", err, want)
	}
}
