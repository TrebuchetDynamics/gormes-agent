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
	tail   int
	buffer []byte
	total  int64
	closed bool
}

// NewBoundedSink returns a Sink that buffers at most tailBytes of stderr in
// memory. With path == "" the sink runs in discard mode: Write reports success
// without retaining bytes and Close never creates a file.
func NewBoundedSink(path string, tailBytes int) Sink {
	return &boundedSink{path: path, tail: tailBytes}
}

func (s *boundedSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(p)
	s.total += int64(n)
	if s.path == "" || s.tail <= 0 {
		return n, nil
	}
	s.buffer = append(s.buffer, p...)
	if len(s.buffer) > s.tail {
		drop := len(s.buffer) - s.tail
		s.buffer = append(s.buffer[:0], s.buffer[drop:]...)
	}
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
	if int64(len(s.buffer)) < s.total {
		dropped := s.total - int64(len(s.buffer))
		if _, err := fmt.Fprintf(f, "[truncated %d bytes]\n", dropped); err != nil {
			return err
		}
	}
	if _, err := f.Write(s.buffer); err != nil {
		return err
	}
	return nil
}
