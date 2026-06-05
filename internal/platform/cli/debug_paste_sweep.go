// Package cli contains operator-facing command helpers for the gormes binary.
package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/diagnostics"

// PasteEntry represents one pending paste deletion record.
type PasteEntry = diagnostics.PasteEntry

// PasteDeleter defines the interface for issuing a DELETE request to a paste
// service. Implementations must be hermetic — no live network in tests.
type PasteDeleter = diagnostics.PasteDeleter

// PasteStore defines the interface for loading and saving pending paste entries.
// In production this reads/writes a JSON file; in tests it uses an in-memory map.
type PasteStore = diagnostics.PasteStore

// InMemoryPasteStore is a hermetic in-memory implementation of PasteStore for
// testing and offline CLI usage.
type InMemoryPasteStore = diagnostics.InMemoryPasteStore

// PasteSweeper coordinates expired-paste deletion.
type PasteSweeper = diagnostics.PasteSweeper

// SweepResult captures the outcome of one sweep pass.
type SweepResult = diagnostics.SweepResult

// SweepError describes one delete failure with evidence.
type SweepError = diagnostics.SweepError

// PasteURLScheme extracts the paste service name from a URL for validation.
func PasteURLScheme(url string) string { return diagnostics.PasteURLScheme(url) }
