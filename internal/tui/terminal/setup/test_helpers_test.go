package setup

import (
	"os"
	"testing"
)

type fakeTerminalFileOps struct {
	readErr error
	body    []byte
	writes  []string
	copies  [][2]string
	mkdirs  []string
}

func (f *fakeTerminalFileOps) ops() TerminalSetupFileOps {
	return TerminalSetupFileOps{
		MkdirAll: func(path string, _ os.FileMode) error {
			f.mkdirs = append(f.mkdirs, path)
			return nil
		},
		ReadFile: func(string) ([]byte, error) {
			if f.readErr != nil {
				return nil, f.readErr
			}
			return append([]byte(nil), f.body...), nil
		},
		WriteFile: func(_ string, body []byte, _ os.FileMode) error {
			f.writes = append(f.writes, string(body))
			return nil
		},
		CopyFile: func(src, dst string) error {
			f.copies = append(f.copies, [2]string{src, dst})
			return nil
		},
	}
}

func defaultCompleteKeybindingsJSON(t *testing.T) string {
	t.Helper()
	fake := &fakeTerminalFileOps{readErr: os.ErrNotExist}
	result := ConfigureTerminalKeybindings("vscode", TerminalSetupOptions{
		HomeDir:  "/Users/me",
		Platform: "darwin",
		FileOps:  fake.ops(),
	})
	if !result.Success || len(fake.writes) != 1 {
		t.Fatalf("build complete bindings result=%+v writes=%d, want one successful write", result, len(fake.writes))
	}
	return fake.writes[0]
}

func sameStringSet(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]bool, len(got))
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			return false
		}
	}
	return true
}
