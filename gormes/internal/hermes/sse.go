package hermes

import (
	"bufio"
	"io"
	"strings"
)

// sseFrame is one SSE event (`event:` line + accumulated `data:` lines).
type sseFrame struct {
	event string
	data  string
}

// sseReader is a pull-based SSE parser with a bounded internal buffer
// (1 MB per line — generous for any sane payload; prevents unbounded growth).
type sseReader struct {
	sc *bufio.Scanner
}

func newSSEReader(r io.Reader) *sseReader {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	return &sseReader{sc: sc}
}

// Next returns the next frame. Returns (nil, io.EOF) at end of stream (normal
// close or abrupt disconnect). Returns (nil, err) for other scanner errors.
func (r *sseReader) Next() (*sseFrame, error) {
	var f sseFrame
	for r.sc.Scan() {
		line := r.sc.Text()
		if line == "" {
			if f.data != "" || f.event != "" {
				return &f, nil
			}
			continue
		}
		if strings.HasPrefix(line, ":") { // SSE comment / keepalive
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			f.event = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			if f.data != "" {
				f.data += "\n"
			}
			f.data += strings.TrimPrefix(line, "data: ")
			continue
		}
	}
	if err := r.sc.Err(); err != nil {
		return nil, err
	}
	if f.data != "" || f.event != "" {
		return &f, nil
	}
	return nil, io.EOF
}
