package config

import "strings"

// WriteRouterConfig replaces the [router] section in path while preserving the
// rest of the operator config document. Router setup uses this instead of
// single-key writes because routes and fallback rules are array-of-table data.
func WriteRouterConfig(path string, router RouterCfg) error {
	doc, err := readTOMLDoc(path)
	if err != nil {
		return err
	}
	doc["router"] = routerConfigDocument(router)
	return writeTOMLDoc(path, doc)
}

func routerConfigDocument(router RouterCfg) map[string]any {
	out := map[string]any{
		"enabled":     router.Enabled,
		"redact_logs": router.RedactLogs,
	}
	if strings.TrimSpace(router.Listen) != "" {
		out["listen"] = strings.TrimSpace(router.Listen)
	}
	if keys := cleanRouterStrings(router.APIKeys); len(keys) > 0 {
		out["api_keys"] = keys
	}
	if strings.TrimSpace(router.APIKeyEnv) != "" {
		out["api_key_env"] = strings.TrimSpace(router.APIKeyEnv)
	}
	if strings.TrimSpace(router.SetupMode) != "" {
		out["setup_mode"] = strings.TrimSpace(router.SetupMode)
	}
	if routes := routerRouteDocuments(router.Routes); len(routes) > 0 {
		out["routes"] = routes
	}
	if fallback := routerFallbackDocuments(router.Fallback); len(fallback) > 0 {
		out["fallback"] = fallback
	}
	return out
}

func routerRouteDocuments(routes []RouterRouteCfg) []map[string]any {
	out := make([]map[string]any, 0, len(routes))
	for _, route := range routes {
		item := map[string]any{}
		if strings.TrimSpace(route.Name) != "" {
			item["name"] = strings.TrimSpace(route.Name)
		}
		if strings.TrimSpace(route.Provider) != "" {
			item["provider"] = strings.TrimSpace(route.Provider)
		}
		if strings.TrimSpace(route.Model) != "" {
			item["model"] = strings.TrimSpace(route.Model)
		}
		if strings.TrimSpace(route.Alias) != "" {
			item["alias"] = strings.TrimSpace(route.Alias)
		}
		if strings.TrimSpace(route.BaseURL) != "" {
			item["base_url"] = strings.TrimSpace(route.BaseURL)
		}
		if strings.TrimSpace(route.APIKeyEnv) != "" {
			item["api_key_env"] = strings.TrimSpace(route.APIKeyEnv)
		}
		if route.APIKeyRef != nil {
			item["api_key_ref"] = route.APIKeyRef
		}
		if strings.TrimSpace(route.Transport) != "" {
			item["transport"] = strings.TrimSpace(route.Transport)
		}
		if route.Optional {
			item["optional"] = true
		}
		if route.Weight != 0 {
			item["weight"] = int64(route.Weight)
		}
		if len(item) > 0 {
			out = append(out, item)
		}
	}
	return out
}

func routerFallbackDocuments(rules []RouterFallbackCfg) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		item := map[string]any{}
		if strings.TrimSpace(rule.From) != "" {
			item["from"] = strings.TrimSpace(rule.From)
		}
		if strings.TrimSpace(rule.To) != "" {
			item["to"] = strings.TrimSpace(rule.To)
		}
		if on := cleanRouterStrings(rule.On); len(on) > 0 {
			item["on"] = on
		}
		if len(item) > 0 {
			out = append(out, item)
		}
	}
	return out
}

func cleanRouterStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
