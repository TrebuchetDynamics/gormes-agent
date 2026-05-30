package reset

import (
	"errors"
	"testing"
)

func TestHandleSlash(t *testing.T) {
	called := false
	res := HandleSlash("/new", false, func() error {
		called = true
		return nil
	})
	if !called || !res.Reset || res.Status != "new session started" {
		t.Fatalf("HandleSlash(new) = %+v called=%v", res, called)
	}
	if res := HandleSlash("/clear", true, func() error { t.Fatal("reset should not run while busy"); return nil }); res.Reset || res.Status != "interrupt the current turn before trying to switch sessions" {
		t.Fatalf("HandleSlash(busy) = %+v", res)
	}
	if res := HandleSlash("/new", false, nil); res.Reset || res.Status != "new: reset unavailable" {
		t.Fatalf("HandleSlash(unavailable) = %+v", res)
	}
	if res := HandleSlash("/clear", false, func() error { return errors.New("boom") }); res.Reset || res.Status != "clear: reset failed: boom" {
		t.Fatalf("HandleSlash(error) = %+v", res)
	}
}

func TestKind(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: KindClear},
		{input: "   ", want: KindClear},
		{input: "/clear", want: KindClear},
		{input: "clear", want: KindClear},
		{input: "/new", want: KindNew},
		{input: "new", want: KindNew},
		{input: "/NEW", want: KindNew},
		{input: "/unknown", want: KindClear},
	}
	for _, tt := range tests {
		if got := Kind(tt.input); got != tt.want {
			t.Fatalf("Kind(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
