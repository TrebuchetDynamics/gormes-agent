package httpjson

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteSetsJSONStatusAndBody(t *testing.T) {
	rec := httptest.NewRecorder()

	Write(rec, http.StatusAccepted, map[string]string{"ok": "yes"})

	if got := rec.Code; got != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", got, http.StatusAccepted)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q", got)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["ok"] != "yes" {
		t.Fatalf("body = %#v", body)
	}
}
