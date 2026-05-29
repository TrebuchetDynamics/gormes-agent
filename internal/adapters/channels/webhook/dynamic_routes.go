package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DynamicRoutesFilename = "webhook_subscriptions.json"

// DynamicRouteSet merges static webhook routes with agent-created runtime
// subscriptions stored under the configured Gormes/Hermes-compatible home.
type DynamicRouteSet struct {
	path    string
	static  map[string]RouteConfig
	dynamic map[string]RouteConfig
	mtime   time.Time
}

func NewDynamicRouteSet(homeDir string, static map[string]RouteConfig) *DynamicRouteSet {
	return &DynamicRouteSet{
		path:    filepath.Join(strings.TrimSpace(homeDir), DynamicRoutesFilename),
		static:  cloneRoutes(static),
		dynamic: map[string]RouteConfig{},
	}
}

func (r *DynamicRouteSet) Reload() error {
	if r == nil {
		return errors.New("webhook_dynamic_routes_unavailable: route set is nil")
	}

	info, err := os.Stat(r.path)
	if errors.Is(err, os.ErrNotExist) {
		r.dynamic = map[string]RouteConfig{}
		r.mtime = time.Time{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("webhook_dynamic_routes_unavailable: stat: %w", err)
	}
	if !info.ModTime().After(r.mtime) {
		return nil
	}

	raw, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("webhook_dynamic_routes_unavailable: read: %w", err)
	}
	var parsed map[string]RouteConfig
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("webhook_dynamic_routes_unavailable: parse: %w", err)
	}
	if parsed == nil {
		return errors.New("webhook_dynamic_routes_unavailable: subscriptions must be an object")
	}

	dynamic := make(map[string]RouteConfig, len(parsed))
	for name, route := range parsed {
		name = trim(name)
		if name == "" {
			continue
		}
		if _, exists := r.static[name]; exists {
			continue
		}
		dynamic[name] = cloneRoute(route)
	}
	r.dynamic = dynamic
	r.mtime = info.ModTime()
	return nil
}

func (r *DynamicRouteSet) Route(name string) (RouteConfig, bool) {
	if r == nil {
		return RouteConfig{}, false
	}
	name = trim(name)
	if route, ok := r.static[name]; ok {
		return cloneRoute(route), true
	}
	route, ok := r.dynamic[name]
	return cloneRoute(route), ok
}

func (r *DynamicRouteSet) DynamicCount() int {
	if r == nil {
		return 0
	}
	return len(r.dynamic)
}

func cloneRoutes(in map[string]RouteConfig) map[string]RouteConfig {
	out := make(map[string]RouteConfig, len(in))
	for name, route := range in {
		name = trim(name)
		if name == "" {
			continue
		}
		out[name] = cloneRoute(route)
	}
	return out
}

func cloneRoute(route RouteConfig) RouteConfig {
	route.Events = append([]string(nil), route.Events...)
	if route.DeliverExtra != nil {
		extra := make(map[string]any, len(route.DeliverExtra))
		for key, value := range route.DeliverExtra {
			extra[key] = value
		}
		route.DeliverExtra = extra
	}
	return route
}
