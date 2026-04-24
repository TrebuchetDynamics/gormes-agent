package tuigateway

import (
	"os"
	"runtime"
	"testing"
)

func TestNormalizeCompletionPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("completion path normalization bakes in non-Windows behavior; upstream short-circuits on nt")
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Fatalf("UserHomeDir: err=%v home=%q", err, home)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"tilde only", "~", home},
		{"tilde slash", "~/", home + "/"},
		{"tilde prefix", "~/Documents", home + "/Documents"},
		{"tilde nested", "~/src/gormes/README.md", home + "/src/gormes/README.md"},
		{"plain relative", "internal/tuigateway", "internal/tuigateway"},
		{"plain absolute", "/etc/hosts", "/etc/hosts"},
		{"backslash to slash", "foo\\bar\\baz", "foo/bar/baz"},
		{"mixed slashes", "dir\\sub/file", "dir/sub/file"},
		{"windows drive forward slash", "C:/Users/xel", "/mnt/c/Users/xel"},
		{"windows drive backslash", "C:\\Users\\xel\\docs", "/mnt/c/Users/xel/docs"},
		{"windows drive lowercase", "d:/projects/gormes", "/mnt/d/projects/gormes"},
		{"windows drive rooted empty tail", "E:/", "/mnt/e/"},
		{"non-letter drive-like", "1:/foo/bar", "1:/foo/bar"},
		{"colon without slash", "C:Users", "C:Users"},
		{"too short drive", "C:", "C:"},
		{"tilde unknown user unchanged", "~ghostuser", "~ghostuser"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeCompletionPath(tc.in)
			if got != tc.want {
				t.Fatalf("NormalizeCompletionPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeCompletionPath_TildeExpansionRespectsHomeEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows only")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if got, want := NormalizeCompletionPath("~"), tmp; got != want {
		t.Fatalf("~ expansion: got %q want %q", got, want)
	}
	if got, want := NormalizeCompletionPath("~/inner"), tmp+"/inner"; got != want {
		t.Fatalf("~/inner expansion: got %q want %q", got, want)
	}
}
