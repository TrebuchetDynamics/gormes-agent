package stderr

import (
	"fmt"
	"os"
	"sync"
)

// Sink is the minimal Writer/Closer surface the MCP transports use to drain a
// server subprocess's stderr without growing unbounded. Implementations must be
// safe for concurrent Write calls.
type Sink interface {
	Write(p []byte) (int, error)
	Close() error
}

// boundedSink keeps only the last `tail` bytes of stderr in memory and, on
// Close, flushes them to `path` (prepending a `[truncated <N> bytes]` marker
// when bytes were dropped). When path is empty the sink discards all writes and
// never touches the filesystem.
type boundedSink struct {
	mu     sync.Mutex
	path   string
	buffer tailBuffer
	closed bool
}

type tailBuffer struct {
	limit int
	buf   []byte
	total int64
}

func (b *tailBuffer) append(p []byte) {
	b.total += int64(len(p))
	if b.limit <= 0 {
		return
	}
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.limit {
		drop := len(b.buf) - b.limit
		b.buf = append(b.buf[:0], b.buf[drop:]...)
	}
}

type tailSnapshot struct {
	Bytes   []byte
	Dropped int64
}

func (b *tailBuffer) snapshot() tailSnapshot {
	return tailSnapshot{
		Bytes:   append([]byte(nil), b.buf...),
		Dropped: b.total - int64(len(b.buf)),
	}
}

func truncationMarker(dropped int64) string {
	return fmt.Sprintf("[truncated %d bytes]\n", dropped)
}

// NewBoundedSink returns a Sink that buffers at most tailBytes of stderr in
// memory. With path == "" the sink runs in discard mode: Write reports success
// without retaining bytes and Close never creates a file.
func NewBoundedSink(path string, tailBytes int) Sink {
	return &boundedSink{path: path, buffer: tailBuffer{limit: tailBytes}}
}

func (s *boundedSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, os.ErrClosed
	}
	n := len(p)
	if s.path == "" {
		return n, nil
	}
	s.buffer.append(p)
	return n, nil
}

func (s *boundedSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.path == "" {
		return nil
	}
	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()
	snapshot := s.buffer.snapshot()
	if snapshot.Dropped > 0 {
		if _, err := f.WriteString(truncationMarker(snapshot.Dropped)); err != nil {
			return err
		}
	}
	if _, err := f.Write(snapshot.Bytes); err != nil {
		return err
	}
	return nil
}
