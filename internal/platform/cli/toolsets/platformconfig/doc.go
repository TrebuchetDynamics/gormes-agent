// Package platformconfig owns Hermes-compatible platform_toolsets and mcp_servers
// normalization, runtime resolution, and save-time persistence policy.
//
// It exposes the pure config model and reports needed by CLI/setup callers. It
// must not depend on picker UI metadata, plugin inventories, or command wiring.
package platformconfig
