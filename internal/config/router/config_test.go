package router

import "testing"

func TestDocumentTrimsAndOmitsEmptyRouterFields(t *testing.T) {
	doc := Document(Config{
		Enabled:    true,
		Listen:     " 127.0.0.1:8787 ",
		APIKeys:    []string{"", " key-a ", "key-b"},
		APIKeyEnv:  " GORMES_ROUTER_KEY ",
		RedactLogs: true,
		SetupMode:  " local ",
		Routes: []RouteConfig{{
			Name:      " primary ",
			Provider:  " openai ",
			Model:     " gpt-4.1 ",
			Alias:     " fast ",
			BaseURL:   " https://example.test/v1 ",
			APIKeyEnv: " OPENAI_API_KEY ",
			Transport: " responses ",
			Optional:  true,
			Weight:    2,
		}},
		Fallback: []Fallback{{From: " gpt-4.1 ", To: " gpt-4.1-mini ", On: []string{"", " 429 "}}},
	})

	if got := doc["listen"]; got != "127.0.0.1:8787" {
		t.Fatalf("listen = %v", got)
	}
	if got := doc["api_keys"].([]string); len(got) != 2 || got[0] != "key-a" || got[1] != "key-b" {
		t.Fatalf("api_keys = %#v", got)
	}
	routes := doc["routes"].([]map[string]any)
	if got := routes[0]["weight"]; got != int64(2) {
		t.Fatalf("route weight = %#v", got)
	}
	fallback := doc["fallback"].([]map[string]any)
	if got := fallback[0]["on"].([]string); len(got) != 1 || got[0] != "429" {
		t.Fatalf("fallback on = %#v", got)
	}
}
