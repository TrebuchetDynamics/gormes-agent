package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestProviderCredentialResolutionDrivesReadModelAndUpstreamAuthOrdering(t *testing.T) {
	const routeSecret = "route-env-secret"
	const providerSecret = "provider-manifest-secret"
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeRouterJSON(w, http.StatusOK, map[string]any{"object": "list", "data": []any{}})
	}))
	defer upstream.Close()

	lookup := mapLookup(map[string]string{
		"ROUTER_OPENROUTER_KEY": routeSecret,
		"OPENROUTER_API_KEY":    providerSecret,
	})
	cfg := config.Config{Router: config.RouterCfg{
		Enabled: true,
		Routes: []config.RouterRouteCfg{{
			Name:      "route-owned-openrouter",
			Alias:     "route-chat",
			Provider:  "openrouter",
			Model:     "openrouter/auto",
			BaseURL:   upstream.URL + "/v1",
			APIKeyEnv: "ROUTER_OPENROUTER_KEY",
		}},
	}}

	model := BuildReadModel(cfg, Options{LookupEnv: lookup, SkipPrimary: true})
	if len(model.Routes) != 1 {
		t.Fatalf("routes = %d, want 1: %+v", len(model.Routes), model.Routes)
	}
	route := model.Routes[0]
	if route.CredentialStatus != CredentialConfigured || route.Status != RouteStatusConfigured {
		t.Fatalf("route status = %s credential=%s evidence=%v, want configured via route env", route.Status, route.CredentialStatus, route.Evidence)
	}

	provider := NewHTTPUpstreamProvider(HTTPUpstreamProviderOptions{LookupEnv: lookup, Client: upstream.Client()})
	probe := provider.Probe(context.Background(), route)
	if !probe.Available {
		t.Fatalf("probe = %+v, want available", probe)
	}
	if gotAuth != "Bearer "+routeSecret {
		t.Fatalf("upstream Authorization = %q, want route env secret before provider manifest fallback", gotAuth)
	}
}

func TestProviderCredentialResolutionLoadsFileSecretRefForReadModelAndRuntimeAuth(t *testing.T) {
	const fileSecret = "file-backed-route-secret"
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeRouterJSON(w, http.StatusOK, map[string]any{"object": "list", "data": []any{}})
	}))
	defer upstream.Close()

	secretFile := t.TempDir() + "/secrets.json"
	if err := os.WriteFile(secretFile, []byte(`{"router":{"openrouter":"`+fileSecret+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Secrets: config.SecretsCfg{Providers: map[string]config.SecretProviderCfg{
			"router-file": {Source: config.SecretRefSourceFile, Path: secretFile},
		}},
		Router: config.RouterCfg{
			Enabled: true,
			Routes: []config.RouterRouteCfg{{
				Name:      "route-owned-openrouter",
				Alias:     "route-chat",
				Provider:  "openrouter",
				Model:     "openrouter/auto",
				BaseURL:   upstream.URL + "/v1",
				APIKeyRef: &config.SecretRef{Source: config.SecretRefSourceFile, Provider: "router-file", ID: "/router/openrouter"},
			}},
		},
	}

	model := BuildReadModel(cfg, Options{SkipPrimary: true})
	if len(model.Routes) != 1 {
		t.Fatalf("routes = %d, want 1: %+v", len(model.Routes), model.Routes)
	}
	route := model.Routes[0]
	if route.CredentialStatus != CredentialConfigured || route.Status != RouteStatusConfigured {
		t.Fatalf("route status = %s credential=%s evidence=%v, want configured via file SecretRef", route.Status, route.CredentialStatus, route.Evidence)
	}

	provider := NewHTTPUpstreamProvider(HTTPUpstreamProviderOptions{Client: upstream.Client()})
	probe := provider.Probe(context.Background(), route)
	if !probe.Available {
		t.Fatalf("probe = %+v, want available", probe)
	}
	if gotAuth != "Bearer "+fileSecret {
		t.Fatalf("upstream Authorization = %q, want file SecretRef secret", gotAuth)
	}
}

func TestResolveProviderCredentialPreservesNotNeededLocalRoute(t *testing.T) {
	resolution := resolveProviderCredential(Route{Provider: "custom", Local: true, Optional: true}, nil)
	if resolution.Status != CredentialNotNeeded || !resolution.Available || resolution.Value != "" {
		t.Fatalf("resolution = %+v, want available not-required without secret material", resolution)
	}
}
