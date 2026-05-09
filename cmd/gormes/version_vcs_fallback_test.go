package main

import (
	"runtime/debug"
	"testing"
)

// TestResolveGitCommitFrom_FavorsInjectedValue: when ldflags injected a
// concrete commit (e.g. release CI or `make build`), prefer it. The
// runtime/debug VCS fallback is only for binaries built with plain
// `go build` that skipped -ldflags injection.
func TestResolveGitCommitFrom_FavorsInjectedValue(t *testing.T) {
	got := resolveGitCommitFrom("a07b1d50c", []debug.BuildSetting{
		{Key: "vcs.revision", Value: "deadbeefdeadbeef"},
	})
	if got != "a07b1d50c" {
		t.Errorf("injected commit must win over vcs.revision; got %q", got)
	}
}

// TestResolveGitCommitFrom_FallsBackToVCSRevision: when ldflags weren't
// injected (developers running plain `go build`), Go 1.18+ embeds the
// commit sha in BuildInfo.Settings under vcs.revision. Surface that so
// `gormes doctor` and `onboard --json` show the real commit instead of
// the "unknown" sentinel.
func TestResolveGitCommitFrom_FallsBackToVCSRevision(t *testing.T) {
	cases := []struct {
		name     string
		injected string
	}{
		{"empty injected", ""},
		{"unknown sentinel", "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveGitCommitFrom(tc.injected, []debug.BuildSetting{
				{Key: "vcs.revision", Value: "1234567890abcdef1234567890abcdef12345678"},
			})
			// Truncate to short-sha length to match the existing
			// 9-char convention from `git rev-parse --short` used in
			// install.sh + doctor output.
			want := "123456789"
			if got != want {
				t.Errorf("vcs.revision fallback should yield short sha %q; got %q", want, got)
			}
		})
	}
}

// TestResolveGitCommitFrom_NoVCSInfo: when neither ldflags nor build-info
// VCS settings are available (e.g. `go run` invocations), fall back to
// the "unknown" sentinel so consumers know the commit is unidentifiable
// instead of seeing an empty string.
func TestResolveGitCommitFrom_NoVCSInfo(t *testing.T) {
	got := resolveGitCommitFrom("", nil)
	if got != "unknown" {
		t.Errorf("missing-vcs fallback should return %q; got %q", "unknown", got)
	}
	got = resolveGitCommitFrom("unknown", []debug.BuildSetting{
		{Key: "vcs.time", Value: "2026-01-01T00:00:00Z"},
	})
	if got != "unknown" {
		t.Errorf("missing-vcs.revision fallback should return %q; got %q", "unknown", got)
	}
}

// TestResolveGitCommitFrom_ShortShaPassesThrough: short shas (already
// 9 chars or fewer) flow through unchanged. Guards against a trim-by-
// length bug that would corrupt them.
func TestResolveGitCommitFrom_ShortShaPassesThrough(t *testing.T) {
	got := resolveGitCommitFrom("", []debug.BuildSetting{
		{Key: "vcs.revision", Value: "abc1234"},
	})
	if got != "abc1234" {
		t.Errorf("short sha must pass through unchanged; got %q", got)
	}
}

// TestResolveGitDirtyFrom_FavorsInjectedValue: when ldflags injected
// GitDirty=true, trust it (release pipeline source-of-truth).
func TestResolveGitDirtyFrom_FavorsInjectedValue(t *testing.T) {
	if !resolveGitDirtyFrom("true", nil) {
		t.Error("injected GitDirty=true must yield dirty=true")
	}
}

// TestResolveGitDirtyFrom_FallsBackToVCSModified: when GitDirty wasn't
// explicitly set, Go 1.18+ embeds vcs.modified=true|false in
// BuildInfo.Settings; use it.
func TestResolveGitDirtyFrom_FallsBackToVCSModified(t *testing.T) {
	if !resolveGitDirtyFrom("false", []debug.BuildSetting{
		{Key: "vcs.modified", Value: "true"},
	}) {
		t.Error("vcs.modified=true should yield dirty=true when injected GitDirty=false")
	}
	if resolveGitDirtyFrom("false", []debug.BuildSetting{
		{Key: "vcs.modified", Value: "false"},
	}) {
		t.Error("vcs.modified=false should yield dirty=false")
	}
}

func TestResolveBuildDateFrom_FavorsInjectedValue(t *testing.T) {
	got := resolveBuildDateFrom("2026-05-09T12:34:56Z", []debug.BuildSetting{
		{Key: "vcs.time", Value: "2026-01-01T00:00:00Z"},
	})
	if got != "2026-05-09T12:34:56Z" {
		t.Errorf("injected build date must win over vcs.time; got %q", got)
	}
}

func TestResolveBuildDateFrom_FallsBackToVCSTime(t *testing.T) {
	cases := []struct {
		name     string
		injected string
	}{
		{"empty injected", ""},
		{"unknown sentinel", "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBuildDateFrom(tc.injected, []debug.BuildSetting{
				{Key: "vcs.time", Value: "2026-05-08T19:20:21Z"},
			})
			if got != "2026-05-08T19:20:21Z" {
				t.Errorf("vcs.time fallback = %q, want %q", got, "2026-05-08T19:20:21Z")
			}
		})
	}
}

func TestResolveBuildDateFrom_NoVCSInfo(t *testing.T) {
	got := resolveBuildDateFrom("", nil)
	if got != "unknown" {
		t.Errorf("missing build-date fallback should return %q; got %q", "unknown", got)
	}
	got = resolveBuildDateFrom("unknown", []debug.BuildSetting{
		{Key: "vcs.revision", Value: "1234567890abcdef"},
	})
	if got != "unknown" {
		t.Errorf("missing vcs.time fallback should return %q; got %q", "unknown", got)
	}
}
