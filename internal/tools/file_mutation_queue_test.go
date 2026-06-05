package tools

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestPatchToolUsesMutationQueueForFullReadModifyWriteWindow(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha beta"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), MutationQueue: NewFileMutationQueue(), TaskID: "agent-a"}
	readOut := executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	if readOut["path"] != "notes.txt" {
		t.Fatalf("read output = %#v", readOut)
	}

	origRead := fileTaskReadFile
	firstReadEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondReadWhileBlocked := make(chan struct{})
	var mu sync.Mutex
	firstRead := false
	released := false
	fileTaskReadFile = func(name string) ([]byte, error) {
		if filepath.Clean(name) == filepath.Clean(path) {
			mu.Lock()
			if !firstRead {
				firstRead = true
				close(firstReadEntered)
				mu.Unlock()
				<-releaseFirst
				mu.Lock()
				released = true
				mu.Unlock()
				return origRead(name)
			}
			if !released {
				select {
				case <-secondReadWhileBlocked:
				default:
					close(secondReadWhileBlocked)
				}
			}
			mu.Unlock()
		}
		return origRead(name)
	}
	defer func() { fileTaskReadFile = origRead }()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		out := executePatchTool(t, NewPatchTool(cfg), `{"path":"notes.txt","old_string":"alpha","new_string":"ALPHA"}`)
		if out["status"] != "ok" {
			t.Errorf("first patch = %#v, want ok", out)
		}
	}()
	<-firstReadEntered
	go func() {
		defer wg.Done()
		out := executePatchTool(t, NewPatchTool(cfg), `{"path":"notes.txt","old_string":"beta","new_string":"BETA"}`)
		if out["status"] != "ok" {
			t.Errorf("second patch = %#v, want ok", out)
		}
	}()
	select {
	case <-secondReadWhileBlocked:
		t.Fatal("second patch read the file while the first patch still held the mutation window")
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseFirst)
	wg.Wait()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if string(raw) != "ALPHA BETA" {
		t.Fatalf("patched content = %q, want both changes preserved", raw)
	}
}
