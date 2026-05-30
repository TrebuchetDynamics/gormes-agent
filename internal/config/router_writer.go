package config

import routerconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/router"

// WriteRouterConfig replaces the [router] section in path while preserving the
// rest of the operator config document. Router setup uses this instead of
// single-key writes because routes and fallback rules are array-of-table data.
func WriteRouterConfig(path string, router RouterCfg) error {
	doc, err := readTOMLDoc(path)
	if err != nil {
		return err
	}
	doc["router"] = routerconfig.Document(router)
	return writeTOMLDoc(path, doc)
}
