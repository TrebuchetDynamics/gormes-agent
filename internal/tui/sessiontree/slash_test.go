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
