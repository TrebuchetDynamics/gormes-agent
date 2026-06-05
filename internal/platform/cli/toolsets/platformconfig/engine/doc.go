// Package engine owns the behavior implementation for Hermes-compatible
// platform_toolsets and mcp_servers normalization, runtime resolution, and
// save-time persistence policy.
//
// It exposes the pure config model consumed by the parent platformconfig
// compatibility facade. It must not depend on picker UI metadata, plugin
// inventories, or command wiring.
package engine
