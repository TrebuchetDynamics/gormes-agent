package version

import (
	"runtime/debug"
	"testing"
)

func TestResolveGitCommitFrom(t *testing.T) {
	if got := ResolveGitCommitFrom("a07b1d50c", []debug.BuildSetting{{Key: "vcs.revision", Value: "deadbeefdeadbeef"}}); got != "a07b1d50c" {
		t.Fatalf("injected commit = %q", got)
	}
	if got := ResolveGitCommitFrom("unknown", []debug.BuildSetting{{Key: "vcs.revision", Value: "1234567890abcdef"}}); got != "123456789" {
		t.Fatalf("vcs commit = %q", got)
	}
	if got := ResolveGitCommitFrom("", nil); got != "unknown" {
		t.Fatalf("missing commit = %q", got)
	}
}

func TestResolveGitDirtyFrom(t *testing.T) {
	if !ResolveGitDirtyFrom("yes", nil) {
		t.Fatal("injected yes should be dirty")
	}
	if !ResolveGitDirtyFrom("false", []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}}) {
		t.Fatal("vcs.modified=true should be dirty")
	}
	if ResolveGitDirtyFrom("false", []debug.BuildSetting{{Key: "vcs.modified", Value: "false"}}) {
		t.Fatal("vcs.modified=false should be clean")
	}
}

func TestResolveBuildDateFrom(t *testing.T) {
	if got := ResolveBuildDateFrom("2026-05-09T12:34:56Z", []debug.BuildSetting{{Key: "vcs.time", Value: "2026-01-01T00:00:00Z"}}); got != "2026-05-09T12:34:56Z" {
		t.Fatalf("injected build date = %q", got)
	}
	if got := ResolveBuildDateFrom("unknown", []debug.BuildSetting{{Key: "vcs.time", Value: "2026-05-08T19:20:21Z"}}); got != "2026-05-08T19:20:21Z" {
		t.Fatalf("vcs build date = %q", got)
	}
	if got := ResolveBuildDateFrom("", nil); got != "unknown" {
		t.Fatalf("missing build date = %q", got)
	}
}
