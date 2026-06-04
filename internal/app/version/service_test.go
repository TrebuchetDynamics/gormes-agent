package version

import (
	"bytes"
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"
)

func TestRunHumanFormatMarksDirtyBuild(t *testing.T) {
	var out bytes.Buffer
	if err := Run(&out, Info{Version: "1.2.3", GitDirty: "true"}, false); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, want := strings.TrimSpace(out.String()), "gormes 1.2.3 (dirty)"; got != want {
		t.Fatalf("human output = %q, want %q", got, want)
	}
}

func TestRunJSONIncludesBinaryIdentity(t *testing.T) {
	var out bytes.Buffer
	info := Info{Version: "1.2.3", DateAlias: "v2026.1.2", GitCommit: "abcdef123", GitDirty: "false", BuildDate: "2026-01-02T03:04:05Z"}
	if err := Run(&out, info, true); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var got ReportJSON
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got.Version != info.Version || got.DateAlias != info.DateAlias || got.GitCommit != info.GitCommit || got.BuildDate != info.BuildDate {
		t.Fatalf("report = %+v, want identity from %+v", got, info)
	}
	if got.GoVersion == "" || got.OS == "" || got.Arch == "" {
		t.Fatalf("platform fields missing: %+v", got)
	}
}

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
