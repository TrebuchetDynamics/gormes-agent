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

func TestTailBufferSnapshotKeepsLastBytesAndCountsDroppedBytes(t *testing.T) {
	buf := tailBuffer{limit: 5}

	buf.append([]byte("abc"))
	buf.append([]byte("defgh"))

	snapshot := buf.snapshot()
	if got := string(snapshot.Bytes); got != "defgh" {
		t.Fatalf("tail bytes = %q, want %q", got, "defgh")
	}
	if snapshot.Dropped != 3 {
		t.Fatalf("dropped bytes = %d, want 3", snapshot.Dropped)
	}
}

func TestTailBufferSnapshotDoesNotExposeMutableBuffer(t *testing.T) {
	buf := tailBuffer{limit: 5}
	buf.append([]byte("abcde"))

	snapshot := buf.snapshot()
	snapshot.Bytes[0] = 'X'

	if got := string(buf.snapshot().Bytes); got != "abcde" {
		t.Fatalf("tail buffer mutated through snapshot = %q, want abcde", got)
	}
}

func TestTailBufferLargeWriteDoesNotRetainDroppedPrefixCapacity(t *testing.T) {
	buf := tailBuffer{limit: 5}
	buf.append(bytes.Repeat([]byte("x"), 1024))

	if got := string(buf.snapshot().Bytes); got != "xxxxx" {
		t.Fatalf("tail bytes = %q, want last five bytes", got)
	}
	if cap(buf.buf) > buf.limit {
		t.Fatalf("tail buffer capacity = %d, want <= limit %d so dropped prefixes are not retained", cap(buf.buf), buf.limit)
	}
}

func TestTruncationMarkerIncludesDroppedByteCount(t *testing.T) {
	if got := truncationMarker(42); got != "[truncated 42 bytes]\n" {
		t.Fatalf("truncation marker = %q", got)
	}
}
