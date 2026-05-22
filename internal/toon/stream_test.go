package toon

import (
	"errors"
	"io"
	"testing"
)

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func TestEncodeJSONToReportsShortWrite(t *testing.T) {
	err := EncodeJSONTo(shortWriter{}, []byte(`{"name":"Ada"}`))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("EncodeJSONTo error = %v, want %v", err, io.ErrShortWrite)
	}
}
