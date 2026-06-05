package gormescli

import (
	"strings"
	"testing"
)

// TestCrashStderrMessageIncludesPanicExcerpt guards against a UX
// regression where `gormes crashed — log at <path>` is the only thing
// printed on stderr after a panic, forcing operators to open the log
// file just to see the panic message. For one-line panics (e.g. the
// `dashboard --help` flag-shorthand collision discovered during install
// testing), the panic value alone is enough to diagnose the issue.
//
// The contract: the crash stderr message must include both the panic
// excerpt AND the path to the full log so the operator can dig deeper
// when the excerpt is truncated.
func TestCrashStderrMessageIncludesPanicExcerpt(t *testing.T) {
	cases := []struct {
		name        string
		panicValue  any
		path        string
		mustContain []string
	}{
		{
			name:        "string panic",
			panicValue:  "unable to redefine 'p' shorthand in \"dashboard\" flagset",
			path:        "/home/x/.gormes/crash-123.log",
			mustContain: []string{"unable to redefine 'p' shorthand", "/home/x/.gormes/crash-123.log"},
		},
		{
			name:        "error panic",
			panicValue:  errorString("nil pointer dereference"),
			path:        "/tmp/crash-9.log",
			mustContain: []string{"nil pointer dereference", "/tmp/crash-9.log"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CrashStderrMessage(tc.panicValue, tc.path)
			for _, want := range tc.mustContain {
				if !strings.Contains(got, want) {
					t.Errorf("CrashStderrMessage(%v) missing %q\ngot:\n%s", tc.panicValue, want, got)
				}
			}
		})
	}
}

// TestCrashStderrMessageTruncatesLongPanicExcerpt keeps the inline
// excerpt readable: a 50-line panic body would otherwise flood stderr.
// First line plus a hint to read the log file is the right balance.
func TestCrashStderrMessageTruncatesLongPanicExcerpt(t *testing.T) {
	long := strings.Repeat("x", 5000) + "\nsecond line should not appear"
	msg := CrashStderrMessage(long, "/tmp/crash-1.log")
	if strings.Contains(msg, "second line should not appear") {
		t.Errorf("CrashStderrMessage must truncate multi-line panics; got:\n%s", msg)
	}
	if !strings.Contains(msg, "/tmp/crash-1.log") {
		t.Errorf("CrashStderrMessage must still surface the log path; got:\n%s", msg)
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }
