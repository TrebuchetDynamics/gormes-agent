package gormescli

import (
	"testing"
)

func TestSanitizeTermuxExecArgsWithExe(t *testing.T) {
	exe := "/data/user/0/com.termux/files/usr/bin/gormes"

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "no args",
			args: []string{},
			want: []string{},
		},
		{
			name: "normal flag only",
			args: []string{"--version"},
			want: []string{"--version"},
		},
		{
			name: "normal subcommand",
			args: []string{"version"},
			want: []string{"version"},
		},
		{
			name: "injected path at args[0]",
			args: []string{exe, "--version"},
			want: []string{"--version"},
		},
		{
			name: "injected path at args[1]",
			args: []string{"gormes", exe, "--version"},
			want: []string{"gormes", "--version"},
		},
		{
			name: "injected path with subcommand",
			args: []string{exe, "version"},
			want: []string{"version"},
		},
		{
			name: "data-data injected path matches data-user executable alias",
			args: []string{"/data/data/com.termux/files/usr/bin/gormes", "version"},
			want: []string{"version"},
		},
		{
			name: "data-data injected path after command matches data-user executable alias",
			args: []string{"version", "/data/data/com.termux/files/usr/bin/gormes"},
			want: []string{"version"},
		},
		{
			name: "other app data-data path is not an executable alias",
			args: []string{"/data/data/com.example/files/usr/bin/gormes", "version"},
			want: []string{"/data/data/com.example/files/usr/bin/gormes", "version"},
		},
		{
			name: "multiple flags after injection",
			args: []string{exe, "doctor", "--offline", "--json"},
			want: []string{"doctor", "--offline", "--json"},
		},
		{
			name: "no injection when path differs",
			args: []string{"/some/other/path", "--version"},
			want: []string{"/some/other/path", "--version"},
		},
		{
			name: "nil args",
			args: nil,
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeTermuxExecArgsWithExe(tc.args, exe)
			if len(got) != len(tc.want) {
				t.Fatalf("len(got)=%d, want %d; got=%v, want=%v", len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got[%d]=%q, want %q; got=%v, want=%v", i, got[i], tc.want[i], got, tc.want)
				}
			}
		})
	}
}

func TestSanitizeTermuxExecArgsWithExeEmptyExe(t *testing.T) {
	got := SanitizeTermuxExecArgsWithExe([]string{"/data/data/com.termux/files/usr/bin/gormes", "--version"}, "")
	want := []string{"/data/data/com.termux/files/usr/bin/gormes", "--version"}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d, want %d; got=%v, want=%v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}
