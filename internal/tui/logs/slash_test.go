package logs

import (
	"errors"
	"testing"
)

func TestHandleSlashOpensLogsOrReportsEvidence(t *testing.T) {
	var gotLimit int
	res := HandleSlash("/logs 7", func(limit int) (string, error) {
		gotLimit = limit
		return "line 1\nline 2\n", nil
	})
	if gotLimit != 7 || !res.Open || res.Title != "Logs" || res.Body != "line 1\nline 2" || res.Status != "logs opened" {
		t.Fatalf("HandleSlash(open) = %+v limit=%d", res, gotLimit)
	}
	if res := HandleSlash("/logs", nil); res.Open || res.Status != "no gateway logs" {
		t.Fatalf("HandleSlash(nil tail) = %+v", res)
	}
	if res := HandleSlash("/logs", func(int) (string, error) { return "\n", nil }); res.Open || res.Status != "no gateway logs" {
		t.Fatalf("HandleSlash(empty) = %+v", res)
	}
	if res := HandleSlash("/logs", func(int) (string, error) { return "", errors.New("missing") }); res.Open || res.Status != "logs: missing" {
		t.Fatalf("HandleSlash(error) = %+v", res)
	}
}

func TestTailLimitDefaultsAndClamps(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{input: "/logs", want: 20},
		{input: "/logs 0", want: 20},
		{input: "/logs -2", want: 1},
		{input: "/logs 7", want: 7},
		{input: "/logs 500", want: 80},
		{input: "/logs nope", want: 20},
	}
	for _, tc := range cases {
		if got := TailLimit(tc.input); got != tc.want {
			t.Fatalf("TailLimit(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
