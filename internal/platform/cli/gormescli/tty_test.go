package gormescli

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

	if StdinIsTerminal(devNull) {
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

	if StdinIsTerminal(reader) {
		t.Fatal("pipe reader should not be treated as an interactive terminal")
	}
}

func TestStdinIsTerminalRejectsNil(t *testing.T) {
	if StdinIsTerminal(nil) {
		t.Fatal("nil file should not be treated as an interactive terminal")
	}
}
