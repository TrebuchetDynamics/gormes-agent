package logs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadContentGatewayAndFileFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"entries":[{"time":"2026-05-08T07:00:00Z","level":"info","message":"gateway started"}]}`))
	}))
	defer srv.Close()

	got, err := ReadContent(srv.Client(), srv.URL+"/api/logs", "")
	if err != nil {
		t.Fatalf("ReadContent gateway: %v", err)
	}
	if got.Source != "gateway" || len(got.Entries) != 1 || got.Entries[0].Message != "gateway started" {
		t.Fatalf("gateway content = %+v", got)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "gormes.log")
	if err := os.WriteFile(path, []byte("line one\nline two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = ReadContent(NewHTTPClient(10*time.Millisecond), "http://127.0.0.1:1/dead", path)
	if err != nil {
		t.Fatalf("ReadContent fallback: %v", err)
	}
	if got.Source != "file" || got.Path != path || !strings.Contains(got.Content, "line two") {
		t.Fatalf("fallback content = %+v", got)
	}
}

func TestFormatSplitAndTail(t *testing.T) {
	formatted := FormatEntries([]Entry{{Time: "t", Level: "info", Message: "hello"}})
	if len(formatted) != 1 || formatted[0] != "[t] info: hello" {
		t.Fatalf("FormatEntries = %v", formatted)
	}
	lines := SplitLines("a\nb\nc\n")
	if strings.Join(lines, ",") != "a,b,c" {
		t.Fatalf("SplitLines = %v", lines)
	}
	tail := TailLines(lines, 2)
	if strings.Join(tail, ",") != "b,c" {
		t.Fatalf("TailLines = %v", tail)
	}
	if got := ReadTail(Content{Source: "file", Content: "a\nb\nc\n"}, 2); got != "b\nc" {
		t.Fatalf("ReadTail = %q, want b\\nc", got)
	}
}
