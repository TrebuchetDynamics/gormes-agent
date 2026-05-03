package main

import (
	"os"
	"testing"
)

func TestStdinIsTerminalRejectsDevNull(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	if stdinIsTerminal(devNull) {
		t.Fatalf("%s should not be treated as an interactive terminal", os.DevNull)
	}
}

func TestStdinIsTerminalRejectsPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer reader.Close()
	defer writer.Close()

	if stdinIsTerminal(reader) {
		t.Fatal("pipe reader should not be treated as an interactive terminal")
	}
}

func TestStdinIsTerminalRejectsNil(t *testing.T) {
	if stdinIsTerminal(nil) {
		t.Fatal("nil file should not be treated as an interactive terminal")
	}
}
