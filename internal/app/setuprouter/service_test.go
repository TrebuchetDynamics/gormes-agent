package setuprouter

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestFallbackRulesFromPrimaryChat(t *testing.T) {
	rules := FallbackRules([]config.RouterRouteCfg{
		{Alias: "primary-chat"},
		{Alias: "fallback-a"},
		{Alias: "fallback-b"},
	})
	if len(rules) != 2 || rules[0].From != "primary-chat" || rules[0].To != "fallback-a" || rules[1].To != "fallback-b" {
		t.Fatalf("rules = %+v", rules)
	}
}

func TestOpenAIBaseURLNormalizesListenAddress(t *testing.T) {
	for _, tt := range []struct {
		listen string
		want   string
	}{
		{listen: "127.0.0.1:8787", want: "http://127.0.0.1:8787/v1"},
		{listen: "https://example.test/router?x=1#frag", want: "https://example.test/router/v1"},
		{listen: "", want: "http://127.0.0.1:9999/v1"},
	} {
		if got := OpenAIBaseURL(tt.listen, "127.0.0.1:9999"); got != tt.want {
			t.Fatalf("OpenAIBaseURL(%q) = %q, want %q", tt.listen, got, tt.want)
		}
	}
}

func TestSlugSanitizesRouteAlias(t *testing.T) {
	if got := Slug(" Open_AI Free/Tier "); got != "open-ai-free-tier" {
		t.Fatalf("Slug = %q", got)
	}
	if got := Slug("!!!"); got != "route" {
		t.Fatalf("empty Slug = %q", got)
	}
}

func TestRouteLabels(t *testing.T) {
	got := RouteLabels(config.RouterRouteCfg{Name: "free-tier", Optional: true})
	want := []string{"requires your provider account/API key; quotas are provider-controlled", "optional; only enabled if already installed and healthy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RouteLabels = %#v, want %#v", got, want)
	}
}
