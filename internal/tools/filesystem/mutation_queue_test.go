package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFileMutationQueueSerializesSameExistingFileFullWindow(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("base"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	queue := NewFileMutationQueue()

	firstEntered := make(chan struct{})
	allowFirst := make(chan struct{})
	secondEntered := make(chan struct{})
	var orderMu sync.Mutex
	var order []string
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := queue.Run(context.Background(), path, func(context.Context) error {
			orderMu.Lock()
			order = append(order, "first-enter")
			orderMu.Unlock()
			close(firstEntered)
			<-allowFirst
			raw, _ := os.ReadFile(path)
			return os.WriteFile(path, append(raw, []byte(" first")...), 0o644)
		}); err != nil {
			t.Errorf("first Run: %v", err)
		}
	}()
	<-firstEntered

	go func() {
		defer wg.Done()
		if err := queue.Run(context.Background(), path, func(context.Context) error {
			orderMu.Lock()
			order = append(order, "second-enter")
			orderMu.Unlock()
			close(secondEntered)
			raw, _ := os.ReadFile(path)
			return os.WriteFile(path, append(raw, []byte(" second")...), 0o644)
		}); err != nil {
			t.Errorf("second Run: %v", err)
		}
	}()

	select {
	case <-secondEntered:
		t.Fatal("same-file second mutation entered before first released; queue did not cover full mutation window")
	case <-time.After(50 * time.Millisecond):
	}
	close(allowFirst)
	wg.Wait()

	if got := strings.Join(order, ","); got != "first-enter,second-enter" {
		t.Fatalf("entry order = %s, want deterministic serial order", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if string(raw) != "base first second" {
		t.Fatalf("file content = %q, want both serialized mutations preserved", raw)
	}
}

func TestFileMutationQueueAllowsDifferentFilesToOverlap(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left.txt")
	right := filepath.Join(root, "right.txt")
	if err := os.WriteFile(left, []byte("left"), 0o644); err != nil {
		t.Fatalf("write left: %v", err)
	}
	if err := os.WriteFile(right, []byte("right"), 0o644); err != nil {
		t.Fatalf("write right: %v", err)
	}
	queue := NewFileMutationQueue()

	leftEntered := make(chan struct{})
	rightEntered := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = queue.Run(context.Background(), left, func(context.Context) error {
			close(leftEntered)
			<-release
			return nil
		})
	}()
	<-leftEntered
	go func() {
		defer wg.Done()
		_ = queue.Run(context.Background(), right, func(context.Context) error {
			close(rightEntered)
			<-release
			return nil
		})
	}()
	select {
	case <-rightEntered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("different-file mutation was blocked behind unrelated file")
	}
	close(release)
	wg.Wait()
}

func TestFileMutationQueueCanonicalizesSymlinkAliasesAndNewFiles(t *testing.T) {
	root := t.TempDir()
	realPath := filepath.Join(root, "real.txt")
	linkPath := filepath.Join(root, "link.txt")
	if err := os.WriteFile(realPath, []byte("real"), 0o644); err != nil {
		t.Fatalf("write real: %v", err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	queue := NewFileMutationQueue()
	realKey, err := queue.Key(realPath)
	if err != nil {
		t.Fatalf("real key: %v", err)
	}
	linkKey, err := queue.Key(linkPath)
	if err != nil {
		t.Fatalf("link key: %v", err)
	}
	if realKey != linkKey {
		t.Fatalf("real key %q != link key %q; existing symlink aliases must share queue key", realKey, linkKey)
	}
	newRel := filepath.Join(root, "missing", "..", "new.txt")
	newKey, err := queue.Key(newRel)
	if err != nil {
		t.Fatalf("new key: %v", err)
	}
	wantNew, err := filepath.Abs(filepath.Join(root, "new.txt"))
	if err != nil {
		t.Fatalf("abs new: %v", err)
	}
	if newKey != filepath.Clean(wantNew) {
		t.Fatalf("new file key = %q, want cleaned absolute path %q", newKey, filepath.Clean(wantNew))
	}
}
