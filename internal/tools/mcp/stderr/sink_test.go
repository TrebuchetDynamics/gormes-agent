package stderr

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBoundedSinkWriteAfterCloseReturnsErrorAndDoesNotMutateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stderr.log")
	sink := NewBoundedSink(path, 8)

	if _, err := sink.Write([]byte("before")); err != nil {
		t.Fatalf("Write before close: %v", err)
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	n, err := sink.Write([]byte("after"))
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Write after close err = %v, want os.ErrClosed", err)
	}
	if n != 0 {
		t.Fatalf("Write after close n = %d, want 0", n)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(contents, []byte("before")) {
		t.Fatalf("contents after post-close write = %q, want %q", contents, "before")
	}
}

func TestTailBufferKeepsLastBytesAndCountsDroppedBytes(t *testing.T) {
	buf := tailBuffer{limit: 5}

	buf.append([]byte("abc"))
	buf.append([]byte("defgh"))

	if got := string(buf.bytes()); got != "defgh" {
		t.Fatalf("tail bytes = %q, want %q", got, "defgh")
	}
	if got := buf.dropped(); got != 3 {
		t.Fatalf("dropped bytes = %d, want 3", got)
	}
}
