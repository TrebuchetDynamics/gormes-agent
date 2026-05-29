package doctor

import (
	"os/exec"
	"strings"
	"testing"
)

func TestTermuxStoragePathIssues(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want int
	}{
		{
			name: "no issues",
			env:  map[string]string{"GORMES_HOME": "/data/data/com.termux/files/home/.gormes", "HOME": "/data/data/com.termux/files/home"},
			want: 0,
		},
		{
			name: "sdcard GORMES_HOME",
			env:  map[string]string{"GORMES_HOME": "/sdcard/gormes", "HOME": "/data/data/com.termux/files/home"},
			want: 1,
		},
		{
			name: "storage HOME",
			env:  map[string]string{"HOME": "/storage/emulated/0"},
			want: 1,
		},
		{
			name: "multiple issues",
			env:  map[string]string{"GORMES_HOME": "/sdcard/gormes", "HOME": "/storage/emulated/0", "TMPDIR": "/sdcard/tmp"},
			want: 3,
		},
		{
			name: "mixed case sdcard",
			env:  map[string]string{"GORMES_HOME": "/SDCARD/gormes"},
			want: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := termuxStoragePathIssues(tc.env)
			if len(got) != tc.want {
				t.Fatalf("len(issues)=%d, want %d; got=%v", len(got), tc.want, got)
			}
		})
	}
}

func TestCheckTermuxRuntime_StoragePaths(t *testing.T) {
	result := CheckTermuxRuntime(TermuxRuntimeOptions{
		Env: map[string]string{
			"TERMUX_VERSION": "0.119.0",
			"PREFIX":         "/data/data/com.termux/files/usr",
			"HOME":           "/sdcard",
			"PATH":           "/data/data/com.termux/files/usr/bin",
		},
		LookPath: func(string) (string, error) { return "", exec.ErrNotFound },
	})

	var found bool
	for _, item := range result.Items {
		if item.Name == "storage_paths" {
			found = true
			if item.Status != StatusWarn {
				t.Fatalf("storage_paths status = %q, want %q", item.Status, StatusWarn)
			}
			if !strings.Contains(item.Note, "external storage") {
				t.Fatalf("storage_paths note missing 'external storage': %q", item.Note)
			}
		}
	}
	if !found {
		t.Fatal("storage_paths check missing from result")
	}
}
