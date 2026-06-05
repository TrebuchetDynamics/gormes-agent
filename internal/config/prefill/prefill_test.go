package prefill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResolvesRelativePathFromGormesHome(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "prefill.json"), []byte(`[
		{"role":"user","content":"example request"},
		{"role":"assistant","content":"example answer"}
	]`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	messages, err := Load("prefill.json", home)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "example request" {
		t.Fatalf("messages[0] = %#v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "example answer" {
		t.Fatalf("messages[1] = %#v", messages[1])
	}
}

func TestLoadMissingOrInvalidReturnsEmpty(t *testing.T) {
	home := t.TempDir()
	messages, err := Load("missing.json", home)
	if err != nil {
		t.Fatalf("missing Load error = %v, want nil", err)
	}
	if len(messages) != 0 {
		t.Fatalf("missing messages = %#v, want empty", messages)
	}

	invalid := filepath.Join(home, "invalid.json")
	if err := os.WriteFile(invalid, []byte(`{"role":"user"}`), 0o600); err != nil {
		t.Fatalf("write invalid: %v", err)
	}
	messages, err = Load(invalid, home)
	if err != nil {
		t.Fatalf("invalid Load error = %v, want nil", err)
	}
	if len(messages) != 0 {
		t.Fatalf("invalid messages = %#v, want empty", messages)
	}
}
