package sessiontree

import (
	"reflect"
	"testing"
)

func TestSlashArgs(t *testing.T) {
	if got := SlashArgs("/tree label sess demo"); !reflect.DeepEqual(got, []string{"label", "sess", "demo"}) {
		t.Fatalf("SlashArgs = %v", got)
	}
	if got := SlashArgs("/tree"); got != nil {
		t.Fatalf("SlashArgs no args = %v, want nil", got)
	}
}

func TestSlashFilterParsing(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{args: nil, want: "default"},
		{args: []string{"--filter", "users"}, want: "user-only"},
		{args: []string{"filter", "all"}, want: "all-equivalent"},
		{args: []string{"--filter=labels"}, want: "labeled-only"},
		{args: []string{"unknown"}, want: "default"},
	}
	for _, tc := range cases {
		if got := ParseSlashFilter(tc.args); got != tc.want {
			t.Fatalf("ParseSlashFilter(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
	for _, arg := range []string{"--filter", "--filter=all", "filter"} {
		if !SlashIsFilter(arg) {
			t.Fatalf("SlashIsFilter(%q) = false", arg)
		}
	}
}

func TestLabelSlashParsingAndStatus(t *testing.T) {
	req, status, ok := ParseLabelRequest([]string{"sess-1", "release", "candidate"}, "set")
	if !ok || status != "" {
		t.Fatalf("ParseLabelRequest ok=%v status=%q, want ok with empty status", ok, status)
	}
	want := LabelRequest{SessionID: "sess-1", Action: "set", Label: "release candidate"}
	if req != want {
		t.Fatalf("ParseLabelRequest = %+v, want %+v", req, want)
	}
	if got := FormatLabelStatus("", req.SessionID, []string{"release", "pinned"}); got != "tree: labels for sess-1: release, pinned" {
		t.Fatalf("FormatLabelStatus fallback = %q", got)
	}
	if _, got, ok := ParseLabelRequest([]string{"sess-1"}, "set"); ok || got != LabelUsage {
		t.Fatalf("ParseLabelRequest missing label ok=%v status=%q, want label usage", ok, got)
	}
}

func TestRestoreSlashParsingAndStatus(t *testing.T) {
	req, status, ok := ParseRestoreRequest([]string{"sess-1", "42"})
	if !ok || status != "" {
		t.Fatalf("ParseRestoreRequest ok=%v status=%q, want ok with empty status", ok, status)
	}
	want := RestoreRequest{SessionID: "sess-1", MessageID: 42}
	if req != want {
		t.Fatalf("ParseRestoreRequest = %+v, want %+v", req, want)
	}
	if got, editable := FormatRestoreStatus(req, RestoreResult{Editable: true}); !editable || got != "tree: restored editable prompt from sess-1#42" {
		t.Fatalf("FormatRestoreStatus editable = %q editable=%v", got, editable)
	}
	if got, editable := FormatRestoreStatus(req, RestoreResult{Editable: false}); editable || got != "tree: replay unavailable: replay_unavailable" {
		t.Fatalf("FormatRestoreStatus fallback = %q editable=%v", got, editable)
	}
	if _, got, ok := ParseRestoreRequest([]string{"sess-1", "nope"}); ok || got != "tree: restore turn_id must be a positive integer" {
		t.Fatalf("ParseRestoreRequest invalid ok=%v status=%q", ok, got)
	}
}
