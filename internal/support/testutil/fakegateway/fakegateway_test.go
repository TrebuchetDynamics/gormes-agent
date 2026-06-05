package fakegateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestFakeGatewayWebhookServerRecordsEvents(t *testing.T) {
	fake := New(t)
	body := bytes.NewBufferString(`{"platform":"telegram","chat_id":"42","text":"hello"}`)
	resp, err := fake.Client.Post(fake.URL+"/webhook", "application/json", body)
	if err != nil {
		t.Fatalf("POST webhook: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST webhook status = %d", resp.StatusCode)
	}
	events := fake.Events()
	if len(events) != 1 || events[0].Platform != "telegram" || events[0].ChatID != "42" || events[0].Text != "hello" {
		t.Fatalf("Events() = %+v, want recorded webhook event", events)
	}
}

func TestFakeGatewayHealthReloadAndLogsContracts(t *testing.T) {
	fake := New(t)
	for _, path := range []string{"/health", "/reload", "/logs"} {
		resp, err := fake.Client.Get(fake.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		var got map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			resp.Body.Close()
			t.Fatalf("decode %s: %v", path, err)
		}
		resp.Body.Close()
		if ok, _ := got["ok"].(bool); !ok {
			t.Fatalf("GET %s = %v, want ok=true", path, got)
		}
	}
}
